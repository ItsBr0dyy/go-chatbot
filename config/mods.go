package config

import (
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Mods struct {
	mu   sync.RWMutex
	pool *pgxpool.Pool
	set  map[string]struct{} // "channel/login"
}

func modKey(channel, login string) string {
	return strings.ToLower(channel) + "/" + strings.ToLower(login)
}

func LoadMods(pool *pgxpool.Pool) (*Mods, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := pool.Query(ctx, `SELECT channel, login FROM channel_mods`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := &Mods{pool: pool, set: map[string]struct{}{}}
	for rows.Next() {
		var channel, login string
		if err := rows.Scan(&channel, &login); err != nil {
			return nil, err
		}
		m.set[modKey(channel, login)] = struct{}{}
	}
	return m, rows.Err()
}

func (m *Mods) Has(channel, login string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.set[modKey(channel, login)]
	return ok
}

func (m *Mods) Add(channel, login, setBy string) (bool, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	tag, err := m.pool.Exec(ctx,
		`INSERT INTO channel_mods (channel, login, set_by) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		strings.ToLower(channel), strings.ToLower(login), strings.ToLower(setBy))
	if err != nil {
		return false, err
	}

	m.mu.Lock()
	m.set[modKey(channel, login)] = struct{}{}
	m.mu.Unlock()
	return tag.RowsAffected() > 0, nil
}

func (m *Mods) Remove(channel, login string) (bool, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	tag, err := m.pool.Exec(ctx,
		`DELETE FROM channel_mods WHERE channel = $1 AND login = $2`,
		strings.ToLower(channel), strings.ToLower(login))
	if err != nil {
		return false, err
	}

	m.mu.Lock()
	delete(m.set, modKey(channel, login))
	m.mu.Unlock()
	return tag.RowsAffected() > 0, nil
}
