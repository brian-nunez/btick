package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenPostgres(dsn string) (*sql.DB, error) {
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	database.SetConnMaxLifetime(30 * time.Minute)
	database.SetMaxIdleConns(10)
	database.SetMaxOpenConns(20)

	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return database, nil
}
