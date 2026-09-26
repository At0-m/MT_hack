package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"
	d "tramflow/internal/domain"
)

type User struct {
	ID          string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}
type Session struct {
	Authenticated bool      `json:"authenticated"`
	User          User      `json:"user"`
	Expires       time.Time `json:"expires_at"`
}
type Repository interface {
	User(context.Context, string) (User, string, error)
	CreateSession(context.Context, string, string, time.Time) error
	Session(context.Context, string) (Session, error)
	RevokeSession(context.Context, string) error
}
type Service struct {
	Repo  Repository
	dummy []byte
}

func New(repo Repository) *Service {
	hash, _ := bcrypt.GenerateFromPassword([]byte("invalid-placeholder-password"), bcrypt.DefaultCost)
	return &Service{Repo: repo, dummy: hash}
}
func Hash(token string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(token))) }
func (s *Service) Login(ctx context.Context, username, password string) (Session, string, error) {
	user, hash, err := s.Repo.User(ctx, username)
	if err != nil {
		return Session{}, "", err
	}
	h := []byte(hash)
	if hash == "" {
		h = s.dummy
	}
	check := bcrypt.CompareHashAndPassword(h, []byte(password))
	if check != nil || hash == "" {
		return Session{}, "", d.Fail(401, "INVALID_CREDENTIALS", "Неверный логин или пароль.")
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return Session{}, "", err
	}
	token := hex.EncodeToString(raw)
	expires := time.Now().UTC().Add(12 * time.Hour)
	if err = s.Repo.CreateSession(ctx, Hash(token), user.ID, expires); err != nil {
		return Session{}, "", err
	}
	return Session{true, user, expires}, token, nil
}
func (s *Service) Resolve(ctx context.Context, token string) (Session, error) {
	if len(token) != 64 {
		return Session{}, d.Fail(401, "UNAUTHORIZED", "Требуется вход.")
	}
	if _, err := hex.DecodeString(token); err != nil {
		return Session{}, d.Fail(401, "UNAUTHORIZED", "Требуется вход.")
	}
	return s.Repo.Session(ctx, Hash(token))
}
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.Repo.RevokeSession(ctx, Hash(token))
}
