package inference

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
	d "tramflow/internal/domain"
)

type blockingPredictor struct {
	started chan struct{}
	finish  chan struct{}
	closed  atomic.Bool
}

func (p *blockingPredictor) Predict(context.Context, [][]float32) ([]float32, error) {
	close(p.started)
	<-p.finish
	if p.closed.Load() {
		return nil, fmt.Errorf("closed during Run")
	}
	return []float32{1}, nil
}
func (p *blockingPredictor) Close() error { p.closed.Store(true); return nil }

func TestSessionReferencesProtectNativeRun(t *testing.T) {
	m := New(t.TempDir())
	model := d.Model{Version: "pinned"}
	raw, _ := json.Marshal(model)
	p := &blockingPredictor{started: make(chan struct{}), finish: make(chan struct{})}
	m.sessions[model.Version] = &modelSession{predictor: p, manifest: sha256.Sum256(raw)}
	for i := 1; i < sessionLimit; i++ {
		m.sessions[fmt.Sprint(i)] = &modelSession{refs: 1}
	}
	h := &modelHandle{manager: m, model: model}
	done := make(chan error, 1)
	go func() { _, err := h.Predict(context.Background(), [][]float32{{1}}); done <- err }()
	<-p.started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := m.acquire(ctx, d.Model{Version: "next"}); err == nil {
		t.Fatal("all-busy cache should wait")
	}
	if p.closed.Load() {
		t.Fatal("in-flight predictor was evicted")
	}
	close(p.finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	if m.sessions[model.Version].refs != 0 {
		t.Fatal("reference not released")
	}
	// Remove artificial entries without owned native predictors.
	for version := range m.sessions {
		if version != model.Version {
			delete(m.sessions, version)
		}
	}
	m.mu.Unlock()
	m.Close()
	if !p.closed.Load() {
		t.Fatal("predictor not closed")
	}
}
