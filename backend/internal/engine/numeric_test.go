package engine

import (
	"math"
	"testing"

	d "tramflow/internal/domain"
)

func TestNumericAdmissionAndDerivedGuard(t *testing.T) {
	for _, invalid := range []float64{math.SmallestNonzeroFloat64, 1e308, math.Inf(1), math.NaN()} {
		if err := ValidateHourNumbers(d.Hour{Reference: &invalid}); err == nil {
			t.Fatal("invalid denominator accepted", invalid)
		}
	}
	for _, valid := range []float64{0, MinPositiveProfile, MaxProfile} {
		if err := ValidateHourNumbers(d.Hour{Reference: &valid}); err != nil {
			t.Fatal(err)
		}
	}
	response := d.Response{Totals: []d.RouteReading{{Evaluated: d.Metrics{Index: d.Available(math.Inf(1))}}}}
	if ValidateResponseNumbers(response) == nil {
		t.Fatal("derived infinity accepted")
	}
}
