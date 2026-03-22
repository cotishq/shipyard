package executor

import (
	"os"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/logs"
	"github.com/cotishq/shipyard/internal/observability"
	"github.com/cotishq/shipyard/internal/storage"
	"github.com/cotishq/shipyard/internal/utils"
)

func ProcessNextDeployment() {
	observability.Info("checking for deployments", nil)

	maxConcurrentBuilds := getEnvInt("MAX_CONCURRENT_BUILDS", 1)

	var currentlyBuilding int
	err := db.DB.QueryRow(`
	  SELECT COUNT(*)
	  FROM deployments
	  WHERE status = 'BUILDING'
	`).Scan(&currentlyBuilding)
	if err != nil {
		observability.Error("failed to count active builds", map[string]any{
			"error": err.Error(),
		})
		return
	}

	if currentlyBuilding >= maxConcurrentBuilds {
		observability.Info("max concurrent builds reached", map[string]any{
			"active_builds":         currentlyBuilding,
			"max_concurrent_builds": maxConcurrentBuilds,
		})
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

	observability.Info("processing deployment", map[string]any{
		"deployment_id": id,
	})
	logs.AddLog(id, "Starting deployment")

	result, err := db.DB.Exec(`
	UPDATE deployments
	SET status = 'BUILDING',
	    started_at = NOW(),
	    finished_at = NULL,
	    error_message = NULL,
	    build_duration_seconds = NULL
	WHERE id = $1 AND status = 'QUEUED'
	`, id)

	if err != nil {
		observability.Error("failed to update deployment status to BUILDING", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
		return
	}
	rows, err := result.RowsAffected()
	if err != nil {
		observability.Error("failed to read affected rows", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
		return
	}
	if rows == 0 {
		observability.Info("invalid state transition to BUILDING", map[string]any{
			"deployment_id": id,
		})
		return
	}

	// Ensure workspace is cleaned regardless of success/failure/return path.
	defer func() {
		if err := cleanupWorkspace(id); err != nil {
			observability.Error("failed to cleanup workspace", map[string]any{
				"deployment_id": id,
				"error":         err.Error(),
			})
		}
	}()

	if err := cleanupWorkspace(id); err != nil {
		observability.Error("failed to cleanup workspace before build", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
	}

	logs.AddLog(id, "Running build...")
	err = RunBuild(id, repoURL, buildCommand, outputDir)

	if err != nil {
		observability.Error("deployment build failed", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
		logs.AddLog(id, "Build failed: "+err.Error())

		var attemptCount, maxAttempts int

		err2 := db.DB.QueryRow(`
		SELECT attempt_count, max_attempts
		FROM deployments
		WHERE id = $1
	`, id).Scan(&attemptCount, &maxAttempts)

		if err2 != nil {
			observability.Error("failed to fetch deployment attempts", map[string]any{
				"deployment_id": id,
				"error":         err2.Error(),
			})
			return
		}

		attemptCount++

		result, err = db.DB.Exec(`
				UPDATE deployments
				SET attempt_count = $1,
				    status = 'FAILED',
				    error_message = $2,
				    finished_at = NOW(),
				    build_duration_seconds = GREATEST(0, EXTRACT(EPOCH FROM (NOW() - COALESCE(started_at, NOW())))::INT)
				WHERE id = $3 AND status = 'BUILDING'
			`, attemptCount, err.Error(), id)
		if err != nil {
			observability.Error("failed to update deployment to FAILED", map[string]any{
				"deployment_id": id,
				"error":         err.Error(),
			})
			return
		}
		rows, err = result.RowsAffected()
		if err != nil {
			observability.Error("failed to read affected rows", map[string]any{
				"deployment_id": id,
				"error":         err.Error(),
			})
			return
		}
		if rows == 0 {
			observability.Info("invalid state transition to FAILED", map[string]any{
				"deployment_id": id,
			})
			return
		}

		if attemptCount < maxAttempts {
			observability.Info("retrying deployment", map[string]any{
				"deployment_id": id,
				"attempt_count": attemptCount,
				"max_attempts":  maxAttempts,
			})
			logs.AddLog(id, "Retrying deployment")

			result, err = db.DB.Exec(`
			UPDATE deployments
			SET attempt_count = $1,
			    status = 'QUEUED'
			WHERE id = $2 AND status = 'FAILED'
		`, attemptCount, id)
			if err != nil {
				observability.Error("failed to queue retry", map[string]any{
					"deployment_id": id,
					"error":         err.Error(),
				})
				return
			}
			rows, err = result.RowsAffected()
			if err != nil {
				observability.Error("failed to read affected rows", map[string]any{
					"deployment_id": id,
					"error":         err.Error(),
				})
				return
			}
			if rows == 0 {
				observability.Info("invalid state transition to QUEUED", map[string]any{
					"deployment_id": id,
				})
				return
			}
		} else {
			observability.Error("max retries reached", map[string]any{
				"deployment_id": id,
				"attempt_count": attemptCount,
				"max_attempts":  maxAttempts,
			})
		}

		return
	}

	logs.AddLog(id, "Build successful")

	checksum, err := utils.CalculateChecksum("/tmp/" + id)
	if err != nil {
		observability.Error("checksum calculation failed", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
	}

	err = storage.UploadFolder(id)
	if err != nil {
		observability.Error("artifact upload failed", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})

		result, err = db.DB.Exec(`
		UPDATE deployments
		SET status = 'FAILED',
		    error_message = $2,
		    finished_at = NOW(),
		    build_duration_seconds = GREATEST(0, EXTRACT(EPOCH FROM (NOW() - COALESCE(started_at, NOW())))::INT)
		WHERE id = $1 AND status = 'BUILDING'
		`, id, err.Error())
		if err != nil {
			observability.Error("failed to update deployment to FAILED after upload error", map[string]any{
				"deployment_id": id,
				"error":         err.Error(),
			})
			return
		}
		rows, err = result.RowsAffected()
		if err != nil {
			observability.Error("failed to read affected rows", map[string]any{
				"deployment_id": id,
				"error":         err.Error(),
			})
			return
		}
		if rows == 0 {
			observability.Info("invalid state transition to FAILED after upload error", map[string]any{
				"deployment_id": id,
			})
		}

		return
	}

	_, err = db.DB.Exec(`
	UPDATE deployments
	SET artifact_checksum = $1
	WHERE id = $2
	`, checksum, id)
	if err != nil {
		observability.Error("failed to store artifact checksum", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
	}

	result, err = db.DB.Exec(`
	UPDATE deployments
	SET status = 'READY',
	    error_message = NULL,
	    finished_at = NOW(),
	    build_duration_seconds = GREATEST(0, EXTRACT(EPOCH FROM (NOW() - COALESCE(started_at, NOW())))::INT)
	WHERE id = $1 AND status = 'BUILDING'
	`, id)

	if err != nil {
		observability.Error("failed to update deployment to READY", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
		return
	}
	rows, err = result.RowsAffected()
	if err != nil {
		observability.Error("failed to read affected rows", map[string]any{
			"deployment_id": id,
			"error":         err.Error(),
		})
		return
	}
	if rows == 0 {
		observability.Info("invalid state transition to READY", map[string]any{
			"deployment_id": id,
		})
		return
	}

	logs.AddLog(id, "Deployment ready")
	observability.Info("deployment ready", map[string]any{
		"deployment_id": id,
	})
}

func cleanupWorkspace(id string) error {
	workspace := "/tmp/" + id
	if err := os.RemoveAll(workspace); err != nil {
		return err
	}
	return nil
}
