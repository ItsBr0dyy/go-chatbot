package config

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Reminder struct {
	ID        int64
	Target    string
	Author    string
	Channel   string
	Message   string
	CreatedAt time.Time
}

type Reminders struct {
	pool    *pgxpool.Pool
	mu      sync.Mutex
	waiting map[string]int
}

const reminderCols = `id, target_login, author_login, channel, message, created_at`

func LoadReminders(pool *pgxpool.Pool) (*Reminders, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := pool.Query(ctx, `SELECT target_login FROM reminders WHERE remind_at IS NULL`)
	if err != nil {
		return nil, err
	}
	logins, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	r := &Reminders{pool: pool, waiting: map[string]int{}}
	for _, l := range logins {
		r.waiting[l]++
	}
	return r, nil
}

func (r *Reminders) Add(target, author, channel, message string, at *time.Time) error {
	target = strings.ToLower(target)

	ctx, cancel := queryCtx()
	defer cancel()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO reminders (target_login, author_login, channel, message, remind_at)
		VALUES ($1, $2, $3, $4, $5)`,
		target, strings.ToLower(author), strings.ToLower(channel), message, at)
	if err != nil {
		return err
	}

	if at == nil {
		r.mu.Lock()
		r.waiting[target]++
		r.mu.Unlock()
	}
	return nil
}

func (r *Reminders) CountPending(author string) (int, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM reminders WHERE author_login = $1`, strings.ToLower(author)).Scan(&n)
	return n, err
}

func (r *Reminders) HasWaiting(target string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.waiting[strings.ToLower(target)] > 0
}

func (r *Reminders) TakeWaiting(target string) ([]Reminder, error) {
	target = strings.ToLower(target)
	list, err := r.take(
		`DELETE FROM reminders WHERE target_login = $1 AND remind_at IS NULL RETURNING `+reminderCols, target)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.waiting[target] -= len(list)
	if r.waiting[target] <= 0 {
		delete(r.waiting, target)
	}
	r.mu.Unlock()
	return list, nil
}

func (r *Reminders) TakeDue() ([]Reminder, error) {
	return r.take(
		`DELETE FROM reminders WHERE remind_at IS NOT NULL AND remind_at <= now() RETURNING ` + reminderCols)
}

func (r *Reminders) take(sql string, args ...any) ([]Reminder, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Reminder
	for rows.Next() {
		var x Reminder
		if err := rows.Scan(&x.ID, &x.Target, &x.Author, &x.Channel, &x.Message, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
