package config

import (
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotEnoughCats = errors.New("not enough cats")

type CatAccount struct {
	UserID    string
	Login     string
	Amount    int64
	LastCatch *time.Time
}

type CatStore struct {
	pool *pgxpool.Pool
}

func LoadCats(pool *pgxpool.Pool) *CatStore {
	return &CatStore{pool: pool}
}

const catCols = `user_id, login, amount, last_catch`

func scanCat(row pgx.Row) (*CatAccount, error) {
	var a CatAccount
	if err := row.Scan(&a.UserID, &a.Login, &a.Amount, &a.LastCatch); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (s *CatStore) Get(userID string) (*CatAccount, error) {
	ctx, cancel := queryCtx()
	defer cancel()
	return scanCat(s.pool.QueryRow(ctx, `SELECT `+catCols+` FROM cats WHERE user_id = $1`, userID))
}

func (s *CatStore) FindByLogin(login string) (*CatAccount, error) {
	ctx, cancel := queryCtx()
	defer cancel()
	return scanCat(s.pool.QueryRow(ctx,
		`SELECT `+catCols+` FROM cats WHERE login = $1 ORDER BY updated_at DESC LIMIT 1`,
		strings.ToLower(login)))
}

func (s *CatStore) Rank(amount int64) (int, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var rank int
	err := s.pool.QueryRow(ctx, `SELECT count(*) + 1 FROM cats WHERE amount > $1`, amount).Scan(&rank)
	return rank, err
}

func (s *CatStore) Top(n int) ([]CatAccount, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT `+catCols+` FROM cats WHERE amount > 0 ORDER BY amount DESC, login ASC LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CatAccount
	for rows.Next() {
		var a CatAccount
		if err := rows.Scan(&a.UserID, &a.Login, &a.Amount, &a.LastCatch); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *CatStore) Catch(userID, login string, gained int64, cooldown time.Duration) (balance int64, ok bool, err error) {
	ctx, cancel := queryCtx()
	defer cancel()

	err = s.pool.QueryRow(ctx, `
		INSERT INTO cats (user_id, login, amount, last_catch)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE
		SET amount = cats.amount + EXCLUDED.amount,
		    login = EXCLUDED.login,
		    last_catch = now(),
		    updated_at = now()
		WHERE cats.last_catch IS NULL
		   OR cats.last_catch <= now() - make_interval(secs => $4::double precision)
		RETURNING amount`,
		userID, strings.ToLower(login), gained, cooldown.Seconds()).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return balance, true, nil
}

func (s *CatStore) ResetCooldown(userID string, cost int64, cooldown time.Duration) (balance int64, ok bool, err error) {
	ctx, cancel := queryCtx()
	defer cancel()

	err = s.pool.QueryRow(ctx, `
		UPDATE cats
		SET amount = amount - $2, last_catch = NULL, updated_at = now()
		WHERE user_id = $1
		  AND amount >= $2
		  AND last_catch IS NOT NULL
		  AND last_catch > now() - make_interval(secs => $3::double precision)
		RETURNING amount`,
		userID, cost, cooldown.Seconds()).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return balance, true, nil
}

func (s *CatStore) Transfer(fromID, fromLogin, toID, toLogin string, n int64) (int64, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var left int64
	err = tx.QueryRow(ctx, `
		UPDATE cats
		SET amount = amount - $2, login = $3, updated_at = now()
		WHERE user_id = $1 AND amount >= $2
		RETURNING amount`,
		fromID, n, strings.ToLower(fromLogin)).Scan(&left)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotEnoughCats
	}
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO cats (user_id, login, amount)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET amount = cats.amount + EXCLUDED.amount,
		    login = EXCLUDED.login,
		    updated_at = now()`,
		toID, strings.ToLower(toLogin), n)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return left, nil
}
