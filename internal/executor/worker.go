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

	var attemptCount, maxAttempts int

	err2 := db.DB.QueryRow(`
		SELECT attempt_count, max_attempts
		FROM deployments
		WHERE id = $1
	`, id).Scan(&attemptCount, &maxAttempts)

	if err2 != nil {
		log.Println("failed to fetch attempts:", err2)
		return
	}

	attemptCount++

	if attemptCount < maxAttempts {
		log.Println("Retrying deployment:", id)

		_, err = db.DB.Exec(`
			UPDATE deployments
			SET attempt_count = $1,
			    status = 'QUEUED'
			WHERE id = $2
		`, attemptCount, id)

	} else {
		log.Println("Max retries reached:", id)

		_, err = db.DB.Exec(`
			UPDATE deployments
			SET attempt_count = $1,
			    status = 'FAILED'
			WHERE id = $2
		`, attemptCount, id)
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