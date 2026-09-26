package storage

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/inference"
)

func TestCandidateGeographyAndWeatherIntegrity(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/demo-bundle.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := contract.Load("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m := inference.New("../../artifacts")
	defer m.Close()
	for _, kind := range []string{"position", "physical conflict", "occurrence", "pattern", "weather source", "weather retrieved", "weather location", "stop schema pair"} {
		t.Run(kind, func(t *testing.T) {
			var b d.Bundle
			if err := json.Unmarshal(raw, &b); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "position", "physical conflict", "occurrence", "pattern":
				index := 0
				if kind == "physical conflict" || kind == "occurrence" || kind == "pattern" {
					index = 1
				}
				var detail map[string]any
				_ = json.Unmarshal(b.Routes[index].Detail, &detail)
				pattern := detail["patterns"].([]any)[0].(map[string]any)
				stop := pattern["stops"].([]any)[0].(map[string]any)
				switch kind {
				case "position":
					stop["position"].(map[string]any)["longitude"] = 37.0
				case "physical conflict":
					stop["stop_id"] = "demo-01-stop-1"
				case "occurrence":
					stop["route_stop_id"] = "demo-01-out-1"
				case "pattern":
					pattern["route_pattern_id"] = "demo-01-out"
				}
				b.Routes[index].Detail = encode(detail)
			case "weather source":
				b.WeatherMetadata.SourceID = "does-not-exist"
			case "weather retrieved":
				b.WeatherMetadata.RetrievedAt = b.Snapshot.Created.AddDate(0, 0, 1)
			case "weather location":
				delete(b.WeatherMetadata.Locations, "demo-01")
			case "stop schema pair":
				b.Snapshot.Provenance.StopModel = "stop-v1"
			}
			if err := ValidateBundle(context.Background(), &b, c, m, true); err == nil {
				t.Fatal("invalid candidate accepted")
			}
		})
	}
}
