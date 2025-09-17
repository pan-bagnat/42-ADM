package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Connect opens a PostgreSQL connection using the provided URL and configures
// sane defaults for connection pooling.
func Connect(ctx context.Context, url string) (*sql.DB, error) {
	if url == "" {
		return nil, fmt.Errorf("database url is required")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	// Ensure the database is reachable before returning.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// Conservative defaults; callers can adjust if necessary.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}
