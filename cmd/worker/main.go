package main

import (
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/executor"
	"github.com/cotishq/shipyard/internal/observability"
	"github.com/cotishq/shipyard/internal/storage"
)

func main() {
	db.Init()
	storage.Init()

	observability.Info("worker started", nil)

	for {
		executor.ProcessNextDeployment()
		time.Sleep(5 * time.Second)
	}
}
