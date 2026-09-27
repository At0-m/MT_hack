package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	d "tramflow/internal/domain"
)

func (s *Server) problem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":       "urn:tramflow:problem:" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		"title":      http.StatusText(status),
		"status":     status,
		"code":       code,
		"detail":     detail,
		"request_id": w.Header().Get("X-Request-ID"),
	})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	var e *d.Error
	if errors.As(err, &e) {
		if e.Status == 429 {
			w.Header().Set("Retry-After", "1")
			if e.Code == "LOGIN_RATE_LIMIT" {
				w.Header().Set("Retry-After", "60")
			}
		}
		if e.Cause != nil {
			slog.Error("request dependency failed", "request_id", w.Header().Get("X-Request-ID"), "operation", e.Operation, "code", e.Code, "error", e.Cause)
		}
		s.problem(w, e.Status, e.Code, e.Detail)
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		s.problem(w, 504, "TIMEOUT", "Время ожидания истекло.")
		return
	}
	slog.Error("request failed", "request_id", w.Header().Get("X-Request-ID"), "error", err)
	s.problem(w, 503, "DEPENDENCY_UNAVAILABLE", "Зависимость сервиса недоступна.")
}

func (s *Server) write(w http.ResponseWriter, r *http.Request, value any, cache bool) {
	b, err := json.Marshal(value)
	if err != nil {
		s.fail(w, err)
		return
	}
	if cache {
		hash := sha256.Sum256(b)
		tag := `"` + hex.EncodeToString(hash[:]) + `"`
		w.Header().Set("ETag", tag)
		w.Header().Set("Cache-Control", "private,no-cache")
		for _, t := range strings.Split(r.Header.Get("If-None-Match"), ",") {
			if strings.TrimSpace(t) == tag || strings.TrimSpace(t) == "*" {
				w.WriteHeader(304)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}
