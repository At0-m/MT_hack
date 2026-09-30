package engine

import (
	"math"
	"testing"
	"time"

	d "tramflow/internal/domain"
)

func TestCriterionFactorsAppliedBeforeHourlyAggregation(t *testing.T) {
	start := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	baseline := []d.Hour{
		{Time: start, Boardings: 100, Fleet: p(2), Reference: p(100)},
		{Time: start.Add(time.Hour), Boardings: 200, Fleet: p(2), Reference: p(100)},
	}
	overrides := &d.Overrides{
		Window:  d.Window{From: start, To: start.Add(time.Hour)},
		Factors: &d.Factors{Weather: p(1.2), Event: p(1.1), Season: p(0.9)},
	}
	evaluated := make([]d.Hour, len(baseline))
	for i, hour := range baseline {
		value, err := Scenario(hour, overrides)
		if err != nil {
			t.Fatal(err)
		}
		evaluated[i] = value
	}
	metrics := Aggregate(evaluated, 0.75)
	if math.Abs(metrics.Boardings-318.8) > 1e-9 {
		t.Fatalf("correction must precede sum: %v", metrics.Boardings)
	}
	if baseline[0].Boardings != 100 || evaluated[1].Boardings != 200 {
		t.Fatal("baseline or outside-window hour changed")
	}
	if metrics.PerVehicle.Value == nil || math.Abs(*metrics.PerVehicle.Value-79.7) > 1e-9 {
		t.Fatal("ratio must use aggregate boardings and vehicle hours")
	}
}
