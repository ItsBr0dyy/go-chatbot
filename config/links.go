package config

import (
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Links struct {
	pool *pgxpool.Pool
}

func LoadLinks(pool *pgxpool.Pool) *Links {
	return &Links{pool: pool}
}

func (l *Links) Get(twitchLogin string) (string, bool) {
	ctx, cancel := queryCtx()
	defer cancel()

	var user string
	err := l.pool.QueryRow(ctx,
		`SELECT lastfm_user FROM links WHERE twitch_login = $1`,
		strings.ToLower(twitchLogin)).Scan(&user)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			log.Printf("links get: %v", err)
		}
		return "", false
	}
	return user, true
}

func (l *Links) Set(twitchLogin, lastfmUser string) error {
	ctx, cancel := queryCtx()
	defer cancel()

	_, err := l.pool.Exec(ctx, `
		INSERT INTO links (twitch_login, lastfm_user)
		VALUES ($1, $2)
		ON CONFLICT (twitch_login)
		DO UPDATE SET lastfm_user = EXCLUDED.lastfm_user, updated_at = now()`,
		strings.ToLower(twitchLogin), lastfmUser)
	return err
}
