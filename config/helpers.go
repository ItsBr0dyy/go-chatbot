package config

import (
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Helpers struct {
	mu   sync.RWMutex
	pool *pgxpool.Pool
	set  map[string]struct{}
}

func LoadHelpers(pool *pgxpool.Pool) (*Helpers, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := pool.Query(ctx, `SELECT login FROM helpers`)
	if err != nil {
		return nil, err
	}
	logins, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	h := &Helpers{pool: pool, set: map[string]struct{}{}}
	for _, l := range logins {
		h.set[strings.ToLower(l)] = struct{}{}
	}
	return h, nil
}

func (h *Helpers) Has(login string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.set[strings.ToLower(login)]
	return ok
}

func (h *Helpers) Add(login, addedBy string) (bool, error) {
	login = strings.ToLower(login)

	ctx, cancel := queryCtx()
	defer cancel()

	tag, err := h.pool.Exec(ctx,
		`INSERT INTO helpers (login, added_by) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		login, strings.ToLower(addedBy))
	if err != nil {
		return false, err
	}

	h.mu.Lock()
	h.set[login] = struct{}{}
	h.mu.Unlock()

	return tag.RowsAffected() > 0, nil
}

func (h *Helpers) Remove(login string) (bool, error) {
	login = strings.ToLower(login)

	ctx, cancel := queryCtx()
	defer cancel()

	tag, err := h.pool.Exec(ctx, `DELETE FROM helpers WHERE login = $1`, login)
	if err != nil {
		return false, err
	}

	h.mu.Lock()
	delete(h.set, login)
	h.mu.Unlock()

	return tag.RowsAffected() > 0, nil
}
