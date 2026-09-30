package engine

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
	d "tramflow/internal/domain"
)

func p(v float64) *float64 { return &v }
func testSnapshot() d.Snapshot {
	o := time.Date(2026, 8, 31, 21, 0, 0, 0, time.UTC)
	return d.Snapshot{
		Provenance: d.Provenance{SnapshotID: "demo-september-2026-v1", Origin: o},
		Target:     0.75,
		Views:      []string{"day", "week", "month"},
		Coverage: []d.Coverage{
			{
				RouteID:     "demo-01",
				Window:      d.Window{From: o, To: o.Add(720 * time.Hour)},
				Resolutions: []string{"hour", "day"},
				MaxLead:     720,
				Scopes:      []string{"route"},
				StopStatus:  "unavailable",
			},
		},
	}
}
func testDesc() d.Descriptor {
	s := testSnapshot()
	return d.Descriptor{
		Kind: "forecast",
		Selection: d.Selection{
			SnapshotID:    s.Provenance.SnapshotID,
			RouteIDs:      []string{"demo-01"},
			View:          "day",
			Window:        d.Window{From: s.Provenance.Origin, To: s.Provenance.Origin.Add(24 * time.Hour)},
			Resolution:    "hour",
			SpatialDetail: "route",
		},
	}
}
func TestCanonicalGolden(t *testing.T) {
	desc := testDesc()
	if got := CalculationID(desc); got != "calc_bfdb42badb4e5a48ad6ee91a595bf1ce84b0009f20b5cc3c342c6250413401ef" {
		t.Fatal(got)
	}
	desc.Selection.Window.From = desc.Selection.Window.From.In(time.FixedZone("Moscow", 10800))
	if CalculationID(desc) != "calc_bfdb42badb4e5a48ad6ee91a595bf1ce84b0009f20b5cc3c342c6250413401ef" {
		t.Fatal("offset changed identity")
	}
}
func TestRatiosUseSummedDenominators(t *testing.T) {
	hs := []d.Hour{
		{Boardings: 100, Fleet: p(1), Reference: p(100), Source: "manual_plan"},
		{Boardings: 100, Fleet: p(3), Reference: p(100), Source: "manual_plan"},
	}
	m := Aggregate(hs, 0.75)
	if *m.Index.Value != 0.5 || *m.MeanFleet.Value != 2 || *m.Required.Value != 2 {
		t.Fatalf("%+v", m)
	}
}
func TestUnavailableMetrics(t *testing.T) {
	tests := []struct {
		name   string
		hours  []d.Hour
		reason string
	}{
		{"unknown supply", []d.Hour{{Boardings: 1, Reference: p(1)}}, "SUPPLY_UNKNOWN"},
		{"zero supply", []d.Hour{{Fleet: p(0), Reference: p(1)}}, "NO_SUPPLY"},
		{"missing reference", []d.Hour{{Boardings: 1, Fleet: p(1)}}, "REFERENCE_MISSING"},
		{"zero reference", []d.Hour{{Fleet: p(1), Reference: p(0)}}, "REFERENCE_ZERO"},
		{"partial supply", []d.Hour{{Fleet: p(1), Reference: p(1)}, {Reference: p(1)}}, "SUPPLY_UNKNOWN"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Aggregate(tc.hours, 0.75)
			if m.Index.Value != nil || m.Index.Reason != tc.reason {
				t.Fatalf("%+v", m.Index)
			}
		})
	}
}
func TestScenarioHourlyWindow(t *testing.T) {
	desc := testDesc()
	i := 12
	desc.Kind = "scenario"
	desc.Overrides = &d.Overrides{
		Window:  d.Window{From: desc.Selection.Window.From, To: desc.Selection.Window.From.Add(time.Hour)},
		Fleet:   &d.Fleet{Kind: "absolute", Absolute: &i},
		Factors: &d.Factors{Event: p(1.1)},
	}
	hs := []d.Hour{}
	for j := 0; j < 24; j++ {
		hs = append(hs, d.Hour{
			RouteID:   "demo-01",
			Time:      desc.Selection.Window.From.Add(time.Duration(j) * time.Hour),
			Boardings: 610,
			Fleet:     p(10),
			Reference: p(65),
			Typical:   p(549),
			Source:    "observed_vehicle_profile",
			Proxy:     true,
			Weather:   d.WeatherPoint{Mode: "missing"},
		})
	}
	r, err := Calculate(desc, testSnapshot(), hs)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r.Totals[0].Evaluated.Boardings-14701) > 1e-8 || *r.Totals[0].Evaluated.VehicleHours.Value != 242 {
		t.Fatal(r.Totals)
	}
	if r.Frames[1].Routes[0].Evaluated.Boardings != 610 {
		t.Fatal("override escaped window")
	}
}
func TestDeltaCannotUseUnknownSupply(t *testing.T) {
	delta := 1
	_, err := Scenario(d.Hour{Time: testDesc().Selection.Window.From}, &d.Overrides{Window: testDesc().Selection.Window, Fleet: &d.Fleet{Kind: "delta", Delta: &delta}})
	if err == nil {
		t.Fatal("expected rejection")
	}
}
func TestAbsoluteCanRestoreUnknownSupply(t *testing.T) {
	n := 5
	h, err := Scenario(d.Hour{Time: testDesc().Selection.Window.From}, &d.Overrides{Window: testDesc().Selection.Window, Fleet: &d.Fleet{Kind: "absolute", Absolute: &n}})
	if err != nil || h.Fleet == nil || *h.Fleet != 5 {
		t.Fatal(h, err)
	}
}
func TestNegativeFleetRejected(t *testing.T) {
	delta := -2
	_, err := Scenario(d.Hour{Fleet: p(1), Time: testDesc().Selection.Window.From}, &d.Overrides{Window: testDesc().Selection.Window, Fleet: &d.Fleet{Kind: "delta", Delta: &delta}})
	if err == nil {
		t.Fatal("expected rejection")
	}
}
func TestFactorProductAndPrecision(t *testing.T) {
	for _, values := range [][3]float64{{2, 2, 2}, {.5, .5, .5}, {1.0001, 1, 1}} {
		desc := testDesc()
		desc.Kind = "scenario"
		desc.Overrides = &d.Overrides{
			Window:  desc.Selection.Window,
			Factors: &d.Factors{Weather: p(values[0]), Event: p(values[1]), Season: p(values[2])},
		}
		if Validate(desc, testSnapshot()) == nil {
			t.Fatal(values)
		}
	}
}
func TestSelectionLimits(t *testing.T) {
	for _, name := range []string{"unaligned", "duplicate", "outside", "too_wide", "day_alignment", "wrong_resolution"} {
		t.Run(name, func(t *testing.T) {
			desc := testDesc()
			switch name {
			case "unaligned":
				desc.Selection.Window.From = desc.Selection.Window.From.Add(time.Minute)
			case "duplicate":
				desc.Selection.RouteIDs = append(desc.Selection.RouteIDs, "demo-01")
			case "outside":
				desc.Selection.Window.From = testSnapshot().Provenance.Origin.Add(-time.Hour)
			case "too_wide":
				desc.Selection.Window.To = desc.Selection.Window.From.Add(25 * time.Hour)
			case "day_alignment":
				desc.Selection.View = "week"
				desc.Selection.Resolution = "day"
			case "wrong_resolution":
				desc.Selection.Resolution = "minute"
			}
			if Validate(desc, testSnapshot()) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
func TestMissingWeatherDoesNotReturnPartialSum(t *testing.T) {
	w := Weather([]d.Hour{
		{Weather: d.WeatherPoint{Mode: "forecast", Temperature: p(10), Precipitation: p(1)}},
		{Weather: d.WeatherPoint{Mode: "missing"}},
	})
	if w.Temperature != nil || w.Precipitation != nil || w.Missing != 1 {
		t.Fatal(w)
	}
}
func TestIncompleteGridRejected(t *testing.T) {
	_, err := Calculate(testDesc(), testSnapshot(), nil)
	if err == nil {
		t.Fatal("expected coverage error")
	}
}
func TestRequiredFleetUsesMaximum(t *testing.T) {
	m := Aggregate([]d.Hour{{Boardings: 750, Fleet: p(1), Reference: p(100)}, {Boardings: 75, Fleet: p(1), Reference: p(100)}}, .75)
	if *m.Required.Value != 10 {
		t.Fatal(m.Required)
	}
}
func TestCacheEviction(t *testing.T) {
	c := NewCache(1024)
	c.Set("a", 1)
	c.Set("b", 2)
	_, _ = c.Get("a")
	c.Set("c", 3)
	if _, ok := c.Get("b"); ok {
		t.Fatal("LRU was not evicted")
	}
}
func TestFixtureHasNoForecastQualityClaim(t *testing.T) {
	b, err := os.ReadFile("../../testdata/demo-bundle.json")
	if os.IsNotExist(err) {
		t.Skip("run cmd/fixtures")
	}
	if err != nil {
		t.Fatal(err)
	}
	var bundle d.Bundle
	if err = json.Unmarshal(b, &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Snapshot.Provenance.Mode != "synthetic_mock" {
		t.Fatal("fixture not marked synthetic")
	}
}
