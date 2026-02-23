package executor

import (
	"log"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/storage"
)

func ProcessNextDeployment() {
	log.Println("checking for deployments")

	var id, repoURL, buildCommand, outputDir string

	err := db.DB.QueryRow(`
	SELECT id, repo_url, build_command, output_dir
	FROM deployments
	WHERE status = 'QUEUED'
	ORDER BY created_at
	LIMIT 1
	`).Scan(&id, &repoURL, &buildCommand, &outputDir)

	if err != nil {
		return
	}

	log.Println("Processing deployment:", id)

	_, err = db.DB.Exec(`
	UPDATE deployments
	SET status = 'BUILDING'
	WHERE id = $1
	`, id)

	if err != nil {
		log.Println("failed to update status:", err)
		return
	}

	err = RunBuild(id, repoURL, buildCommand, outputDir)

	if err != nil {
		log.Println("Build failed:", err)

		_, err = db.DB.Exec(`
		UPDATE deployments
		SET status = 'FAILED'
		WHERE id = $1
		`, id)

		if err != nil {
			log.Println("failed to update status:", err)
		}

		return
	}

	err = storage.UploadFolder(id)
	if err != nil {
		log.Println("Upload failed:", err)

		_, err = db.DB.Exec(`
		UPDATE deployments
		SET status = 'FAILED'
		WHERE id = $1
		`, id)

		return
	}

	_, err = db.DB.Exec(`
	UPDATE deployments
	SET status = 'READY'
	WHERE id = $1
	`, id)

	if err != nil {
		log.Println("failed to update status:", err)
		return
	}

	log.Println("Deployment ready:", id)
}