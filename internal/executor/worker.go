package executor

import (
	"log"
	"time"

	"github.com/cotishq/shipyard/internal/db"
)



func ProcessNextDeployment() {
	log.Println("checking for deployments")

	var id string

	err := db.DB.QueryRow(`
	SELECT id FROM deployments
	WHERE status = 'QUEUED'
	ORDER BY created_at
	LIMIT 1
	`).Scan(&id)

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

	time.Sleep(5 * time.Second)

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