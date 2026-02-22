package executor

import (
	"log"
	"os"
	"os/exec"
)

func RunBuild(deploymentID , repoURL , buildCommand, outputDir string ) error {
	log.Println("Starting Docker build...")

	hostDir := "/tmp/" + deploymentID

	err := os.Mkdir(hostDir, os.ModePerm)
	if err != nil {
		return err
	}

	cmd := exec.Command("docker","run", "--rm",
        "-v", hostDir+":/workspace", 
	    "-w", "/workspace",
	    "node:20",
	    "bash", "-c",
	    "git clone "+repoURL+".git repo && cd repo && npm install && "+buildCommand+" && if [ -d "+outputDir+" ]; then cp -r "+outputDir+"/* /workspace/; else echo 'No output dir'; fi",
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