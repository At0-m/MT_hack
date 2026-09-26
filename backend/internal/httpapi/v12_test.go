package httpapi

import (
	"encoding/json"
	"testing"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
)

func TestUnsupportedSelectionsReturn422(t *testing.T) {
	handler, spec := handler(t)
	for _, change := range []string{"stop", "custom_hour", "short_day"} {
		payload, err := requestExample(spec, "ForecastRequest")
		if err != nil {
			t.Fatal(err)
		}
		var query d.Query
		if err := json.Unmarshal(payload, &query); err != nil {
			t.Fatal(err)
		}
		switch change {
		case "stop":
			query.Selection.SpatialDetail = "route_stop"
		case "custom_hour":
			query.Selection.View = "custom"
		case "short_day":
			query.Selection.Window.To = query.Selection.Window.From.Add(3600000000000)
		}
		payload, err = json.Marshal(query)
		if err != nil {
			t.Fatal(err)
		}
		response := req(handler, "POST", "/api/v1/forecasts/query", string(payload), true)
		if response.Code != 422 {
			t.Fatal(change, response.Code, response.Body.String())
		}
		if err := spec.Validate("Problem", response.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceScenarioExportGolden(t *testing.T) {
	handler, spec := handler(t)
	payload, err := json.Marshal(spec.Spec.Components.Examples["ExportRequest"].Value.Value)
	if err != nil {
		t.Fatal(err)
	}
	var query d.ExportRequest
	if err := json.Unmarshal(payload, &query); err != nil {
		t.Fatal(err)
	}
	if got := engine.CalculationID(query.Calculation); got != query.Expected {
		t.Fatal(got)
	}
	response := req(handler, "POST", "/api/v1/exports", string(payload), true)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
}

func TestIndicatorVariantsMatchKeys(t *testing.T) {
	allowed := map[string]map[string]bool{
		"weather":  {"none": true, "heat": true, "cold": true, "clear": true, "cloudy": true, "other": true, "rain": true, "snow": true, "unknown": true},
		"fleet":    {"none": true, "deficit": true, "balanced": true, "surplus": true, "unknown": true},
		"peak":     {"none": true, "peak": true, "off_peak": true, "unknown": true},
		"trend":    {"none": true, "up": true, "down": true, "flat": true},
		"event":    {"active": true, "none": true, "unknown": true},
		"calendar": {"holiday": true, "weekday": true, "weekend": true, "none": true, "unknown": true},
	}
	handler, spec := handler(t)
	for _, example := range []string{"ForecastRequest", "ScenarioRequest"} {
		payload, err := requestExample(spec, example)
		if err != nil {
			t.Fatal(err)
		}
		path := "/api/v1/forecasts/query"
		if example == "ScenarioRequest" {
			path = "/api/v1/scenarios/evaluate"
		}
		response := req(handler, "POST", path, string(payload), true)
		var result d.Response
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		for _, frame := range result.Frames {
			for _, route := range frame.Routes {
				for _, indicator := range route.Indicators {
					if !allowed[indicator.Key][indicator.Variant] {
						t.Fatal(indicator)
					}
					if indicator.Variant == "none" && indicator.Icon != "none" {
						t.Fatal(indicator)
					}
				}
			}
		}
	}
}
