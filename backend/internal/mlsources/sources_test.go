package mlsources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOccupancyCalendarMappingAndNewYearLags(t *testing.T) {
	var csv strings.Builder
	csv.WriteString("time,avg_occupancy_rate,source\n")
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, Moscow)
	for i := 0; i < 8760; i++ {
		timestamp := start.Add(time.Duration(i) * time.Hour)
		fmt.Fprintf(&csv, "%s,%d,2025_synthetic\n", timestamp.Format("2006-01-02 15:04:05"), i)
	}
	profile, err := LoadOccupancy(strings.NewReader(csv.String()), 2025)
	if err != nil {
		t.Fatal(err)
	}
	features, err := profile.Features(time.Date(2026, 1, 1, 0, 0, 0, 0, Moscow))
	if err != nil {
		t.Fatal(err)
	}
	if features["occupancy_rate"] != 0 || features["occupancy_lag_1"] != 8759 || features["occupancy_lag_24"] != 8736 || features["occupancy_lag_168"] != 8592 {
		t.Fatalf("incorrect year boundary mapping: %v", features)
	}
	if features["occupancy_roll_24"] != 8747.5 || features["occupancy_roll_168"] != 8675.5 {
		t.Fatalf("incorrect rolling: %v", features)
	}
	leap, err := profile.Value(time.Date(2028, 2, 29, 8, 0, 0, 0, Moscow))
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := profile.Value(time.Date(2025, 2, 28, 8, 0, 0, 0, Moscow))
	if leap != reference {
		t.Fatal("February 29 fallback differs from February 28")
	}
	utc := time.Date(2026, 1, 1, 21, 0, 0, 0, time.UTC)
	value, err := profile.Value(utc)
	if err != nil || value != 24 {
		t.Fatalf("UTC hour not converted to Moscow: %v %v", value, err)
	}
	if !profile.Sources["2025_synthetic"] {
		t.Fatal("source provenance lost")
	}
}

func TestOccupancyRejectsDuplicatesAndNonfiniteValues(t *testing.T) {
	for _, body := range []string{
		"2025-01-01 00:00:00,NaN\n",
		"2025-01-01 00:00:00,1\n2025-01-01 00:00:00,2\n",
		"2026-01-01 00:00:00,1\n",
	} {
		if _, err := LoadOccupancy(strings.NewReader("time,avg_occupancy_rate\n"+body), 2025); err == nil {
			t.Fatalf("accepted invalid profile %q", body)
		}
	}
}

func TestMeteoHourlyForecastAndValidation(t *testing.T) {
	for _, invalid := range []string{"", "null", "unit", "gap", "http"} {
		t.Run("case_"+invalid, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("timezone") != "Europe/Moscow" || r.URL.Query().Get("wind_speed_unit") != "kmh" || !strings.Contains(r.URL.Query().Get("hourly"), "wind_speed_100m") {
					t.Error("incorrect API query")
				}
				if invalid == "http" {
					w.WriteHeader(429)
					return
				}
				units := map[string]string{"temperature_2m": "°C", "rain": "mm", "snowfall": "cm", "snow_depth": "m", "wind_speed_10m": "km/h", "wind_speed_100m": "km/h"}
				hourly := map[string]any{"time": []string{"2026-09-27T00:00", "2026-09-27T01:00"}}
				for _, column := range WeatherColumns {
					hourly[column] = []float64{0, 3}
				}
				hourly["temperature_2m"] = []float64{-5, 12}
				hourly["wind_speed_100m"] = []float64{35, 40}
				if invalid == "null" {
					hourly["rain"] = []any{nil, 3}
				}
				if invalid == "unit" {
					units["wind_speed_100m"] = "m/s"
				}
				if invalid == "gap" {
					hourly["time"] = []string{"2026-09-27T00:00", "2026-09-27T02:00"}
				}
				json.NewEncoder(w).Encode(map[string]any{"timezone": "Europe/Moscow", "utc_offset_seconds": 10800, "hourly_units": units, "hourly": hourly})
			}))
			defer server.Close()
			result, err := (MeteoClient{Endpoint: server.URL}).Fetch(context.Background(), 55.78, 37.58, 2)
			if invalid != "" {
				if err == nil {
					t.Fatalf("accepted invalid weather %s", invalid)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Hours) != 2 || result.Hours[0].Features["temperature_2m"] != -5 || result.Hours[1].Features["temperature_2m"] != 12 {
				t.Fatal("hourly weather averaged")
			}
			if result.Hours[0].Features["wind_speed_100m"] != 35 || result.Hours[0].Features["temp_below_zero"] != 1 || result.Hours[0].Features["weather_is_climatology"] != 0 {
				t.Fatal("weather features mismatch")
			}
			if result.Hours[0].Time.UTC().Hour() != 21 || result.RetrievedAt.IsZero() {
				t.Fatal("weather provenance/timezone mismatch")
			}
		})
	}
}
