package engine

import (
	"testing"
	"time"
	d "tramflow/internal/domain"
)

func TestCustomAcrossCalendarBoundaries(t *testing.T) {
	for _, start := range []string{"2025-12-31T00:00:00+03:00", "2024-02-28T00:00:00+03:00"} {
		from, _ := time.Parse(time.RFC3339, start)
		to := from.Add(72 * time.Hour)
		s := testSnapshot()
		s.Provenance.Origin = from
		s.Coverage[0].Window = d.Window{From: from, To: from.Add(744 * time.Hour)}
		s.Coverage[0].MaxLead = 744
		desc := testDesc()
		desc.Selection.View = "custom"
		desc.Selection.Resolution = "day"
		desc.Selection.Window = d.Window{From: from, To: to}
		if err := Validate(desc, s); err != nil {
			t.Fatal(start, err)
		}
	}
}
func TestCustomHourlyTooWideIsNotClipped(t *testing.T) {
	desc := testDesc()
	desc.Selection.View = "custom"
	desc.Selection.Window.To = desc.Selection.Window.From.Add(169 * time.Hour)
	if Validate(desc, testSnapshot()) == nil {
		t.Fatal("must reject width")
	}
}
func TestZeroReferenceHasExplicitReason(t *testing.T) {
	m := Aggregate([]d.Hour{{Boardings: 100, Fleet: p(1), Reference: p(0)}}, .75)
	if m.Index.Reason != "REFERENCE_ZERO" || m.Required.Reason != "REFERENCE_ZERO" {
		t.Fatal(m)
	}
}
