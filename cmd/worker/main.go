package main

import (
	"log"
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/executor"
)

func main() {
	db.Init()

	log.Println("worker started")

	for {
		executor.ProcessNextDeployment()
		time.Sleep(5 * time.Second)
	}
}
