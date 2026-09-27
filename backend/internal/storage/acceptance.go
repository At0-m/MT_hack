package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
)

func validateWeatherMode(point d.WeatherPoint) error {
	switch point.Mode {
	case "missing":
		if point.Temperature != nil || point.Precipitation != nil || point.Code != nil {
			return fmt.Errorf("missing weather must have null values")
		}
	case "climatology":
		if point.Code != nil || point.RunAt != nil {
			return fmt.Errorf("climatology cannot have a provider code/run")
		}
	case "forecast":
		if point.RunAt == nil || point.RunAt.IsZero() {
			return fmt.Errorf("forecast requires provider run time")
		}
	default:
		return fmt.Errorf("unsupported weather mode")
	}
	return nil
}

// Every advertised view must have a working calendar window for every route.
// We preserve the existing contract: event occupies peak's slot when active.
func validateCapabilities(snapshot d.Snapshot, hours []d.Hour, contract *contract.Contract) error {
	byRoute := make(map[string][]d.Hour)
	for _, hour := range hours {
		byRoute[hour.RouteID] = append(byRoute[hour.RouteID], hour)
	}
	for _, coverage := range snapshot.Coverage {
		views := append(append([]string(nil), snapshot.Views...), "custom")
		for _, view := range views {
			from := coverage.Window.From.In(engine.Moscow)
			if from.Hour() != 0 {
				from = time.Date(from.Year(), from.Month(), from.Day()+1, 0, 0, 0, 0, engine.Moscow)
			}
			resolution, days := "day", 1
			var to time.Time
			switch view {
			case "day":
				resolution = "hour"
			case "week":
				days = 7
			case "month":
				if from.Day() != 1 {
					from = time.Date(from.Year(), from.Month()+1, 1, 0, 0, 0, 0, engine.Moscow)
				}
				to = from.AddDate(0, 1, 0)
			case "custom":
				days = min(engine.MaxCustomDays, int(coverage.Window.To.Sub(from)/(24*time.Hour)))
			default:
				return fmt.Errorf("unsupported advertised view %s", view)
			}
			if to.IsZero() {
				to = from.AddDate(0, 0, days)
			}
			selection := d.Selection{
				SnapshotID: snapshot.Provenance.SnapshotID,
				RouteIDs:   []string{coverage.RouteID}, View: view,
				Resolution: resolution, SpatialDetail: "route",
				Window: d.Window{From: from.UTC(), To: to.UTC()},
			}
			selected := make([]d.Hour, 0)
			for _, hour := range byRoute[coverage.RouteID] {
				if !hour.Time.Before(from) && hour.Time.Before(to) {
					selected = append(selected, hour)
				}
			}
			response, err := engine.Calculate(d.Descriptor{Kind: "forecast", Selection: selection}, snapshot, selected)
			if err != nil {
				return fmt.Errorf("capability %s/%s: %w", coverage.RouteID, view, err)
			}
			engine.Enrich(&response)
			raw, err := json.Marshal(response)
			if err != nil {
				return err
			}
			if err := contract.Validate("CalculationResponse", raw); err != nil {
				return fmt.Errorf("capability response: %w", err)
			}
		}
	}
	return nil
}
