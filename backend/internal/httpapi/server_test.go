package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"tramflow/internal/auth"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
	"tramflow/internal/inference"
)

type fixtureStore struct{ b d.Bundle }

func (f fixtureStore) Snapshot(ctx context.Context, id string) (d.Snapshot, d.Model, error) {
	if id != f.b.Snapshot.Provenance.SnapshotID {
		return d.Snapshot{}, d.Model{}, d.Fail(404, "SNAPSHOT_NOT_FOUND", "РќРµ РЅР°Р№РґРµРЅ")
	}
	return f.b.Snapshot, f.b.Model, nil
}
func (f fixtureStore) Active(context.Context) (d.Snapshot, d.Model, error) {
	return f.b.Snapshot, f.b.Model, nil
}
func (f fixtureStore) CheckSchema(context.Context) error { return nil }
func (f fixtureStore) Hours(ctx context.Context, s d.Snapshot, q engine.SelectionInput) ([]d.Hour, error) {
	out := []d.Hour{}
	for _, h := range f.b.Hours {
		for _, r := range q.Routes {
			if h.RouteID == r && !h.Time.Before(q.Window.From) && h.Time.Before(q.Window.To) {
				out = append(out, h)
			}
		}
	}
	return out, nil
}
func (f fixtureStore) Routes(context.Context, string) ([]json.RawMessage, error) {
	out := []json.RawMessage{}
	for _, r := range f.b.Routes {
		var detail map[string]json.RawMessage
		_ = json.Unmarshal(r.Detail, &detail)
		out = append(out, detail["route"])
	}
	return out, nil
}
func (f fixtureStore) Route(context.Context, string, string) (json.RawMessage, error) {
	return f.b.Routes[0].Detail, nil
}
func (f fixtureStore) Geometry(context.Context, string, string) (json.RawMessage, error) {
	return f.b.Routes[0].Geometry, nil
}
func (f fixtureStore) Report(ctx context.Context, id, kind string) (json.RawMessage, error) {
	if kind == "data-sources" {
		return f.b.Sources, nil
	}
	return f.b.Quality, nil
}
func fixture(t *testing.T) (fixtureStore, *contract.Contract) {
	t.Helper()
	b, err := os.ReadFile("../../testdata/demo-bundle.json")
	if err != nil {
		t.Fatal("run go run ./cmd/fixtures --onnx first:", err)
	}
	var data d.Bundle
	if err = json.Unmarshal(b, &data); err != nil {
		t.Fatal(err)
	}
	c, err := contract.Load("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return fixtureStore{data}, c
}
func handler(t *testing.T) (http.Handler, *contract.Contract) {
	t.Helper()
	f, c := fixture(t)
	m := inference.New("../../artifacts")
	t.Cleanup(m.Close)
	svc := &engine.Service{Repo: f, Models: m, Cache: engine.NewCache(65536)}
	s := &Server{
		Store:    f,
		Service:  svc,
		Models:   m,
		Contract: c,
		Auth:     auth.New(newTestAuth()),
		Origin:   "http://localhost:5173",
	}
	return s.Handler(), c
}
func req(h http.Handler, method, path, body string, auth bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth {
		r.AddCookie(&http.Cookie{Name: "tramflow_session", Value: strings.Repeat("a", 64)})
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAllJSONEndpointsMatchContract(t *testing.T) {
	h, c := handler(t)
	tests := []struct{ path, schema string }{
		{"/health/live", "Health"},
		{"/health/ready", "Health"},
		{"/api/v1/bootstrap", "Bootstrap"},
		{"/api/v1/snapshots/demo-september-2026-v1", "Snapshot"},
		{"/api/v1/routes?network_version=synthetic-network-v1", "RouteList"},
		{"/api/v1/routes/demo-01?network_version=synthetic-network-v1", "RouteDetail"},
		{"/api/v1/routes/demo-01/geometry?network_version=synthetic-network-v1", "RouteGeometry"},
		{"/api/v1/model-quality?forecast_snapshot_id=demo-september-2026-v1", "QualityResponse"},
		{"/api/v1/data-sources?forecast_snapshot_id=demo-september-2026-v1", "SourceList"},
		{
			"/api/v1/weather?forecast_snapshot_id=demo-september-2026-v1&route_id=demo-01&from=2026-09-01T15:00:00Z&to=2026-09-01T16:00:00Z",
			"WeatherResponse",
		},
	}
	for _, tc := range tests {
		t.Run(tc.schema, func(t *testing.T) {
			w := req(h, "GET", tc.path, "", true)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			if err := c.Validate(tc.schema, w.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestForecastAndScenarioContractAndGolden(t *testing.T) {
	h, c := handler(t)
	for _, e := range []struct{ example, path string }{{"ForecastRequest", "/api/v1/forecasts/query"}, {"ScenarioRequest", "/api/v1/scenarios/evaluate"}} {
		raw, _ := requestExample(c, e.example)
		w := req(h, "POST", e.path, string(raw), true)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		if err := c.Validate("CalculationResponse", w.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
		var result d.Response
		_ = json.Unmarshal(w.Body.Bytes(), &result)
		if e.example == "ForecastRequest" {
			if result.ID != "calc_bfdb42badb4e5a48ad6ee91a595bf1ce84b0009f20b5cc3c342c6250413401ef" || result.Totals[0].Baseline.Boardings != 14640 {
				t.Fatal(result.ID, result.Totals)
			}
		}
	}
}
func TestExportReplayAndMismatch(t *testing.T) {
	h, c := handler(t)
	raw, _ := requestExample(c, "ForecastRequest")
	w := req(h, "POST", "/api/v1/forecasts/query", string(raw), true)
	var r d.Response
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	q := d.ExportRequest{Calculation: r.Descriptor, Expected: r.ID, Format: "csv"}
	raw, _ = json.Marshal(q)
	w = req(h, "POST", "/api/v1/exports", string(raw), true)
	if w.Code != 200 || w.Header().Get("X-Calculation-ID") != r.ID {
		t.Fatal(w.Code, w.Body.String())
	}
	if !bytes.HasPrefix(w.Body.Bytes(), []byte{0xef, 0xbb, 0xbf}) || !strings.Contains(w.Body.String(), "\r\n") {
		t.Fatal("wrong CSV encoding")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(w.Body.String(), "\ufeff")))
	reader.Comma = ';'
	rows, err := reader.ReadAll()
	if err != nil || len(rows) != 25 || rows[1][5] != "610" {
		t.Fatal(rows, err)
	}
	q.Expected = "calc_" + strings.Repeat("0", 64)
	raw, _ = json.Marshal(q)
	w = req(h, "POST", "/api/v1/exports", string(raw), true)
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
}
func TestAuthenticationAndETag(t *testing.T) {
	h, _ := handler(t)
	path := "/api/v1/routes?network_version=synthetic-network-v1"
	if w := req(h, "GET", path, "", false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := req(h, "GET", path, "", true)
	r := httptest.NewRequest("GET", path, nil)
	r.AddCookie(&http.Cookie{Name: "tramflow_session", Value: strings.Repeat("a", 64)})
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	out := httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 304 || out.Body.Len() != 0 {
		t.Fatal(out.Code)
	}
	r.Header.Del("Cookie")
	out = httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 401 {
		t.Fatal("ETag bypassed auth")
	}
}
func TestInvalidRequests(t *testing.T) {
	h, _ := handler(t)
	for _, b := range []string{`{"selection":null}`, `{"selection":{},"selection":{}}`, `{"selection":{},"x":true}`, `{} {}`} {
		w := req(h, "POST", "/api/v1/forecasts/query", b, true)
		if w.Code != 400 {
			t.Fatal(b, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/api/v1/forecasts/query", strings.NewReader(strings.Repeat("x", 65537)))
	r.AddCookie(&http.Cookie{Name: "tramflow_session", Value: strings.Repeat("a", 64)})
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatal(w.Code)
	}
	r = httptest.NewRequest("POST", "/api/v1/forecasts/query", strings.NewReader("{}"))
	r.AddCookie(&http.Cookie{Name: "tramflow_session", Value: strings.Repeat("a", 64)})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal(w.Code)
	}
}
func TestCORS(t *testing.T) {
	h, _ := handler(t)
	r := httptest.NewRequest("OPTIONS", "/api/v1/forecasts/query", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	r.Header.Set("Origin", "http://untrusted.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
func BenchmarkForecastHTTP(b *testing.B) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	raw, err := os.ReadFile("../../testdata/demo-bundle.json")
	if err != nil {
		b.Fatal(err)
	}
	var bundle d.Bundle
	_ = json.Unmarshal(raw, &bundle)
	c, err := contract.Load("../../openapi/openapi.yaml")
	if err != nil {
		b.Fatal(err)
	}
	f := fixtureStore{bundle}
	m := inference.New("../../artifacts")
	defer m.Close()
	srv := &Server{
		Store:    f,
		Contract: c,
		Models:   m,
		Service:  &engine.Service{Repo: f, Models: m, Cache: engine.NewCache(65536)},
		Auth:     auth.New(newTestAuth()),
	}
	h := srv.Handler()
	payload, _ := requestExample(c, "ForecastRequest")
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := req(h, "POST", "/api/v1/forecasts/query", string(payload), true)
			if w.Code != 200 {
				b.Fatal(fmt.Sprint(w.Code))
			}
		}
	})
}

type testAuth struct {
	mu       sync.Mutex
	hash     string
	sessions map[string]auth.Session
}

func newTestAuth() *testAuth {
	h, _ := bcrypt.GenerateFromPassword([]byte("test-password-long"), bcrypt.MinCost)
	u := auth.User{ID: "tester", Username: "tester", DisplayName: "Tester"}
	return &testAuth{
		hash: string(h),
		sessions: map[string]auth.Session{
			auth.Hash(strings.Repeat("a", 64)): {Authenticated: true, User: u, Expires: time.Now().UTC().Add(time.Hour)},
		},
	}
}
func (t *testAuth) User(ctx context.Context, username string) (auth.User, string, error) {
	if username != "tester" {
		return auth.User{}, "", nil
	}
	return auth.User{ID: "tester", Username: "tester", DisplayName: "Tester"}, t.hash, nil
}
func (t *testAuth) CreateSession(ctx context.Context, hash, id string, expires time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessions[hash] = auth.Session{
		Authenticated: true,
		User:          auth.User{ID: id, Username: "tester", DisplayName: "Tester"},
		Expires:       expires,
	}
	return nil
}
func (t *testAuth) Session(ctx context.Context, hash string) (auth.Session, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.sessions[hash]
	if !ok || time.Now().After(s.Expires) {
		return s, d.Fail(401, "UNAUTHORIZED", "Войдите снова.")
	}
	return s, nil
}
func (t *testAuth) RevokeSession(ctx context.Context, hash string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.sessions, hash)
	return nil
}
func TestLoginSessionLogoutAndSummary(t *testing.T) {
	h, c := handler(t)
	w := req(h, "POST", "/api/v1/auth/login", `{"username":"tester","password":"test-password-long"}`, false)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := c.Validate("AuthSession", w.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal(cookies)
	}
	request := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	request.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	raw, _ := requestExample(c, "ForecastRequest")
	w = req(h, "POST", "/api/v1/forecasts/query", string(raw), true)
	var result d.Response
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if len(result.Frames[0].Routes[0].StopReadings) != 0 || len(result.Frames[0].Routes[0].Indicators) != 5 {
		t.Fatal("missing UI contracts")
	}
	q := d.SummaryQuery{Calculation: result.Descriptor, Expected: result.ID, Focus: d.Focus{Kind: "overall"}}
	raw, _ = json.Marshal(q)
	w = req(h, "POST", "/api/v1/summaries/query", string(raw), true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := c.Validate("SummaryResponse", w.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	q.Expected = "calc_" + strings.Repeat("0", 64)
	raw, _ = json.Marshal(q)
	if w = req(h, "POST", "/api/v1/summaries/query", string(raw), true); w.Code != 409 {
		t.Fatal(w.Code)
	}
	request = httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	request.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	request = httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	request.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 401 {
		t.Fatal("revoked session accepted")
	}
}

// Embedded examples demonstrate optional synthetic stop forecasts. This backend's
// fixture intentionally publishes route-only coverage until the stop model arrives.
func requestExample(c *contract.Contract, name string) ([]byte, error) {
	encoded, err := json.Marshal(c.Spec.Components.Examples[name].Value.Value)
	if err != nil {
		return nil, err
	}
	var query d.Query
	if err = json.Unmarshal(encoded, &query); err != nil {
		return nil, err
	}
	query.Selection.View = "day"
	query.Selection.Resolution = "hour"
	query.Selection.SpatialDetail = "route"
	return json.Marshal(query)
}
