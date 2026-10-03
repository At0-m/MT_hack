package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/netip"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"tramflow/internal/auth"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
	"tramflow/internal/inference"
	"tramflow/internal/telemetry"
)

type Store interface {
	WeatherMetadata(context.Context, string) (d.WeatherSnapshot, error)
	engine.Repository
	Active(context.Context) (d.Snapshot, d.Model, error)
	CheckSchema(context.Context) error
	Routes(context.Context, string) ([]json.RawMessage, error)
	Route(context.Context, string, string) (json.RawMessage, error)
	Geometry(context.Context, string, string) (json.RawMessage, error)
	Report(context.Context, string, string) (json.RawMessage, error)
}
type Server struct {
	TrustedProxies []netip.Prefix
	Store          Store
	Service        *engine.Service
	Contract       *contract.Contract
	Models         *inference.Manager
	Auth           *auth.Service
	Origin         string
	CookieSecure   bool
	loginMu        sync.Mutex
	loginAttempts  map[string]loginAttempt
	loginSlots     chan struct{}
	slots          chan struct{}
	readySlots     chan struct{}
	Timeout        time.Duration
}

func (s *Server) Handler() http.Handler {
	s.slots = make(chan struct{}, 32)
	s.readySlots = make(chan struct{}, 2)
	s.loginSlots = make(chan struct{}, 2)
	s.loginAttempts = map[string]loginAttempt{}
	if s.Timeout == 0 {
		s.Timeout = 10 * time.Second
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		s.write(w, r, map[string]any{"status": "ok", "component": "api", "checks": []any{}}, false)
	})
	mux.HandleFunc("GET /health/ready", s.ready)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/auth/session", s.session)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/bootstrap", s.bootstrap)
	mux.HandleFunc("GET /api/v1/snapshots/{id}", s.snapshot)
	mux.HandleFunc("GET /api/v1/routes", s.routes)
	mux.HandleFunc("GET /api/v1/routes/{id}", s.route)
	mux.HandleFunc("GET /api/v1/routes/{id}/geometry", s.geometry)
	mux.HandleFunc("POST /api/v1/forecasts/query", s.forecast)
	mux.HandleFunc("POST /api/v1/scenarios/evaluate", s.scenario)
	mux.HandleFunc("GET /api/v1/weather", s.weather)
	mux.HandleFunc("GET /api/v1/model-quality", s.report)
	mux.HandleFunc("GET /api/v1/data-sources", s.report)
	mux.HandleFunc("POST /api/v1/exports", s.export)
	mux.HandleFunc("POST /api/v1/summaries/query", s.summary)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded := &statusWriter{ResponseWriter: w}
		w = recorded
		_, routePattern := mux.Handler(r)
		start := time.Now()
		id := r.Header.Get("X-Request-ID")
		if !engine.IDPattern.MatchString(id) {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		defer func() {
			if panicValue := recover(); panicValue != nil {
				slog.Error("http panic", "request_id", id, "panic", panicValue, "stack", string(debug.Stack()))
				s.problem(w, 500, "INTERNAL_ERROR", "Внутренняя ошибка сервиса.")
			}
			status := recorded.status
			if status == 0 {
				status = http.StatusOK
			}
			elapsed := time.Since(start)
			endpoint := routePattern
			if _, path, ok := strings.Cut(routePattern, " "); ok {
				endpoint = path
			}
			telemetry.Default.HTTP(r.Method, endpoint, status, elapsed)
			slog.Info("http", "request_id", id, "method", r.Method, "route", routePattern, "status", status, "user_id", d.UserID(r.Context()), "duration_ms", elapsed.Milliseconds())
		}()
		if origin := r.Header.Get("Origin"); origin != "" {
			if origin != s.Origin {
				s.problem(w, 403, "ORIGIN_FORBIDDEN", "Источник запроса не разрешён.")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "ETag, X-Calculation-ID, X-Request-ID, Content-Disposition")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, If-None-Match")
				w.WriteHeader(204)
				return
			}
		}
		if r.Method == http.MethodPost && r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			s.problem(w, 403, "CSRF_REJECTED", "Запрос с другого сайта не разрешён.")
			return
		}
		if r.Method == http.MethodGet && (r.URL.Path == "/health/live" || r.URL.Path == "/health/ready") {
			if r.URL.Path == "/health/ready" {
				select {
				case s.readySlots <- struct{}{}:
					defer func() { <-s.readySlots }()
				default:
					s.problem(w, 503, "READINESS_BUSY", "Проверка готовности занята.")
					return
				}
			}
			probeCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			mux.ServeHTTP(w, r.WithContext(probeCtx))
			return
		}
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		default:
			w.Header().Set("Retry-After", "1")
			s.problem(w, 429, "BUSY", "Сервис занят; повторите запрос.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
		defer cancel()
		r = r.WithContext(ctx)
		if !strings.HasPrefix(r.URL.Path, "/health/") && r.URL.Path != "/api/v1/auth/login" {
			cookie, err := r.Cookie("tramflow_session")
			if err != nil || s.Auth == nil {
				s.problem(w, 401, "UNAUTHORIZED", "Требуется вход.")
				return
			}
			session, err := s.Auth.Resolve(ctx, cookie.Value)
			if err != nil {
				s.fail(w, err)
				return
			}
			r = r.WithContext(d.WithUser(ctx, session.User.ID))
		}
		if _, pattern := mux.Handler(r); pattern == "" {
			s.problem(w, 404, "NOT_FOUND", "API endpoint не найден.")
			return
		}
		mux.ServeHTTP(w, r)
	})
}
