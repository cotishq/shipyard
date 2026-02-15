package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)



var DB *sql.DB

func Init() {
	connStr := "postgres://postgres:postgres@localhost:5432/shipyard?sslmode=disable"

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("DB not reachable:", err)
	}

	log.Println("Connected to PostgreSQL")
}