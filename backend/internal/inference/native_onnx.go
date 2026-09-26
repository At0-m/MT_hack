//go:build onnx

package inference

import (
	"context"
	"fmt"
	ort "github.com/yalue/onnxruntime_go"
	"os"
	"sync"
	d "tramflow/internal/domain"
)

var initOnce sync.Once
var initErr error

type native struct {
	session *ort.DynamicAdvancedSession
	columns int
}

func nativeOpen(path string, model d.Model) (Predictor, error) {
	initOnce.Do(func() {
		ort.SetSharedLibraryPath(os.Getenv("ONNX_LIBRARY_PATH"))
		initErr = ort.InitializeEnvironment()
	})
	if initErr != nil {
		return nil, initErr
	}
	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	defer opts.Destroy()
	if err = opts.SetIntraOpNumThreads(1); err != nil {
		return nil, err
	}
	if err = opts.SetInterOpNumThreads(1); err != nil {
		return nil, err
	}
	s, err := ort.NewDynamicAdvancedSession(path, []string{model.Input}, []string{model.Output}, opts)
	if err != nil {
		return nil, err
	}
	return &native{s, len(model.Columns)}, nil
}
func (n *native) Predict(ctx context.Context, rows [][]float32) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []float32{}, nil
	}
	flat := make([]float32, 0, len(rows)*n.columns)
	for _, r := range rows {
		if len(r) != n.columns {
			return nil, fmt.Errorf("feature width mismatch")
		}
		flat = append(flat, r...)
	}
	input, err := ort.NewTensor(ort.NewShape(int64(len(rows)), int64(n.columns)), flat)
	if err != nil {
		return nil, err
	}
	defer input.Destroy()
	output, err := ort.NewEmptyTensor[float32](ort.NewShape(int64(len(rows)), 1))
	if err != nil {
		return nil, err
	}
	defer output.Destroy()
	if err = n.session.Run([]ort.Value{input}, []ort.Value{output}); err != nil {
		return nil, err
	}
	return append([]float32(nil), output.GetData()...), nil
}
func (n *native) Close() error { return n.session.Destroy() }
