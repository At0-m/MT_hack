package httpapi

import (
	"encoding/json"
	"testing"
	"time"
	"tramflow/internal/auth"
	"tramflow/internal/inference"
)

func TestWeatherMetadataStableAcrossForecastSnapshots(t *testing.T) {
	f, c := fixture(t)
	m := inference.New("../../artifacts")
	defer m.Close()
	var first map[string]any
	for i := 0; i < 2; i++ {
		f.b.Snapshot.Created = f.b.Snapshot.Created.Add(time.Hour)
		f.b.Snapshot.Provenance.SnapshotID = "weather-reuse-" + string(rune('a'+i))
		s := &Server{Store: f, Models: m, Contract: c, Auth: auth.New(newTestAuth())}
		for _, route := range []string{"demo-01", "demo-02"} {
			path := "/api/v1/weather?forecast_snapshot_id=" + f.b.Snapshot.Provenance.SnapshotID + "&route_id=" + route + "&from=2026-09-01T15:00:00Z&to=2026-09-01T16:00:00Z"
			response := req(s.Handler(), "GET", path, "", true)
			if response.Code != 200 {
				t.Fatal(response.Code, response.Body.String())
			}
			if err := c.Validate("WeatherResponse", response.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result["location_id"] != "synthetic-moscow" || result["source_id"] != "synthetic-weather-source" {
				t.Fatal(result)
			}
			if first == nil {
				first = result
				continue
			}
			for _, key := range []string{"weather_snapshot_id", "location_id", "provider", "retrieved_at", "source_id"} {
				if result[key] != first[key] {
					t.Fatal("metadata changed", key)
				}
			}
		}
	}
}

func TestReadinessRejectsNoBootstrapDay(t *testing.T) {
	f, c := fixture(t)
	for i := range f.b.Snapshot.Coverage {
		coverage := &f.b.Snapshot.Coverage[i]
		coverage.Window.From = coverage.Window.From.Add(12 * time.Hour)
		coverage.Window.To = coverage.Window.From.Add(24 * time.Hour)
	}
	m := inference.New("../../artifacts")
	defer m.Close()
	s := &Server{Store: f, Models: m, Contract: c, Auth: auth.New(newTestAuth())}
	h := s.Handler()
	for _, path := range []string{"/health/ready", "/api/v1/bootstrap"} {
		response := req(h, "GET", path, "", true)
		if response.Code != 503 {
			t.Fatal(path, response.Code, response.Body.String())
		}
	}
}
