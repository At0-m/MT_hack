// Creates only labelled synthetic fixtures. It never claims model quality.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
)

func ptr(v float64) *float64 { return &v }
func write(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(path, b, 0644); err != nil {
		panic(err)
	}
}
func main() {
	native := flag.Bool("onnx", false, "Generate a tiny synthetic ONNX MatMul model for native integration tests")
	out := flag.String("output", "testdata/demo-bundle.json", "Bundle path")
	flag.Parse()
	c, err := contract.Load("openapi/openapi.yaml")
	if err != nil {
		panic(err)
	}
	origin := time.Date(2026, 8, 31, 21, 0, 0, 0, time.UTC)
	end := origin.Add(720 * time.Hour)
	now := origin
	p := d.Provenance{
		SnapshotID:      "demo-september-2026-v1",
		Origin:          origin,
		CompleteThrough: origin.Add(-2 * time.Hour),
		Model:           "synthetic-no-onnx",
		Features:        "features-v1",
		History:         "synthetic-history-v1",
		Weather:         "synthetic-weather-v1",
		Supply:          "synthetic-supply-v1",
		Reference:       "synthetic-reference-v1",
		Policy:          "scenario-formulas-v1",
		Network:         "synthetic-network-v1",
		Mode:            "synthetic_mock",
	}
	notices := []d.Notice{
		{
			Code:     "SYNTHETIC_MOCK",
			Message:  "Синтетические данные для интеграции; качество модели не измерено.",
			Severity: "warning",
		},
		{
			Code:     "ROUTE_LEVEL_ONLY",
			Message:  "Числовой прогноз по остановкам отсутствует.",
			Severity: "info",
		},
		{
			Code:     "OBSERVED_FLEET_PROXY",
			Message:  "Выпуск — наблюдаемая прокси, индекс не равен заполненности салона.",
			Severity: "warning",
		},
	}
	b := d.Bundle{
		Snapshot: d.Snapshot{
			Provenance:  p,
			Created:     now,
			Coverage:    []d.Coverage{},
			Views:       []string{"day", "week", "month"},
			Quantile:    0.9,
			Target:      0.75,
			Retention:   24,
			Limitations: notices,
		},
		Model: d.Model{
			Version:     p.Model,
			Schema:      p.Features,
			Release:     "synthetic",
			Columns:     []string{"demand_base", "boost"},
			TrainCutoff: p.CompleteThrough,
		},
		Routes: []d.RouteImport{},
		Hours:  []d.Hour{},
	}
	example := c.Spec.Components.Examples["Geometry"].Value.Value
	raw, _ := json.Marshal(example)
	var geo map[string]any
	_ = json.Unmarshal(raw, &geo)
	for n := 1; n <= 10; n++ {
		id := fmt.Sprintf("demo-%02d", n)
		coords := [][]float64{
			{37.596 + float64(n-1)*0.006, 55.75},
			{37.606 + float64(n-1)*0.006, 55.753},
			{37.616 + float64(n-1)*0.006, 55.756},
		}
		pattern := id + "-out"
		stops := []any{}
		features := []any{
			map[string]any{
				"type":       "Feature",
				"id":         pattern,
				"geometry":   map[string]any{"type": "LineString", "coordinates": coords},
				"properties": map[string]any{"kind": "route_line", "route_id": id, "route_pattern_id": pattern},
			},
		}
		for i, xy := range coords {
			stopID := fmt.Sprintf("%s-stop-%d", id, i+1)
			rs := fmt.Sprintf("%s-%d", pattern, i+1)
			name := fmt.Sprintf("Synthetic stop %d", i+1)
			stops = append(stops, map[string]any{
				"route_stop_id": rs,
				"stop_id":       stopID,
				"sequence":      i + 1,
				"name":          name,
				"position":      map[string]any{"longitude": xy[0], "latitude": xy[1]},
			})
			features = append(features, map[string]any{
				"type":     "Feature",
				"id":       rs,
				"geometry": map[string]any{"type": "Point", "coordinates": xy},
				"properties": map[string]any{
					"kind":             "stop",
					"route_id":         id,
					"route_pattern_id": pattern,
					"route_stop_id":    rs,
					"stop_id":          stopID,
					"name":             name,
					"sequence":         i + 1,
				},
			})
		}
		summary := map[string]any{
			"route_id":                id,
			"name":                    "Synthetic route " + id,
			"route_number":            fmt.Sprint(n),
			"geometry_status":         "approximate",
			"representative_position": map[string]any{"longitude": coords[1][0], "latitude": coords[1][1]},
		}
		detail, _ := json.Marshal(map[string]any{
			"route":                   summary,
			"network_version":         p.Network,
			"patterns":                []any{map[string]any{"route_pattern_id": pattern, "name": "Synthetic outbound", "stops": stops}},
			"source_id":               "synthetic-geometry",
			"geometry_observed_at":    now,
			"historical_match_status": "unverified",
			"notices":                 notices,
		})
		geometry, _ := json.Marshal(map[string]any{
			"type":            "FeatureCollection",
			"network_version": p.Network,
			"route_id":        id,
			"source_id":       "synthetic-geometry",
			"features":        features,
		})
		b.Routes = append(b.Routes, d.RouteImport{Detail: detail, Geometry: geometry})
		b.Snapshot.Coverage = append(b.Snapshot.Coverage, d.Coverage{
			RouteID:     id,
			Window:      d.Window{From: origin, To: end},
			Resolutions: []string{"hour", "day"},
			Scopes:      []string{"route"},
			StopStatus:  "unavailable",
			MaxLead:     720,
			Status:      "experimental",
		})
		for t := origin; t.Before(end); t = t.Add(time.Hour) {
			b.Hours = append(b.Hours, d.Hour{
				RouteID:             id,
				Time:                t,
				Features:            map[string]float64{"demand_base": 600, "boost": 10},
				FeaturesAvailableAt: p.CompleteThrough,
				Fleet:               ptr(10),
				Reference:           ptr(65),
				Typical:             ptr(549),
				Source:              "observed_vehicle_profile",
				Proxy:               true,
				SyntheticBoardings:  ptr(610),
				Weather: d.WeatherPoint{
					Window:        d.Window{From: t, To: t.Add(time.Hour)},
					Mode:          "forecast",
					Temperature:   ptr(9.8),
					Precipitation: ptr(0.4),
					Code:          new(int),
					RunAt:         &now,
					AvailableAt:   &now,
					Verified:      true,
				},
			})
		}
	}
	b.Quality, _ = json.Marshal(map[string]any{
		"forecast_snapshot_id": p.SnapshotID,
		"target":               "boardings",
		"metrics":              []any{},
		"source_effects":       []any{},
		"limitations":          []string{"Synthetic fixtures; WAPE not measured."},
	})
	b.Sources, _ = json.Marshal(map[string]any{
		"forecast_snapshot_id": p.SnapshotID,
		"sources": []any{
			map[string]any{
				"source_id":        "synthetic-geometry",
				"name":             "Generated synthetic fixtures",
				"category":         "geometry",
				"retrieval_method": "go run ./cmd/fixtures",
				"version":          p.Network,
				"license_note":     "Test data only",
				"retrieved_at":     now,
				"used_for_model":   false,
				"limitations":      []string{"Not real transport data"},
			},
		},
	})
	if *native {
		baseModel := b.Model
		root := "artifacts/synthetic-matmul-v1"
		_ = os.MkdirAll(root, 0755)
		model := tinyModel()
		if err = os.WriteFile(filepath.Join(root, "boardings.onnx"), model, 0644); err != nil {
			panic(err)
		}
		write(filepath.Join(root, "feature_schema.json"), map[string]any{"version": p.Features, "columns": b.Model.Columns})
		write(filepath.Join(root, "golden_vectors.json"), map[string]any{"rows": [][]float32{{600, 10}, {0, 0}, {-1, 2}}, "expected": []float32{610, 0, 1}})
		b.Model.Version = "synthetic-matmul-v1"
		b.Model.Release = "published"
		b.Model.Input = "features"
		b.Model.Output = "boardings"
		b.Model.Postprocessing = "identity"
		for _, file := range []string{"boardings.onnx", "feature_schema.json", "golden_vectors.json"} {
			path := filepath.Join(root, file)
			content, _ := os.ReadFile(path)
			hash := fmt.Sprintf("%x", sha256.Sum256(content))
			rel := filepath.ToSlash(filepath.Join("synthetic-matmul-v1", file))
			switch file {
			case "boardings.onnx":
				b.Model.Path = rel
				b.Model.SHA256 = hash
			case "feature_schema.json":
				b.Model.SchemaPath = rel
				b.Model.SchemaSHA256 = hash
			case "golden_vectors.json":
				b.Model.GoldenPath = rel
				b.Model.GoldenSHA256 = hash
			}
		}
		// Synthetic mode remains explicit, but the native test calls the adapter directly.
		write("testdata/native-model.json", b.Model)
		b.Model = baseModel
	}
	_ = os.MkdirAll(filepath.Dir(*out), 0755)
	write(*out, b)
	fmt.Println(*out)
}
func varint(v uint64) []byte {
	var b [10]byte
	n := binary.PutUvarint(b[:], v)
	return append([]byte(nil), b[:n]...)
}
func vint(field int, v uint64) []byte { return append(varint(uint64(field<<3)), varint(v)...) }
func message(field int, data []byte) []byte {
	b := varint(uint64(field<<3 | 2))
	b = append(b, varint(uint64(len(data)))...)
	return append(b, data...)
}
func str(field int, s string) []byte { return message(field, []byte(s)) }
func join(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}
func valueInfo(name string, width uint64) []byte {
	shape := join(message(1, str(2, "N")), message(1, vint(1, width)))
	tensor := join(vint(1, 1), message(2, shape))
	return join(str(1, name), message(2, message(1, tensor)))
}
func tinyModel() []byte {
	weights := make([]byte, 8)
	binary.LittleEndian.PutUint32(weights, math.Float32bits(1))
	binary.LittleEndian.PutUint32(weights[4:], math.Float32bits(1))
	tensor := join(vint(1, 2), vint(1, 1), vint(2, 1), message(4, weights), str(8, "weights"))
	node := join(str(1, "features"), str(1, "weights"), str(2, "boardings"), str(4, "MatMul"))
	graph := join(message(1, node), str(2, "synthetic-matmul"), message(5, tensor), message(11, valueInfo("features", 2)), message(12, valueInfo("boardings", 1)))
	return join(vint(1, 8), str(2, "tramflow-synthetic"), message(7, graph), message(8, vint(2, 13)))
}
