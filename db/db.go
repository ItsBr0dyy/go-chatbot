package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS channels (
	name     TEXT PRIMARY KEY,
	added_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS links (
	twitch_login TEXT PRIMARY KEY,
	lastfm_user  TEXT NOT NULL,
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS helpers (
	login    TEXT PRIMARY KEY,
	added_by TEXT NOT NULL,
	added_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("opening pool: %w", err)
	}

	for i := 1; i <= 15; i++ {
		if err = pool.Ping(ctx); err == nil {
			break
		}
		log.Printf("waiting for postgres (%d/15): %v", i, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres unreachable: %w", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("creating tables: %w", err)
	}
	return pool, nil
}
