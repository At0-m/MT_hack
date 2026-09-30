package engine

import (
	"math"
	d "tramflow/internal/domain"
)

// Unknown hours take precedence: an incomplete period cannot certify adequacy.
// With complete data, any deficit wins; surplus means at least one spare hour.
func AssessFleet(hours []d.Hour, target float64) d.FleetAssessment {
	a := d.FleetAssessment{Variant: "balanced"}
	unknown, surplus := len(hours) == 0, false
	for _, h := range hours {
		if h.Fleet == nil || h.Reference == nil || (*h.Reference == 0 && h.Boardings > 0) {
			unknown = true
			continue
		}
		required := 0.0
		if *h.Reference > 0 {
			required = math.Ceil(h.Boardings / (target * *h.Reference))
		}
		gap := required - *h.Fleet
		if gap > 0 {
			a.DeficitHours++
			a.MaxDeficit = math.Max(a.MaxDeficit, gap)
		} else if gap < 0 {
			surplus = true
		}
	}
	switch {
	case unknown:
		a.Variant = "unknown"
	case a.DeficitHours > 0:
		a.Variant = "deficit"
	case surplus:
		a.Variant = "surplus"
	}
	return a
}
