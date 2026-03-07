package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)



var DB *sql.DB

func Init() {
	var err error

	dsn := "postgres://postgres:postgres@postgres:5432/shipyard?sslmode=disable"

	for i := 0; i < 10; i++ {
		DB, err = sql.Open("postgres", dsn)
		if err == nil {
			err = DB.Ping()
			if err == nil {
				log.Println("Connected to PostgreSQL")
				return
			}
		}

		log.Println("Waiting for database...")
		time.Sleep(2 * time.Second)
	}

	log.Fatal("DB not reachable:", err)
}