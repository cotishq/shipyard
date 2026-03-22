package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/config"
	"github.com/cotishq/shipyard/internal/observability"
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
				observability.Info("connected to PostgreSQL", nil)
				if err := RunMigrations(DB); err != nil {
					log.Fatal("failed to run migrations:", err)
				}
				return
			}
		}

		observability.Info("waiting for database", map[string]any{
			"attempt": i + 1,
		})
		time.Sleep(2 * time.Second)
	}

	log.Fatal("DB not reachable:", err)
}

func HealthCheck(ctx context.Context) error {
	if DB == nil {
		return sql.ErrConnDone
	}
	return DB.PingContext(ctx)
}
