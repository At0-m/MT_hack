package engine

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	d "tramflow/internal/domain"
)

// Canonical lines are versioned protocol data, not JSON serialization.
// Keep the final LF and fixed field order to match frontend/export replay.
func CalculationID(desc d.Descriptor) string {
	desc = Normalize(desc)
	selection := desc.Selection
	lines := []string{
		"tramflow-calculation-v2",
		"snapshot=" + selection.SnapshotID,
		"routes=" + strings.Join(selection.RouteIDs, ","),
		"view=" + selection.View,
		"from=" + selection.Window.From.Format(time.RFC3339),
		"to=" + selection.Window.To.Format(time.RFC3339),
		"resolution=" + selection.Resolution,
		"spatial_detail=" + selection.SpatialDetail,
		"kind=" + desc.Kind,
	}
	if overrides := desc.Overrides; overrides != nil {
		factors := FactorMillis(overrides)
		lines = append(lines,
			"effective_from="+overrides.Window.From.Format(time.RFC3339),
			"effective_to="+overrides.Window.To.Format(time.RFC3339),
			"fleet="+canonicalFleet(overrides.Fleet),
			fmt.Sprintf("factor_weather_milli=%d", factors[0]),
			fmt.Sprintf("factor_event_milli=%d", factors[1]),
			fmt.Sprintf("factor_season_milli=%d", factors[2]),
		)
	}
	canonical := strings.Join(lines, "\n") + "\n"
	return fmt.Sprintf("calc_%x", sha256.Sum256([]byte(canonical)))
}

func canonicalFleet(fleet *d.Fleet) string {
	if fleet == nil {
		return "unchanged"
	}
	if fleet.Kind == "absolute" {
		return fmt.Sprintf("absolute:%d", *fleet.Absolute)
	}
	return fmt.Sprintf("delta:%d", *fleet.Delta)
}
