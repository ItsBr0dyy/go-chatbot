package config

import (
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Links struct {
	pool *pgxpool.Pool
}

func LoadLinks(pool *pgxpool.Pool) *Links {
	return &Links{pool: pool}
}

func (l *Links) Get(service, twitchLogin string) (string, bool) {
	ctx, cancel := queryCtx()
	defer cancel()

	var user string
	err := l.pool.QueryRow(ctx,
		`SELECT username FROM account_links WHERE twitch_login = $1 AND service = $2`,
		strings.ToLower(twitchLogin), service).Scan(&user)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("links get: %v", err)
		}
		return "", false
	}
	return user, true
}

func (l *Links) Set(service, twitchLogin, username string) error {
	ctx, cancel := queryCtx()
	defer cancel()

	_, err := l.pool.Exec(ctx, `
		INSERT INTO account_links (twitch_login, service, username)
		VALUES ($1, $2, $3)
		ON CONFLICT (twitch_login, service)
		DO UPDATE SET username = EXCLUDED.username, updated_at = now()`,
		strings.ToLower(twitchLogin), service, username)
	return err
}
