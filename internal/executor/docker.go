package executor

import (
	"log"
	"os/exec"
)

func RunBuild(repoURL string, buildCommand string) error {
	log.Println("Starting Docker build...")

	cmd := exec.Command("docker","run", "--rm",
        "-v", "/tmp:/workspace", 
	    "-w", "/workspace",
	    "node:20",
	    "bash", "-c",
	    "git clone "+repoURL+"repo && cd repo && npm install && "+buildCommand,
	)

	output, err := cmd.CombinedOutput()

	log.Println("Docker output:\n", string(output))

	if err != nil {
		log.Println("Build failed:", err)
		return err
	}

	log.Println("Build successful")
	return nil
}