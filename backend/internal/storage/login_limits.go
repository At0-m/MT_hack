package storage

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// Shared across API replicas, including attempts against nonexistent usernames.
func (s *Store) AllowLogin(ctx context.Context, account string) (bool, error) {
	_, err := s.Pool.Exec(ctx, `DELETE FROM login_limits WHERE account_hash IN
 (SELECT account_hash FROM login_limits WHERE window_start<=now()-interval '1 minute' LIMIT 512)`)
	if err != nil {
		return false, dependency(err)
	}
	var count int
	err = s.Pool.QueryRow(ctx, `INSERT INTO login_limits VALUES($1,now(),1)
 ON CONFLICT(account_hash) DO UPDATE SET
 attempts=CASE WHEN login_limits.window_start<=now()-interval '1 minute' THEN 1 ELSE login_limits.attempts+1 END,
 window_start=CASE WHEN login_limits.window_start<=now()-interval '1 minute' THEN now() ELSE login_limits.window_start END
 WHERE login_limits.window_start<=now()-interval '1 minute' OR login_limits.attempts<10
 RETURNING attempts`, account).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, dependency(err)
	}
	return true, nil
}
