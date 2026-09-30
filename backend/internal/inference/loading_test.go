package inference

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	d "tramflow/internal/domain"
)

type idlePredictor struct{}

func (*idlePredictor) Predict(context.Context, [][]float32) ([]float32, error) {
	return []float32{1}, nil
}

func TestArtifactLimitUsesRoleNotFilenameExtension(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "schema-disguised.onnx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(metadataArtifactLimit + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	manager := New(root)
	defer manager.Close()
	if _, err := manager.file("schema-disguised.onnx", "unused", metadataArtifactLimit); err == nil {
		t.Fatal("metadata exceeds limit using model filename extension")
	}
}

type slowClosePredictor struct {
	idlePredictor
	started chan struct{}
	finish  chan struct{}
}

func (p *slowClosePredictor) Close() error {
	close(p.started)
	<-p.finish
	return nil
}

func TestEvictionCloseDoesNotBlockOtherCachedModels(t *testing.T) {
	manager := New(t.TempDir())
	defer manager.Close()
	victim := &slowClosePredictor{started: make(chan struct{}), finish: make(chan struct{})}
	cachedModel(manager, d.Model{Version: "victim"}, victim)
	for index := 1; index < sessionLimit; index++ {
		version := fmt.Sprint(index)
		cachedModel(manager, d.Model{Version: version}, &idlePredictor{})
		manager.sessions[version].refs = 1
	}
	manager.open = func(context.Context, d.Model) (Predictor, error) { return &idlePredictor{}, nil }
	done := make(chan error, 1)
	go func() {
		session, err := manager.acquire(context.Background(), d.Model{Version: "new"})
		if err == nil {
			manager.release(session)
		}
		done <- err
	}()
	<-victim.started
	progress := make(chan error, 1)
	go func() {
		session, err := manager.acquire(context.Background(), d.Model{Version: "1"})
		if err == nil {
			manager.release(session)
		}
		progress <- err
	}()
	select {
	case err := <-progress:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(time.Second):
		t.Error("cached model blocked by victim Close")
	}
	close(victim.finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func (*idlePredictor) Close() error { return nil }

func cachedModel(manager *Manager, model d.Model, predictor Predictor) {
	raw, _ := json.Marshal(model)
	manager.sessions[model.Version] = &modelSession{manifest: sha256.Sum256(raw), predictor: predictor}
}

func TestLoadingDoesNotBlockCachedModelsAndCoalescesVersion(t *testing.T) {
	manager := New(t.TempDir())
	defer manager.Close()
	cached := d.Model{Version: "cached"}
	cachedModel(manager, cached, &idlePredictor{})
	started, finish := make(chan struct{}), make(chan struct{})
	var opens atomic.Int32
	manager.open = func(ctx context.Context, model d.Model) (Predictor, error) {
		opens.Add(1)
		close(started)
		<-finish
		return &idlePredictor{}, nil
	}
	done := make(chan error, 1)
	go func() {
		session, err := manager.acquire(context.Background(), d.Model{Version: "loading"})
		if err == nil {
			manager.release(session)
		}
		done <- err
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	// This deadline would not be observed by the old implementation stuck on mu.
	progress := make(chan error, 1)
	go func() {
		session, err := manager.acquire(ctx, cached)
		if err == nil {
			manager.release(session)
		}
		progress <- err
	}()
	select {
	case err := <-progress:
		if err != nil {
			t.Error(err)
		}
	case <-ctx.Done():
		t.Error("unrelated cached model blocked by load")
	}
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer waitCancel()
	if _, err := manager.acquire(waitCtx, d.Model{Version: "loading"}); err == nil {
		t.Error("waiting loader ignored cancellation")
	}
	close(finish)
	if err := <-done; err != nil || opens.Load() != 1 {
		t.Fatal("duplicate version load", opens.Load(), err)
	}
}
