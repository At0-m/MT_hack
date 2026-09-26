//go:build onnx

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/inference"
)

type countedFeatures struct {
	snapshot d.Snapshot
	model    d.Model
	batches  []int
	forbid   bool
}

func (r *countedFeatures) Snapshot(context.Context, string) (d.Snapshot, d.Model, error) {
	return r.snapshot, r.model, nil
}
func (r *countedFeatures) Hours(_ context.Context, _ d.Snapshot, q SelectionInput) ([]d.Hour, error) {
	out := []d.Hour{}
	for _, route := range q.Routes {
		for hour := q.Window.From; hour.Before(q.Window.To); hour = hour.Add(time.Hour) {
			out = append(out, d.Hour{RouteID: route, Time: hour, Features: map[string]float64{}, Fleet: p(10), Reference: p(65), Typical: p(549), Source: "manual_plan", FeaturesAvailableAt: r.snapshot.Provenance.CompleteThrough, Weather: d.WeatherPoint{Mode: "missing"}})
		}
	}
	return out, nil
}
func (r *countedFeatures) Features(_ context.Context, _ d.Snapshot, hours []d.Hour) ([]d.Hour, error) {
	if r.forbid {
		return nil, fmt.Errorf("hot cache fetched features")
	}
	r.batches = append(r.batches, len(hours))
	out := append([]d.Hour(nil), hours...)
	for i := range out {
		out[i].Features = map[string]float64{"demand_base": 600, "boost": 10}
	}
	return out, nil
}

func TestColdFeaturesAreChunkedAndHotCacheSkipsThem(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/native-model.json")
	if err != nil {
		t.Fatal(err)
	}
	var model d.Model
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	repo := &countedFeatures{snapshot: testSnapshot(), model: model}
	repo.snapshot.Provenance.Mode = "synthetic_mock"
	repo.snapshot.Provenance.CompleteThrough = repo.snapshot.Provenance.Origin.Add(-time.Hour)
	m := inference.New("../../artifacts")
	defer m.Close()
	service := &Service{Repo: repo, Models: m, Cache: NewCache(1 << 20)}
	query := testDesc()
	query.Selection.View = "week"
	query.Selection.Resolution = "day"
	query.Selection.Window.To = query.Selection.Window.From.Add(168 * time.Hour)
	cold, err := service.Calculate(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.batches) != 2 || repo.batches[0] != 128 || repo.batches[1] != 40 {
		t.Fatal(repo.batches)
	}
	repo.forbid = true
	hot, err := service.Calculate(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	coldJSON, _ := json.Marshal(cold)
	hotJSON, _ := json.Marshal(hot)
	if string(coldJSON) != string(hotJSON) {
		t.Fatal("cache changed calculation")
	}
}
