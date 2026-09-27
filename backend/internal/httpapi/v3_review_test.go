package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"tramflow/internal/auth"
)

type failureAuth struct {
	*testAuth
	failures int
}

func (r *failureAuth) RecordLogin(_ context.Context, _ string, success bool) (bool, error) {
	if success {
		r.failures = 0
		return true, nil
	}
	r.failures++
	return r.failures <= 10, nil
}

func TestAccountFailuresDoNotLockOutCorrectPassword(t *testing.T) {
	repo := &failureAuth{testAuth: newTestAuth()}
	service := auth.New(repo)
	for range 11 {
		if _, _, err := service.Login(context.Background(), "tester", "incorrect"); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if _, token, err := service.Login(context.Background(), "tester", "test-password-long"); err != nil || token == "" || repo.failures != 0 {
		t.Fatal("correct password locked out or failure state not reset", err)
	}
}

func TestInvalidForwardedChainsUseCoarsePeerBudget(t *testing.T) {
	s := &Server{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}, loginAttempts: map[string]loginAttempt{}}
	for _, chain := range []string{"", "malformed", "10.0.0.2", strings.Repeat("10.0.0.2,", 17)} {
		request := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
		request.RemoteAddr = "10.0.0.1:1234"
		request.Header.Set("X-Forwarded-For", chain)
		if got := s.clientIP(request); got != "10.0.0.1" {
			t.Fatal("invalid chain loses coarse peer", got)
		}
	}
	for range 60 {
		if !s.allowLogin("10.0.0.1") {
			t.Fatal("unexpected early IP rejection")
		}
	}
	if s.allowLogin("10.0.0.1") || s.allowLogin("") {
		t.Fatal("coarse or empty bucket allows unlimited spray")
	}
}

func TestHTTPStatusAndAuthOutcomeAreLoggedWithoutSecrets(t *testing.T) {
	handler, _ := handler(t)
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(old)
	response := req(handler, "POST", "/api/v1/auth/login", `{"username":"tester","password":"private-invalid-password"}`, false)
	if response.Code != 401 {
		t.Fatal(response.Code)
	}
	text := logs.String()
	if !strings.Contains(text, `"status":401`) || !strings.Contains(text, `"outcome":"invalid"`) || strings.Contains(text, "private-invalid-password") || strings.Contains(text, `"username":"tester"`) {
		t.Fatal("missing audit fields or exposed secret", text)
	}
}
