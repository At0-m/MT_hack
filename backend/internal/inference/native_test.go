//go:build onnx

package inference

import (
	"context"
	"encoding/json"
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
