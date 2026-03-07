package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func RunMigrations(ctx context.Context, database *sql.DB, migrationDir string) error {
	if err := ensureMigrationsTable(ctx, database); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versions = append(versions, entry.Name())
	}
	sort.Strings(versions)

	for _, version := range versions {
		applied, err := migrationApplied(ctx, database, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationDir, version))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s transaction: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func migrationApplied(ctx context.Context, database *sql.DB, version string) (bool, error) {
	var exists bool
	err := database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return exists, nil
}
