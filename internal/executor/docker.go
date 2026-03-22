package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/logs"
	"github.com/cotishq/shipyard/internal/observability"
)

func RunBuild(deploymentID, repoURL, buildCommand, outputDir string) error {
	observability.Info("starting docker build", map[string]any{
		"deployment_id": deploymentID,
		"output_dir":    outputDir,
	})

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

	// Start each attempt with a clean workspace.
	if err := os.RemoveAll(hostDir); err != nil {
		return fmt.Errorf("failed to cleanup existing workspace: %w", err)
	}

	err := os.MkdirAll(hostDir, 0o755)
	if err != nil {
		return err
	}

	buildTimeoutMinutes := getEnvInt("MAX_BUILD_TIME_MINUTES", 10)
	maxRepoSizeMB := getEnvInt("MAX_REPO_SIZE_MB", 200)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(buildTimeoutMinutes)*time.Minute)
	defer cancel()

	builderImage := strings.TrimSpace(os.Getenv("BUILDER_IMAGE"))
	if builderImage == "" {
		builderImage = "node:20"
	}

	script := `
set -eu

SRC_DIR=$(mktemp -d)
trap 'rm -rf "$SRC_DIR"' EXIT

git clone --depth 1 "$REPO_URL" "$SRC_DIR/repo"
REPO_SIZE_MB=$(du -sm "$SRC_DIR/repo" | cut -f1)
if [ "$REPO_SIZE_MB" -gt "$MAX_REPO_SIZE_MB" ]; then
	echo "repository exceeds max size: ${REPO_SIZE_MB}MB > ${MAX_REPO_SIZE_MB}MB" >&2
	exit 1
fi

cd "$SRC_DIR/repo"
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
		"--user", strconv.Itoa(os.Getuid())+":"+strconv.Itoa(os.Getgid()),
		"-v", hostDir+":/workspace",
		"-w", "/workspace",
		"-e", "REPO_URL="+repoURL,
		"-e", "BUILD_COMMAND="+buildCommand,
		"-e", "OUTPUT_DIR="+outputDir,
		"-e", "MAX_REPO_SIZE_MB="+strconv.Itoa(maxRepoSizeMB),
		builderImage,
		"sh", "-c", script,
	)

	output, err := cmd.CombinedOutput()

	outputText := truncateBuildOutput(string(output))
	observability.Info("docker build output", map[string]any{
		"deployment_id": deploymentID,
		"output":        outputText,
	})
	logs.AddLog(deploymentID, outputText)

	if ctx.Err() == context.DeadlineExceeded {
		observability.Error("build timed out", map[string]any{
			"deployment_id":         deploymentID,
			"build_timeout_minutes": buildTimeoutMinutes,
		})
		logs.AddLog(deploymentID, "Build timed out")
		return fmt.Errorf("build timed out after %d minutes", buildTimeoutMinutes)
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("docker CLI not found in worker runtime")
	}

	if err != nil {
		observability.Error("build failed", map[string]any{
			"deployment_id": deploymentID,
			"error":         err.Error(),
		})
		return err
	}

	observability.Info("build successful", map[string]any{
		"deployment_id": deploymentID,
	})
	return nil
}

func truncateBuildOutput(output string) string {
	maxLen := getEnvInt("MAX_LOG_SIZE_BYTES", 8192)
	if maxLen <= 0 {
		return ""
	}
	if len(output) <= maxLen {
		return output
	}

	suffix := "\n...[build output truncated]"
	if maxLen <= len(suffix) {
		return suffix[:maxLen]
	}

	cutoff := maxLen - len(suffix)
	if cutoff < 0 {
		cutoff = 0
	}

	return output[:cutoff] + suffix
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
