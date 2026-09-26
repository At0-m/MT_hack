package engine

import (
	"time"
	d "tramflow/internal/domain"
)

// Pick a complete published day; never silently shorten the default window.
func DefaultSelection(snapshot d.Snapshot) (d.Selection, error) {
	for _, coverage := range snapshot.Coverage {
		from := coverage.Window.From.In(Moscow)
		if from.Hour() != 0 {
			from = time.Date(from.Year(), from.Month(), from.Day()+1, 0, 0, 0, 0, Moscow)
		}
		selection := d.Selection{
			SnapshotID:    snapshot.Provenance.SnapshotID,
			RouteIDs:      []string{coverage.RouteID},
			View:          "day",
			Resolution:    "hour",
			SpatialDetail: "route",
			Window:        d.Window{From: from.UTC(), To: from.AddDate(0, 0, 1).UTC()},
		}
		if Validate(d.Descriptor{Kind: "forecast", Selection: selection}, snapshot) == nil {
			return selection, nil
		}
	}
	return d.Selection{}, d.Fail(503, "NO_DEFAULT_SELECTION", "Нет полностью опубликованного календарного дня.")
}
