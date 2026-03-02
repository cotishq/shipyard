package logs

import (
	"log"

	"github.com/cotishq/shipyard/internal/db"
)

func AddLog(deploymentID string, message string) {
	_, err := db.DB.Exec(`
		INSERT INTO deployment_logs (deployment_id, message)
		VALUES ($1, $2)
	`, deploymentID, message)

	if err != nil {
		log.Println("failed to insert log:", err)
	}
}
