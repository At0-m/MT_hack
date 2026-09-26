package storage

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// Global pre-auth budget shared across replicas. The caller uses one fixed key.
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
 WHERE login_limits.window_start<=now()-interval '1 minute' OR login_limits.attempts<120
 RETURNING attempts`, account).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, dependency(err)
	}
	return true, nil
}

// Failure state is updated after bcrypt. It changes only invalid-login responses;
// correct credentials always get checked and clear this state on success.
func (s *Store) RecordLogin(ctx context.Context, account string, success bool) (bool, error) {
	if success {
		_, err := s.Pool.Exec(ctx, "DELETE FROM login_limits WHERE account_hash=$1", account)
		if err != nil {
			return false, dependency(err)
		}
		return true, nil
	}
	var attempts int
	err := s.Pool.QueryRow(ctx, `INSERT INTO login_limits VALUES($1,now(),1)
 ON CONFLICT(account_hash) DO UPDATE SET
 attempts=CASE WHEN login_limits.window_start<=now()-interval '1 minute' THEN 1 ELSE LEAST(login_limits.attempts+1,11) END,
 window_start=CASE WHEN login_limits.window_start<=now()-interval '1 minute' THEN now() ELSE login_limits.window_start END
 RETURNING attempts`, account).Scan(&attempts)
	if err != nil {
		return false, dependency(err)
	}
	return attempts <= 10, nil
}
