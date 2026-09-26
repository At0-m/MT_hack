package inference

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	d "tramflow/internal/domain"
)

const sessionLimit = 16

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
	select {
	case m.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-m.slots }()
	// Cooperative termination requests do not release buffers/session before Run exits.
	return session.predictor.Predict(ctx, rows)
}

func (m *Manager) acquire(ctx context.Context, model d.Model) (*modelSession, error) {
	raw, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	for {
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
		if len(m.sessions) >= sessionLimit {
			var victim string
			var oldest *modelSession
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
			if err := oldest.predictor.Close(); err != nil {
				m.mu.Unlock()
				return nil, err
			}
			delete(m.sessions, victim)
		}
		// Serial loading prevents duplicate sessions. Runs release their slot before
		// acquiring mu, so golden inference here cannot deadlock with a running model.
		predictor, err := m.load(ctx, model)
		if err != nil {
			m.mu.Unlock()
			return nil, err
		}
		m.clock++
		session := &modelSession{predictor: predictor, manifest: digest, refs: 1, used: m.clock}
		m.sessions[model.Version] = session
		m.mu.Unlock()
		return session, nil
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
