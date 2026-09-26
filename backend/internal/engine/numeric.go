package engine

import (
	"fmt"
	"math"

	d "tramflow/internal/domain"
)

// Admission bounds, not estimates of transport capacity. Zero remains meaningful;
// positive denominators below this floor are rejected rather than rounded.
const (
	MaxHourlyBoardings = 1e9
	MinPositiveProfile = 1e-6
	MaxProfile         = 1e9
)

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func bounded(p *float64, minimum, maximum float64) bool {
	return p == nil || (finite(*p) && *p >= minimum && *p <= maximum)
}

func profile(p *float64, maximum float64) bool {
	return bounded(p, 0, maximum) && (p == nil || *p == 0 || *p >= MinPositiveProfile)
}

func ValidateHourNumbers(h d.Hour) error {
	if !profile(h.Fleet, 200) || !profile(h.Reference, MaxProfile) || !profile(h.Typical, MaxProfile) {
		return fmt.Errorf("profile outside numeric admission bounds")
	}
	if !bounded(h.Weather.Temperature, -100, 100) || !bounded(h.Weather.Precipitation, 0, 1000) {
		return fmt.Errorf("weather outside numeric admission bounds")
	}
	return nil
}

func finiteMetrics(m d.Metrics) bool {
	if !finite(m.Boardings) {
		return false
	}
	for _, reading := range []d.Reading{m.VehicleHours, m.MeanFleet, m.PerVehicle, m.Reference, m.Index, m.Required} {
		if reading.Value != nil && !finite(*reading.Value) {
			return false
		}
	}
	return true
}

func finiteReading(r d.RouteReading) bool {
	if !finiteMetrics(r.Baseline) || !finiteMetrics(r.Evaluated) || !finite(r.Delta.Boardings) || !finite(r.FleetAssessment.MaxDeficit) {
		return false
	}
	for _, value := range []*float64{r.Typical.Value, r.Relative, r.Delta.Index, r.Delta.VehicleHours, r.Weather.Temperature, r.Weather.Precipitation} {
		if value != nil && !finite(*value) {
			return false
		}
	}
	return true
}

// Check arithmetic results explicitly before JSON serialization or publication.
func ValidateResponseNumbers(response d.Response) error {
	for _, reading := range response.Totals {
		if !finiteReading(reading) {
			return fmt.Errorf("nonfinite total for route %s", reading.RouteID)
		}
	}
	for _, frame := range response.Frames {
		for _, reading := range frame.Routes {
			if !finiteReading(reading) {
				return fmt.Errorf("nonfinite frame for route %s", reading.RouteID)
			}
		}
	}
	return nil
}
