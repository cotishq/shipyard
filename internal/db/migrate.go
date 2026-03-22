package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cotishq/shipyard/internal/observability"
	projectmigrations "github.com/cotishq/shipyard/migrations"
)

const migrationLockID int64 = 918273645

func RunMigrations(db *sql.DB) error {
	if err := acquireMigrationLock(db); err != nil {
		return err
	}
	defer func() {
		if err := releaseMigrationLock(db); err != nil {
			observability.Error("failed to release migration lock", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	entries, err := projectmigrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		migrationFiles = append(migrationFiles, entry.Name())
	}
	sort.Strings(migrationFiles)

	for _, name := range migrationFiles {
		applied, err := migrationApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		contents, err := projectmigrations.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if err := applyMigration(db, name, string(contents)); err != nil {
			return err
		}

		observability.Info("applied migration", map[string]any{
			"migration": name,
		})
	}

	return nil
}

func acquireMigrationLock(db *sql.DB) error {
	_, err := db.Exec(`SELECT pg_advisory_lock($1)`, migrationLockID)
	if err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	return nil
}

func releaseMigrationLock(db *sql.DB) error {
	_, err := db.Exec(`SELECT pg_advisory_unlock($1)`, migrationLockID)
	if err != nil {
		return fmt.Errorf("release migration lock: %w", err)
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func migrationApplied(db *sql.DB, version string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE version = $1
		)
	`, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return exists, nil
}

func applyMigration(db *sql.DB, version, sqlText string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(strings.TrimSpace(sqlText)); err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}

	if _, err = tx.Exec(`
		INSERT INTO schema_migrations (version)
		VALUES ($1)
	`, version); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}

	return nil
}
