package inference

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/telemetry"
)

// Conservative admission cap; native working memory still depends on the model.
const sessionLimit = 4

type modelLoad struct {
	manifest [32]byte
	done     chan struct{}
	err      error
}

type modelSession struct {
	predictor Predictor
	manifest  [32]byte
	refs      int
	used      uint64
}

type modelHandle struct {
	manager *Manager
	model   d.Model
}

func (h *modelHandle) Close() error { return nil } // Manager owns cached sessions.
func (h *modelHandle) Predict(ctx context.Context, rows [][]float32) ([]float32, error) {
	h.manager.life.RLock()
	defer h.manager.life.RUnlock()
	if h.manager.closed {
		return nil, fmt.Errorf("model manager closed")
	}
	return h.manager.run(ctx, h.model, rows)
}

// Called while holding life.RLock. A reference covers both queued and native Run.
func (m *Manager) run(ctx context.Context, model d.Model, rows [][]float32) ([]float32, error) {
	session, err := m.acquire(ctx, model)
	if err != nil {
		return nil, err
	}
	defer m.release(session)
	waitStarted := time.Now()
	select {
	case m.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	telemetry.Default.Observe("onnx_admission_wait_seconds", nil, time.Since(waitStarted))
	defer func() { <-m.slots }()
	// Cooperative termination requests do not release buffers/session before Run exits.
	done := telemetry.Timer("onnx_inference_duration_seconds", nil)
	defer done()
	return session.predictor.Predict(ctx, rows)
}

func (m *Manager) acquire(ctx context.Context, model d.Model) (*modelSession, error) {
	raw, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		m.mu.Lock()
		if session, ok := m.sessions[model.Version]; ok {
			if session.manifest != digest {
				m.mu.Unlock()
				return nil, fmt.Errorf("cached model manifest changed")
			}
			m.clock++
			session.used, session.refs = m.clock, session.refs+1
			m.mu.Unlock()
			return session, nil
		}
		if loading, ok := m.loading[model.Version]; ok {
			m.mu.Unlock()
			if loading.manifest != digest {
				return nil, fmt.Errorf("loading model manifest changed")
			}
			select {
			case <-loading.done:
				if loading.err != nil {
					return nil, loading.err
				}
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		var victim string
		var oldest *modelSession
		if len(m.sessions)+len(m.loading)+len(m.retired) >= sessionLimit {
			for version, session := range m.sessions {
				if session.refs == 0 && (oldest == nil || session.used < oldest.used) {
					victim, oldest = version, session
				}
			}
			if oldest == nil {
				changed := m.changed
				m.mu.Unlock()
				select {
				case <-changed:
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			delete(m.sessions, victim)
		}
		loading := &modelLoad{manifest: digest, done: make(chan struct{})}
		m.loading[model.Version] = loading // Reserves one slot, including victim Close.
		m.mu.Unlock()

		started := time.Now()
		var predictor Predictor
		closeFailed := false
		if oldest != nil {
			err = oldest.predictor.Close()
			closeFailed = err != nil
		}
		if err == nil {
			predictor, err = m.open(ctx, model)
		}
		m.mu.Lock()
		delete(m.loading, model.Version)
		var session *modelSession
		if err == nil {
			m.clock++
			session = &modelSession{predictor: predictor, manifest: digest, refs: 1, used: m.clock}
			m.sessions[model.Version] = session
		} else if closeFailed {
			m.retired = append(m.retired, oldest)
		}
		loading.err = err
		close(loading.done)
		close(m.changed)
		m.changed = make(chan struct{})
		count := len(m.sessions)
		m.mu.Unlock()
		slog.Info("model load", "model_version", model.Version, "duration_ms", time.Since(started).Milliseconds(), "session_count", count, "evicted_version", victim, "success", err == nil)
		return session, err
	}
}

func (m *Manager) release(session *modelSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session.refs--
	m.clock++
	session.used = m.clock
	close(m.changed)
	m.changed = make(chan struct{})
}
