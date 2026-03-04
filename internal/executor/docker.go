package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/cotishq/shipyard/internal/logs"
)

func RunBuild(deploymentID, repoURL, buildCommand, outputDir string) error {
	log.Println("Starting Docker build...")

	hostDir := "/tmp/" + deploymentID

	err := os.Mkdir(hostDir, os.ModePerm)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"-v", hostDir+":/workspace",
		"-w", "/workspace",
		"node:20",
		"bash", "-c",
		"git clone "+repoURL+".git repo && cd repo && "+buildCommand+" && if [ -z \""+outputDir+"\" ]; then cp -r * /workspace/; else cp -r "+outputDir+"/* /workspace/; fi",
	)

	output, err := cmd.CombinedOutput()

	log.Println("Docker output:\n", string(output))
	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Build timed out")
		logs.AddLog(deploymentID, "Build timed out")
		return fmt.Errorf("build timed out")
	}

	if err != nil {
		log.Println("Build failed:", err)
		return err
	}

	log.Println("Build successful")
	return nil
}
