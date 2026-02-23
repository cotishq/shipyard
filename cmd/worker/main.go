package main

import (
	"log"
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/executor"
	"github.com/cotishq/shipyard/internal/storage"
)

func main() {
	db.Init()
	storage.Init()

	log.Println("worker started")

	for {
		executor.ProcessNextDeployment()
		time.Sleep(5 * time.Second)
	}
}
