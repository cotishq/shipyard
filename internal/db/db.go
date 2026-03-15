package db

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/config"
	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() {
	var err error

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = config.DefaultDatabaseURL
	}
	if err := config.ValidateDatabaseURL(dsn); err != nil && !config.AllowInsecureDefaults() {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		DB, err = sql.Open("postgres", dsn)
		if err == nil {
			err = DB.Ping()
			if err == nil {
				log.Println("Connected to PostgreSQL")
				if err := RunMigrations(DB); err != nil {
					log.Fatal("failed to run migrations:", err)
				}
				return
			}
		}

		log.Println("Waiting for database...")
		time.Sleep(2 * time.Second)
	}

	log.Fatal("DB not reachable:", err)
}
