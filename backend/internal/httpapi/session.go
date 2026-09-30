package httpapi

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
	"tramflow/internal/auth"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
)

type loginAttempt struct {
	Start time.Time
	Count int
}

func (s *Server) allowLogin(address string) bool {
	if address == "" {
		return false
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	now := time.Now()
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	a := s.loginAttempts[host]
	if now.Sub(a.Start) >= time.Minute {
		a = loginAttempt{Start: now}
	}
	if a.Count >= 60 {
		return false
	}
	if len(s.loginAttempts) >= 4096 {
		for k, v := range s.loginAttempts {
			if now.Sub(v.Start) >= time.Minute {
				delete(s.loginAttempts, k)
			}
		}
		if len(s.loginAttempts) >= 4096 {
			return false
		}
	}
	a.Count++
	s.loginAttempts[host] = a
	return true
}
func (s *Server) cookie(w http.ResponseWriter, value string, expires time.Time, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "tramflow_session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		Expires:  expires,
		MaxAge:   maxAge,
	})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	outcome, subject := "invalid_request", ""
	defer func() {
		slog.Info("auth login", "request_id", w.Header().Get("X-Request-ID"), "outcome", outcome, "subject_hash", subject, "client_hash", auth.Hash(s.clientIP(r)))
	}()
	if !s.allowLogin(s.clientIP(r)) {
		outcome = "ip_limited"
		w.Header().Set("Retry-After", "60")
		s.problem(w, 429, "LOGIN_RATE_LIMIT", "Слишком много попыток входа.")
		return
	}
	select {
	case s.loginSlots <- struct{}{}:
		defer func() { <-s.loginSlots }()
	default:
		outcome = "busy"
		s.problem(w, 429, "LOGIN_BUSY", "Повторите вход позже.")
		return
	}
	var q struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !s.body(w, r, "LoginRequest", &q) {
		return
	}
	subject = auth.Hash(q.Username)
	session, token, err := s.Auth.Login(r.Context(), q.Username, q.Password)
	if err != nil {
		outcome = "error"
		var problem *d.Error
		if errors.As(err, &problem) {
			switch problem.Code {
			case "INVALID_CREDENTIALS":
				outcome = "invalid"
			case "INVALID_CREDENTIALS_LIMIT":
				outcome = "account_limited"
			case "LOGIN_RATE_LIMIT":
				outcome = "global_limited"
			}
		}
		s.fail(w, err)
		return
	}
	outcome = "success"
	s.cookie(w, token, session.Expires, 43200)
	s.write(w, r, session, false)
}
func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("tramflow_session")
	session, err := s.Auth.Resolve(r.Context(), cookie.Value)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, session, false)
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("tramflow_session")
	if err := s.Auth.Logout(r.Context(), cookie.Value); err != nil {
		s.fail(w, err)
		return
	}
	s.cookie(w, "", time.Unix(0, 0), -1)
	w.WriteHeader(204)
}
func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	var q d.SummaryQuery
	if !s.body(w, r, "SummaryQuery", &q) {
		return
	}
	result, err := s.Service.Calculate(r.Context(), q.Calculation)
	if err != nil {
		s.fail(w, err)
		return
	}
	if result.ID != q.Expected {
		s.problem(w, 409, "CALCULATION_MISMATCH", "Расчёт не совпадает с подтверждённым идентификатором.")
		return
	}
	s.write(w, r, engine.Summarize(result, q.Focus), false)
}
