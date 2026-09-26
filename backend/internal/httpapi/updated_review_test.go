package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"tramflow/internal/auth"
	d "tramflow/internal/domain"
	"tramflow/internal/inference"
)

func TestHealthBypassesFullApplicationGate(t *testing.T) {
	f, c := fixture(t)
	m := inference.New("../../artifacts")
	defer m.Close()
	s := &Server{Store: f, Contract: c, Models: m, Auth: auth.New(newTestAuth())}
	h := s.Handler()
	for range cap(s.slots) {
		s.slots <- struct{}{}
	}
	defer func() {
		for range cap(s.slots) {
			<-s.slots
		}
	}()
	for _, path := range []string{"/health/live", "/health/ready"} {
		if response := req(h, "GET", path, "", false); response.Code != 200 {
			t.Fatal(path, response.Code)
		}
	}
	if response := req(h, "GET", "/api/v1/bootstrap", "", true); response.Code != 429 {
		t.Fatal(response.Code)
	}
}

func TestForwardedAddressRequiresTrustedPeer(t *testing.T) {
	s := &Server{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}}
	for _, test := range []struct{ peer, chain, want string }{
		{"192.0.2.1:123", "203.0.113.2", "192.0.2.1"},
		{"10.0.0.1:123", "203.0.113.2, 10.0.0.2", "203.0.113.2"},
		{"10.0.0.1:123", "198.51.100.3, 203.0.113.2", "203.0.113.2"},
		{"10.0.0.1:123", "", ""},
	} {
		r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
		r.RemoteAddr = test.peer
		r.Header.Set("X-Forwarded-For", test.chain)
		if got := s.clientIP(r); got != test.want {
			t.Fatal(got, test.want)
		}
	}
}

func TestDependencyCauseIsLoggedButNotExposed(t *testing.T) {
	var log bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&log, nil)))
	defer slog.SetDefault(old)
	w := httptest.NewRecorder()
	w.Header().Set("X-Request-ID", "review-request")
	cause := errors.New("internal diagnostic marker")
	err := d.Caused(503, "DATABASE_UNAVAILABLE", "Хранилище недоступно.", "postgres", cause)
	if !errors.Is(err, cause) {
		t.Fatal("lost private cause")
	}
	(&Server{}).fail(w, err)
	if strings.Contains(w.Body.String(), cause.Error()) || !strings.Contains(log.String(), cause.Error()) || !strings.Contains(log.String(), "review-request") {
		t.Fatal("cause logging/exposure mismatch")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("wrong error identity")
	}
}
