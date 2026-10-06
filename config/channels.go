package config

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Channels struct {
	pool *pgxpool.Pool
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "#"))
}

func queryCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func LoadChannels(pool *pgxpool.Pool, seed ...string) (*Channels, error) {
	c := &Channels{pool: pool}

	ctx, cancel := queryCtx()
	defer cancel()

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM channels`).Scan(&n); err != nil {
		return nil, err
	}
	if n == 0 {
		for _, s := range seed {
			if _, err := c.Add(s); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

func (c *Channels) List() ([]string, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := c.pool.Query(ctx, `SELECT name FROM channels ORDER BY added_at`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (c *Channels) Add(name string) (bool, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	tag, err := c.pool.Exec(ctx,
		`INSERT INTO channels (name) VALUES ($1) ON CONFLICT DO NOTHING`, normalize(name))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (c *Channels) Remove(name string) (bool, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	tag, err := c.pool.Exec(ctx, `DELETE FROM channels WHERE name = $1`, normalize(name))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
