package inference

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	d "tramflow/internal/domain"
)

type Predictor interface {
	Predict(context.Context, [][]float32) ([]float32, error)
	Close() error
}
type Golden struct {
	Rows     [][]float32 `json:"rows"`
	Expected []float32   `json:"expected"`
}
type Schema struct {
	Version string   `json:"version"`
	Columns []string `json:"columns"`
}
type Manager struct {
	life     sync.RWMutex
	closed   bool
	Root     string
	mu       sync.Mutex
	sessions map[string]*modelSession
	clock    uint64
	changed  chan struct{}
	slots    chan struct{}
}

func New(root string) *Manager {
	return &Manager{Root: root, sessions: map[string]*modelSession{}, changed: make(chan struct{}), slots: make(chan struct{}, 2)}
}
func (m *Manager) file(path, hash string) (string, error) {
	root, err := filepath.Abs(m.Root)
	if err != nil {
		return "", err
	}
	p := filepath.Join(root, filepath.FromSlash(path))
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("artifact outside root")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != hash {
		return "", fmt.Errorf("artifact checksum mismatch")
	}
	return p, nil
}
func (m *Manager) Prepare(ctx context.Context, model d.Model, synthetic bool) (Predictor, error) {
	m.life.RLock()
	defer m.life.RUnlock()
	if m.closed {
		return nil, fmt.Errorf("model manager closed")
	}
	return m.prepare(ctx, model, synthetic)
}
func (m *Manager) prepare(ctx context.Context, model d.Model, synthetic bool) (Predictor, error) {
	if synthetic {
		if model.Release != "synthetic" {
			return nil, fmt.Errorf("synthetic model required")
		}
		return nil, nil
	}
	session, err := m.acquire(ctx, model)
	if err != nil {
		return nil, err
	}
	m.release(session)
	// A handle pins the manifest, not the native pointer; it can reload after eviction.
	return &modelHandle{manager: m, model: model}, nil
}

func (m *Manager) load(ctx context.Context, model d.Model) (Predictor, error) {
	if model.Release != "published" || model.Input != "features" || model.Output != "boardings" || len(model.Columns) < 1 || len(model.Columns) > 256 {
		return nil, fmt.Errorf("invalid model manifest")
	}
	path, err := m.file(model.Path, model.SHA256)
	if err != nil {
		return nil, err
	}
	schemaPath, err := m.file(model.SchemaPath, model.SchemaSHA256)
	if err != nil {
		return nil, err
	}
	goldenPath, err := m.file(model.GoldenPath, model.GoldenSHA256)
	if err != nil {
		return nil, err
	}
	var schema Schema
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &schema); err != nil {
		return nil, err
	}
	if schema.Version != model.Schema || strings.Join(schema.Columns, "\x00") != strings.Join(model.Columns, "\x00") {
		return nil, fmt.Errorf("feature schema mismatch")
	}
	p, err := nativeOpen(path, model)
	if err != nil {
		return nil, err
	}
	good := false
	defer func() {
		if !good {
			_ = p.Close()
		}
	}()
	var g Golden
	b, err = os.ReadFile(goldenPath)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &g); err != nil || len(g.Rows) < 1 || len(g.Rows) != len(g.Expected) {
		return nil, fmt.Errorf("invalid golden vectors")
	}
	select {
	case m.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	out, err := p.Predict(ctx, g.Rows)
	<-m.slots
	if err != nil || len(out) != len(g.Expected) {
		return nil, fmt.Errorf("golden inference failed: %v", err)
	}
	for i, v := range out {
		want := float64(g.Expected[i])
		actual, err := Postprocess(float64(v), model.Postprocessing)
		if err != nil || math.IsNaN(want) || math.IsInf(want, 0) || math.Abs(actual-want) > 1e-4+1e-5*math.Abs(want) {
			return nil, fmt.Errorf("golden parity failed at row %d", i)
		}
	}
	good = true
	return p, nil
}
func Postprocess(v float64, policy string) (float64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("nonfinite output")
	}
	switch policy {
	case "identity":
	case "clamp_zero":
		v = math.Max(0, v)
	case "expm1_clamp_zero":
		v = math.Max(0, math.Expm1(v))
	default:
		return 0, fmt.Errorf("unknown postprocessing")
	}
	if v < 0 || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid processed output")
	}
	return v, nil
}
func (m *Manager) Predict(ctx context.Context, model d.Model, s d.Snapshot, hours []d.Hour) ([]float64, error) {
	m.life.RLock()
	defer m.life.RUnlock()
	if m.closed {
		return nil, fmt.Errorf("model manager closed")
	}
	out := make([]float64, len(hours))
	synthetic := s.Provenance.Mode == "synthetic_mock"
	if synthetic {
		for i, h := range hours {
			if h.SyntheticBoardings == nil {
				return nil, fmt.Errorf("missing synthetic prediction")
			}
			out[i] = *h.SyntheticBoardings
		}
		return out, nil
	}
	rows := make([][]float32, len(hours))
	zone := time.FixedZone("Europe/Moscow", 10800)
	for i, h := range hours {
		if h.FeaturesAvailableAt.After(s.Provenance.Origin) {
			return nil, fmt.Errorf("future feature availability")
		}
		rows[i] = make([]float32, len(model.Columns))
		for j, name := range model.Columns {
			local := h.Time.In(zone)
			v, ok := h.Features[name]
			switch name {
			case "hour":
				v = float64(local.Hour())
				ok = true
			case "weekday":
				v = float64((int(local.Weekday()) + 6) % 7)
				ok = true
			case "month":
				v = float64(local.Month())
				ok = true
			case "lead_hours":
				v = h.Time.Sub(s.Provenance.Origin).Hours()
				ok = true
			}
			if !ok || math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > math.MaxFloat32 {
				return nil, fmt.Errorf("missing or invalid feature %s", name)
			}
			rows[i][j] = float32(v)
		}
	}
	raw, err := m.run(ctx, model, rows)
	if err != nil {
		return nil, err
	}
	if len(raw) != len(hours) {
		return nil, fmt.Errorf("wrong output count")
	}
	for i, v := range raw {
		out[i], err = Postprocess(float64(v), model.Postprocessing)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (m *Manager) Close() {
	m.life.Lock()
	defer m.life.Unlock()
	m.closed = true
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := 0; i < cap(m.slots); i++ {
		m.slots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(m.slots); i++ {
			<-m.slots
		}
	}()
	for _, session := range m.sessions {
		_ = session.predictor.Close()
	}
	m.sessions = map[string]*modelSession{}
}
