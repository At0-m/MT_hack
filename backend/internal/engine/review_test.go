package engine

import (
	"strings"
	"testing"
	"time"
	d "tramflow/internal/domain"
)

func TestFleetHourlyAdequacy(t *testing.T) {
	for _, test := range []struct {
		name    string
		hours   []d.Hour
		want    string
		deficit float64
	}{
		{"false surplus", []d.Hour{{Boardings: 750, Fleet: p(1), Reference: p(100)}, {Boardings: 75, Fleet: p(25), Reference: p(100)}}, "deficit", 9},
		{"false deficit", []d.Hour{{Boardings: 750, Fleet: p(10), Reference: p(100)}, {Boardings: 75, Fleet: p(1), Reference: p(100)}}, "balanced", 0},
		{"unknown hour", []d.Hour{{Boardings: 750, Fleet: p(1), Reference: p(100)}, {Boardings: 75, Reference: p(100)}}, "unknown", 9},
		{"zero supply", []d.Hour{{Boardings: 75, Fleet: p(0), Reference: p(100)}}, "deficit", 1},
		{"zero demand", []d.Hour{{Fleet: p(0), Reference: p(0)}}, "balanced", 0},
		{"spare hour", []d.Hour{{Boardings: 75, Fleet: p(2), Reference: p(100)}}, "surplus", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := AssessFleet(test.hours, 0.75)
			if a.Variant != test.want || a.MaxDeficit != test.deficit {
				t.Fatalf("%+v", a)
			}
		})
	}
}

func TestPartialScenarioIndicators(t *testing.T) {
	desc := testDesc()
	desc.Kind = "scenario"
	w := desc.Selection.Window
	weatherFactor := 1.2
	vehicles := 0
	desc.Overrides = &d.Overrides{Window: d.Window{From: w.From, To: w.From.Add(time.Hour)}, Factors: &d.Factors{Weather: &weatherFactor}, Fleet: &d.Fleet{Kind: "absolute", Absolute: &vehicles}}
	code := 61
	hours := []d.Hour{}
	for i := 0; i < 24; i++ {
		hours = append(hours, d.Hour{RouteID: "demo-01", Time: w.From.Add(time.Duration(i) * time.Hour), Boardings: 75, Fleet: p(1), Reference: p(100), Typical: p(75), Weather: d.WeatherPoint{Mode: "forecast", Code: &code}})
	}
	r, err := Calculate(desc, testSnapshot(), hours)
	if err != nil {
		t.Fatal(err)
	}
	Enrich(&r)
	first, next := r.Frames[0].Routes[0], r.Frames[1].Routes[0]
	if first.Indicators[1].Variant != "deficit" || next.Indicators[1].Variant != "balanced" || r.Totals[0].Indicators[1].Variant != "deficit" {
		t.Fatal("fleet scenario window")
	}
	if first.Indicators[0].Variant != "rain" || next.Indicators[0].Variant != "rain" || strings.Contains(next.Indicators[0].Text, "коэффициент") {
		t.Fatal("weather category or window")
	}
	if !strings.Contains(r.Totals[0].Indicators[0].Text, "части периода") {
		t.Fatal("partial aggregate not explained")
	}
}

func TestTrendAndWeatherCategories(t *testing.T) {
	for _, test := range []struct {
		value float64
		want  string
	}{{0.1, "up"}, {-0.1, "down"}, {0.05, "flat"}, {0, "flat"}} {
		if got := trendIndicator(d.RouteReading{Relative: p(test.value)}).Variant; got != test.want {
			t.Fatal(got)
		}
	}
	for code, want := range map[int]string{0: "clear", 3: "cloudy", 61: "rain", 71: "snow", 95: "other", 100: "unknown"} {
		point := d.WeatherPoint{Mode: "forecast", Code: &code, Temperature: p(-5)}
		if got := weatherCategory(point); got != want {
			t.Fatal(code, got)
		}
	}
	rain, snow := 61, 71
	if got := Weather([]d.Hour{{Weather: d.WeatherPoint{Mode: "forecast", Code: &rain}}, {Weather: d.WeatherPoint{Mode: "forecast", Code: &snow}}}).Category; got != "unknown" {
		t.Fatal(got)
	}
	clear := 0
	for temp, want := range map[float64]string{30: "heat", -15: "cold"} {
		if got := weatherCategory(d.WeatherPoint{Mode: "forecast", Code: &clear, Temperature: p(temp)}); got != want {
			t.Fatal(temp, got)
		}
	}
	saturday := time.Date(2026, 9, 26, 0, 0, 0, 0, Moscow)
	if got := calendarCategory([]d.Hour{{Time: saturday}}); got != "weekend" {
		t.Fatal(got)
	}
	if got := calendarCategory([]d.Hour{{Time: saturday, Features: map[string]float64{"is_holiday": 1}}}); got != "holiday" {
		t.Fatal(got)
	}
	if got := calendarCategory([]d.Hour{{Time: saturday}, {Time: saturday.Add(48 * time.Hour)}}); got != "unknown" {
		t.Fatal(got)
	}
}

func TestDefaultSelectionRejectsUnusableSnapshot(t *testing.T) {
	for _, variant := range []string{"shifted", "no day", "no hour"} {
		s := testSnapshot()
		s.Coverage[0].Window.To = s.Coverage[0].Window.From.Add(24 * time.Hour)
		switch variant {
		case "shifted":
			s.Coverage[0].Window.From = s.Coverage[0].Window.From.Add(12 * time.Hour)
			s.Coverage[0].Window.To = s.Coverage[0].Window.To.Add(12 * time.Hour)
			s.Coverage[0].MaxLead = 36
		case "no day":
			s.Views = []string{"week"}
		case "no hour":
			s.Coverage[0].Resolutions = []string{"day"}
		}
		if _, err := DefaultSelection(s); err == nil {
			t.Fatal(variant)
		}
	}
}
