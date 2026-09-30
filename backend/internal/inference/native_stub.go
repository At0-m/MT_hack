//go:build !onnx

package inference

import (
	"fmt"
	d "tramflow/internal/domain"
)

func nativeOpen(path string, model d.Model) (Predictor, error) {
	return nil, fmt.Errorf("build with -tags onnx and CGO_ENABLED=1 to load a real ONNX model")
}
