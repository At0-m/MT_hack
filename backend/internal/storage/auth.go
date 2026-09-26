package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"time"
	"tramflow/internal/auth"
	d "tramflow/internal/domain"
)

func (s *Store) User(ctx context.Context, username string) (auth.User, string, error) {
	var u auth.User
	var hash string
	err := s.Pool.QueryRow(ctx, "SELECT user_id,username,display_name,password_hash FROM app_users WHERE username=$1 AND NOT disabled", username).Scan(&u.ID, &u.Username, &u.DisplayName, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, "", nil
	}
	if err != nil {
		return u, "", dependency(err)
	}
	return u, hash, nil
}
func (s *Store) ProvisionUser(ctx context.Context, username, password string) error {
	if len(username) < 1 || len(username) > 128 || len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("username must be 1..128 bytes; password 12..72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, "INSERT INTO app_users(user_id,username,display_name,password_hash) VALUES($1,$2,$3,$4) ON CONFLICT(username) DO UPDATE SET password_hash=EXCLUDED.password_hash", auth.Hash(username)[:32], username, username, string(hash))
	return err
}
func (s *Store) CreateSession(ctx context.Context, hash, id string, expires time.Time) error {
	_, err := s.Pool.Exec(ctx, "INSERT INTO user_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hash, id, expires)
	if err != nil {
		return dependency(err)
	}
	return nil
}
func (s *Store) Session(ctx context.Context, hash string) (auth.Session, error) {
	var result auth.Session
	err := s.Pool.QueryRow(ctx, "SELECT u.user_id,u.username,u.display_name,s.expires_at FROM user_sessions s JOIN app_users u USING(user_id) WHERE token_hash=$1 AND expires_at>now() AND NOT u.disabled", hash).Scan(&result.User.ID, &result.User.Username, &result.User.DisplayName, &result.Expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, d.Fail(401, "UNAUTHORIZED", "Сессия истекла; войдите снова.")
	}
	if err != nil {
		return result, dependency(err)
	}
	result.Authenticated = true
	result.Expires = result.Expires.UTC()
	return result, nil
}
func (s *Store) RevokeSession(ctx context.Context, hash string) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM user_sessions WHERE token_hash=$1", hash)
	if err != nil {
		return dependency(err)
	}
	return nil
}
