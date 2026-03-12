package executor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/logs"
)

func RunBuild(deploymentID, repoURL, buildCommand, outputDir string) error {
	log.Println("Starting Docker build...")

	repoURL = strings.TrimSpace(repoURL)
	buildCommand = strings.TrimSpace(buildCommand)
	outputDir = strings.TrimSpace(outputDir)
	if repoURL == "" || strings.ContainsAny(repoURL, "\r\n") {
		return fmt.Errorf("invalid repo_url")
	}
	if buildCommand == "" {
		return fmt.Errorf("invalid build_command")
	}
	if strings.HasPrefix(outputDir, "/") || outputDir == ".." || strings.HasPrefix(outputDir, "../") || strings.Contains(outputDir, "/../") {
		return fmt.Errorf("invalid output_dir")
	}

	hostDir := "/tmp/" + deploymentID

	err := os.Mkdir(hostDir, os.ModePerm)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	builderImage := strings.TrimSpace(os.Getenv("BUILDER_IMAGE"))
	if builderImage == "" {
		builderImage = "node:20"
	}

	script := `
set -eu

git clone --depth 1 "$REPO_URL" repo
cd repo
sh -lc "$BUILD_COMMAND"

if [ -z "$OUTPUT_DIR" ]; then
	cp -r ./* /workspace/
else
	if [ ! -d "$OUTPUT_DIR" ]; then
		echo "output directory not found: $OUTPUT_DIR" >&2
		exit 1
	fi
	cp -r "$OUTPUT_DIR"/. /workspace/
fi
`

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"--memory", "1024m",
		"--cpus", "1.0",
		"--pids-limit", "256",
		"-v", hostDir+":/workspace",
		"-w", "/workspace",
		"-e", "REPO_URL="+repoURL,
		"-e", "BUILD_COMMAND="+buildCommand,
		"-e", "OUTPUT_DIR="+outputDir,
		builderImage,
		"sh", "-c", script,
	)

	output, err := cmd.CombinedOutput()

	log.Println("Docker output:\n", string(output))
	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Build timed out")
		logs.AddLog(deploymentID, "Build timed out")
		return fmt.Errorf("build timed out")
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("docker CLI not found in worker runtime")
	}

	if err != nil {
		log.Println("Build failed:", err)
		return err
	}

	log.Println("Build successful")
	return nil
}
