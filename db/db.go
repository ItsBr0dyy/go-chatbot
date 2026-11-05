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

CREATE TABLE IF NOT EXISTS account_links (
	twitch_login TEXT NOT NULL,
	service      TEXT NOT NULL,
	username     TEXT NOT NULL,
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (twitch_login, service)
);

CREATE TABLE IF NOT EXISTS reminders (
	id           BIGSERIAL PRIMARY KEY,
	target_login TEXT NOT NULL,
	author_login TEXT NOT NULL,
	channel      TEXT NOT NULL,
	message      TEXT NOT NULL,
	remind_at    TIMESTAMPTZ,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS reminders_target_idx ON reminders (target_login);
CREATE INDEX IF NOT EXISTS reminders_due_idx ON reminders (remind_at) WHERE remind_at IS NOT NULL;

INSERT INTO account_links (twitch_login, service, username, updated_at)
SELECT twitch_login, 'lastfm', lastfm_user, updated_at FROM links
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS cats (
	user_id    TEXT PRIMARY KEY,
	login      TEXT NOT NULL,
	amount     BIGINT NOT NULL DEFAULT 0 CHECK (amount >= 0),
	last_catch TIMESTAMPTZ,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cats_amount_idx ON cats (amount DESC);
CREATE INDEX IF NOT EXISTS cats_login_idx ON cats (login);
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
