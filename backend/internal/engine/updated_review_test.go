package engine

import (
	"context"
	"strings"
	"testing"
	"time"
	d "tramflow/internal/domain"
)

func TestSummaryUsesFrameMaximum(t *testing.T) {
	response := d.Response{
		Totals: []d.RouteReading{{RouteID: "a", Evaluated: d.Metrics{Index: d.Available(.5), Boardings: 100}}, {RouteID: "b", Evaluated: d.Metrics{Index: d.Available(.6)}}},
		Frames: []d.Frame{{Routes: []d.RouteReading{{RouteID: "a", Evaluated: d.Metrics{Index: d.Available(1.5)}}, {RouteID: "b", Evaluated: d.Metrics{Index: d.Available(1.8)}}}}, {Routes: []d.RouteReading{{RouteID: "a", Evaluated: d.Metrics{Index: d.Available(.3)}}}}},
	}
	for _, test := range []struct {
		focus d.Focus
		want  string
	}{{d.Focus{Kind: "route", Route: "a"}, "1.50"}, {d.Focus{Kind: "overall"}, "1.80"}} {
		if got := Summarize(response, test.focus).Text; !strings.Contains(got, test.want) {
			t.Fatal(got)
		}
	}
}

func TestFleetScenarioProvenance(t *testing.T) {
	start := time.Now()
	base := d.Hour{Time: start, Fleet: p(10), Proxy: true, Source: "observed_vehicle_profile"}
	n, delta := 5, 1
	for _, test := range []struct {
		fleet  *d.Fleet
		proxy  bool
		source string
	}{{&d.Fleet{Kind: "absolute", Absolute: &n}, false, "scenario_absolute"}, {&d.Fleet{Kind: "delta", Delta: &delta}, true, "scenario_delta"}} {
		changed, err := Scenario(base, &d.Overrides{Window: d.Window{From: start, To: start.Add(time.Hour)}, Fleet: test.fleet})
		if err != nil || changed.Proxy != test.proxy || changed.Source != test.source {
			t.Fatal(changed, err)
		}
	}
}

func TestWeightedBudgetAtMaximum(t *testing.T) {
	q := testDesc().Selection
	q.RouteIDs = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	q.Window.To = q.Window.From.Add(744 * time.Hour)
	b := requestBudget{}
	release, err := b.acquire(d.WithUser(context.Background(), "one"), q)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.acquire(d.WithUser(context.Background(), "two"), q); err == nil {
		t.Fatal("two maximum requests admitted")
	}
	release()
	release, err = b.acquire(d.WithUser(context.Background(), "two"), q)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if len(b.users) != 0 || b.used != 0 {
		t.Fatal("budget leaked")
	}
}
