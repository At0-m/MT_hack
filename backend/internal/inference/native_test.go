//go:build onnx

package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	d "tramflow/internal/domain"
)

func TestNativeGoldenAndConcurrentBuffers(t *testing.T) {
	b, err := os.ReadFile("../../testdata/native-model.json")
	if os.IsNotExist(err) {
		t.Skip("generate native test model with go run ./cmd/fixtures --onnx")
	}
	if err != nil {
		t.Fatal(err)
	}
	var model d.Model
	if err = json.Unmarshal(b, &model); err != nil {
		t.Fatal(err)
	}
	m := New("../../artifacts")
	defer m.Close()
	p, err := m.Prepare(context.Background(), model, false)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				v := float32(i*25 + j)
				out, err := p.Predict(context.Background(), [][]float32{{v, 1}, {0, v}})
				if err != nil || len(out) != 2 || out[0] != v+1 || out[1] != v {
					t.Errorf("out=%v err=%v", out, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestNativeSessionEvictionAndReload(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/native-model.json")
	if err != nil {
		t.Fatal(err)
	}
	var model d.Model
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	m := New("../../artifacts")
	defer m.Close()
	old, err := m.Prepare(context.Background(), model, false)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 24; i++ {
				version := model
				version.Version = fmt.Sprintf("native-eviction-%d-%d", worker, i)
				p, err := m.Prepare(context.Background(), version, false)
				if err != nil {
					t.Error(err)
					return
				}
				out, err := p.Predict(context.Background(), [][]float32{{3, 4}})
				if err != nil || len(out) != 1 || out[0] != 7 {
					t.Errorf("out=%v err=%v", out, err)
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	if len(m.sessions) > sessionLimit {
		t.Fatal("session limit exceeded")
	}
	if out, err := old.Predict(context.Background(), [][]float32{{5, 6}}); err != nil || len(out) != 1 || out[0] != 11 {
		t.Fatal("old manifest cannot reload", out, err)
	}
	m.Close()
	if _, err := old.Predict(context.Background(), [][]float32{{5, 6}}); err == nil {
		t.Fatal("closed manager accepted escaped handle")
	}
}
