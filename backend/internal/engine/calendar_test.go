package engine

import (
	"testing"
	"time"
	d "tramflow/internal/domain"
)

func TestCalendarMonthsAndWeek(t *testing.T) {
	for _, start := range []string{
		"2024-02-01T00:00:00+03:00",
		"2025-02-01T00:00:00+03:00",
		"2026-04-01T00:00:00+03:00",
		"2026-12-01T00:00:00+03:00",
	} {
		from, err := time.Parse(time.RFC3339, start)
		if err != nil {
			t.Fatal(err)
		}
		selection := d.Selection{
			View: "month", Resolution: "day",
			Window: d.Window{From: from, To: from.AddDate(0, 1, 0)},
		}
		if err := ValidateCalendar(selection); err != nil {
			t.Fatal(start, err)
		}
		selection.Window.To = selection.Window.To.Add(-24 * time.Hour)
		if ValidateCalendar(selection) == nil {
			t.Fatal("shortened month accepted", start)
		}
		selection.View = "week"
		selection.Window.To = from.AddDate(0, 0, 7)
		if err := ValidateCalendar(selection); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceCanonicalV2Golden(t *testing.T) {
	desc := testDesc()
	desc.Selection.View = "custom"
	desc.Selection.Resolution = "day"
	desc.Selection.SpatialDetail = "route_stop"
	want := "calc_1eb3d59b2415909fdc14783d3a2c2746798e7bb3961a4945bfd5983bcbe72745"
	if got := CalculationID(desc); got != want {
		t.Fatal(got)
	}
	desc.Selection.SpatialDetail = "route"
	if CalculationID(desc) == want {
		t.Fatal("spatial scope omitted from hash")
	}
}
