package executor

import (
	"log"
	"os"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/logs"
	"github.com/cotishq/shipyard/internal/storage"
	"github.com/cotishq/shipyard/internal/utils"
)

func ProcessNextDeployment() {
	log.Println("checking for deployments")

	maxConcurrentBuilds := getEnvInt("MAX_CONCURRENT_BUILDS", 1)

	var currentlyBuilding int
	err := db.DB.QueryRow(`
	  SELECT COUNT(*)
	  FROM deployments
	  WHERE status = 'BUILDING'
	  `).Scan(&currentlyBuilding)
	  if err != nil {
		log.Println("failed to count active builds:", err)
		return
	  }

	  if currentlyBuilding >= maxConcurrentBuilds {
		log.Println("max concurrent builds reached")
		return
	  }

	var id, repoURL, buildCommand, outputDir string

	err = db.DB.QueryRow(`
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
	logs.AddLog(id, "Starting deployment")

	result, err := db.DB.Exec(`
	UPDATE deployments
	SET status = 'BUILDING'
	WHERE id = $1 AND status = 'QUEUED'
	`, id)

	if err != nil {
		log.Println("failed to update status:", err)
		return
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Println("failed to read affected rows:", err)
		return
	}
	if rows == 0 {
		log.Println("invalid state transition, skipping:", id)
		return
	}

	// Ensure workspace is cleaned regardless of success/failure/return path.
	defer func() {
		if err := cleanupWorkspace(id); err != nil {
			log.Println("failed to cleanup workspace:", err)
		}
	}()

	if err := cleanupWorkspace(id); err != nil {
		log.Println("failed to cleanup workspace before build:", err)
	}

	logs.AddLog(id, "Running build...")
	err = RunBuild(id, repoURL, buildCommand, outputDir)

	if err != nil {
		log.Println("Build failed:", err)
		logs.AddLog(id, "Build failed: "+err.Error())

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

		result, err = db.DB.Exec(`
			UPDATE deployments
			SET attempt_count = $1,
			    status = 'FAILED'
			WHERE id = $2 AND status = 'BUILDING'
		`, attemptCount, id)
		if err != nil {
			log.Println("failed to update failed status:", err)
			return
		}
		rows, err = result.RowsAffected()
		if err != nil {
			log.Println("failed to read affected rows:", err)
			return
		}
		if rows == 0 {
			log.Println("invalid state transition, skipping:", id)
			return
		}

		if attemptCount < maxAttempts {
			log.Println("Retrying deployment:", id)
			logs.AddLog(id, "Retrying deployment")

			result, err = db.DB.Exec(`
			UPDATE deployments
			SET attempt_count = $1,
			    status = 'QUEUED'
			WHERE id = $2 AND status = 'FAILED'
		`, attemptCount, id)
			if err != nil {
				log.Println("failed to queue retry:", err)
				return
			}
			rows, err = result.RowsAffected()
			if err != nil {
				log.Println("failed to read affected rows:", err)
				return
			}
			if rows == 0 {
				log.Println("invalid state transition, skipping:", id)
				return
			}
		} else {
			log.Println("Max retries reached:", id)
		}

		return
	}

	logs.AddLog(id, "Build successful")

	checksum, err := utils.CalculateChecksum("/tmp/" + id + "/repo/index.html")
	if err != nil {
		log.Println("checksum error:", err)
	}

	err = storage.UploadFolder(id)
	if err != nil {
		log.Println("Upload failed:", err)

		result, err = db.DB.Exec(`
		UPDATE deployments
		SET status = 'FAILED'
		WHERE id = $1 AND status = 'BUILDING'
		`, id)
		if err != nil {
			log.Println("failed to update status:", err)
			return
		}
		rows, err = result.RowsAffected()
		if err != nil {
			log.Println("failed to read affected rows:", err)
			return
		}
		if rows == 0 {
			log.Println("invalid state transition, skipping:", id)
		}

		return
	}

	_, err = db.DB.Exec(`
	UPDATE deployments
	SET artifact_checksum = $1
	WHERE id = $2
	`, checksum, id)
	if err != nil {
		log.Println("failed to store checksum:", err)
	}

	result, err = db.DB.Exec(`
	UPDATE deployments
	SET status = 'READY'
	WHERE id = $1 AND status = 'BUILDING'
	`, id)

	if err != nil {
		log.Println("failed to update status:", err)
		return
	}
	rows, err = result.RowsAffected()
	if err != nil {
		log.Println("failed to read affected rows:", err)
		return
	}
	if rows == 0 {
		log.Println("invalid state transition, skipping:", id)
		return
	}

	logs.AddLog(id, "Deployment ready")
	log.Println("Deployment ready:", id)
}

func cleanupWorkspace(id string) error {
	workspace := "/tmp/" + id
	if err := os.RemoveAll(workspace); err != nil {
		return err
	}
	return nil
}
