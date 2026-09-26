package engine

import (
	"math"
	"time"
	d "tramflow/internal/domain"
)

func Scenario(h d.Hour, o *d.Overrides) (d.Hour, error) {
	if o == nil || h.Time.Before(o.Window.From) || !h.Time.Before(o.Window.To) {
		return h, nil
	}
	m := FactorMillis(o)
	h.Boardings *= float64(m[0]*m[1]*m[2]) / 1e9
	if o.Fleet != nil {
		f := o.Fleet
		n := 0.0
		if f.Kind == "absolute" {
			n = float64(*f.Absolute)
			h.Source = "scenario_absolute"
			h.Proxy = false
		} else {
			if h.Fleet == nil {
				return h, d.Fail(422, "SUPPLY_UNKNOWN", "Delta требует известного базового выпуска.")
			}
			n = *h.Fleet + float64(*f.Delta)
			h.Source = "scenario_delta"
		}
		if n < 0 || n > 200 {
			return h, d.Fail(422, "FLEET_OUT_OF_RANGE", "Результат выпуска выходит за 0–200.")
		}
		h.Fleet = &n
	}
	return h, nil
}
func Aggregate(hours []d.Hour, target float64) d.Metrics {
	x := d.Metrics{Source: hours[0].Source, Level: "unavailable"}
	supplyOK, refOK, requiredOK := true, true, true
	vehicle, ref, required := 0.0, 0.0, 0.0
	for _, h := range hours {
		x.Boardings += h.Boardings
		x.Proxy = x.Proxy || h.Proxy
		if h.Source != x.Source {
			x.Source = "mixed"
		}
		if h.Fleet == nil {
			supplyOK = false
		} else {
			vehicle += *h.Fleet
		}
		if h.Reference == nil {
			refOK = false
			requiredOK = false
		} else {
			if h.Fleet != nil {
				ref += *h.Fleet * *h.Reference
			}
			if *h.Reference > 0 {
				required = math.Max(required, math.Ceil(h.Boardings/(target**h.Reference)))
			} else if h.Boardings > 0 {
				requiredOK = false
			}
		}
	}
	x.Required = d.Missing("REFERENCE_MISSING")
	if requiredOK {
		x.Required = d.Available(required)
	} else if refOK {
		x.Required = d.Missing("REFERENCE_ZERO")
	}
	x.VehicleHours = d.Missing("SUPPLY_UNKNOWN")
	x.MeanFleet = x.VehicleHours
	x.PerVehicle = x.VehicleHours
	x.Reference = x.VehicleHours
	x.Index = x.VehicleHours
	if !supplyOK {
		return x
	}
	x.VehicleHours = d.Available(vehicle)
	x.MeanFleet = d.Available(vehicle / float64(len(hours)))
	x.PerVehicle = d.Missing("NO_SUPPLY")
	x.Index = x.PerVehicle
	if vehicle > 0 {
		x.PerVehicle = d.Available(x.Boardings / vehicle)
	}
	x.Reference = d.Missing("REFERENCE_MISSING")
	if !refOK {
		if vehicle > 0 {
			x.Index = x.Reference
		}
		return x
	}
	x.Reference = d.Available(ref)
	if vehicle == 0 {
		return x
	}
	if ref == 0 {
		x.Index = d.Missing("REFERENCE_ZERO")
		return x
	}
	index := x.Boardings / ref
	x.Index = d.Available(index)
	x.Level = "normal"
	if index >= 1.25 {
		x.Level = "very_high"
	} else if index >= 1 {
		x.Level = "high"
	} else if index >= 0.75 {
		x.Level = "elevated"
	}
	return x
}
func Weather(hours []d.Hour) d.WeatherSummary {
	w := d.WeatherSummary{Category: weatherCategory(hours[0].Weather)}
	temp, precip := 0.0, 0.0
	tempOK, precipOK := true, true
	for _, h := range hours {
		p := h.Weather
		if weatherCategory(p) != w.Category {
			w.Category = "unknown"
		}
		switch p.Mode {
		case "forecast":
			w.Forecast++
		case "climatology":
			w.Climatology++
		default:
			w.Missing++
		}
		if p.Temperature == nil {
			tempOK = false
		} else {
			temp += *p.Temperature
		}
		if p.Precipitation == nil {
			precipOK = false
		} else {
			precip += *p.Precipitation
		}
	}
	w.Mode = "mixed"
	if w.Missing == len(hours) {
		w.Mode = "missing"
	} else if w.Forecast == len(hours) {
		w.Mode = "forecast"
	} else if w.Climatology == len(hours) {
		w.Mode = "climatology"
	}
	if tempOK && w.Missing == 0 {
		v := temp / float64(len(hours))
		w.Temperature = &v
	}
	if precipOK && w.Missing == 0 {
		w.Precipitation = &precip
	}
	return w
}
func difference(a, b d.Reading) *float64 {
	if a.Value == nil || b.Value == nil {
		return nil
	}
	v := *a.Value - *b.Value
	return &v
}
func routeReading(id string, base, eval []d.Hour, target float64) d.RouteReading {
	b, e := Aggregate(base, target), Aggregate(eval, target)
	r := d.RouteReading{
		CalendarCategory: calendarCategory(base),
		RouteID:          id,
		Baseline:         b,
		Evaluated:        e,
		Typical:          d.Missing("NO_TYPICAL_PROFILE"),
		Weather:          Weather(base),
		FleetAssessment:  AssessFleet(eval, target),
		Delta: d.Delta{
			Boardings:    e.Boardings - b.Boardings,
			Index:        difference(e.Index, b.Index),
			VehicleHours: difference(e.VehicleHours, b.VehicleHours),
		},
	}
	typical := 0.0
	ok := true
	for _, h := range base {
		if h.Typical == nil {
			ok = false
		} else {
			typical += *h.Typical
		}
	}
	if ok {
		r.Typical = d.Available(typical)
		if typical > 0 {
			v := (e.Boardings - typical) / typical
			r.Relative = &v
		}
	}
	return r
}
func Calculate(desc d.Descriptor, s d.Snapshot, hours []d.Hour) (d.Response, error) {
	if err := Validate(desc, s); err != nil {
		return d.Response{}, err
	}
	desc = Normalize(desc)
	byRoute := map[string][]d.Hour{}
	evaluated := map[string][]d.Hour{}
	for _, h := range hours {
		if err := ValidateHourNumbers(h); err != nil {
			return d.Response{}, d.Caused(503, "INVALID_INPUT_PROFILE", "Некорректные подготовленные данные.", "calculation", err)
		}
		if math.IsNaN(h.Boardings) || math.IsInf(h.Boardings, 0) || h.Boardings < 0 {
			return d.Response{}, d.Fail(503, "INVALID_MODEL_OUTPUT", "Модель вернула недопустимый прогноз.")
		}
		e, err := Scenario(h, desc.Overrides)
		if err != nil {
			return d.Response{}, err
		}
		byRoute[h.RouteID] = append(byRoute[h.RouteID], h)
		evaluated[h.RouteID] = append(evaluated[h.RouteID], e)
	}
	q := desc.Selection
	width := int(q.Window.To.Sub(q.Window.From) / time.Hour)
	for _, r := range q.RouteIDs {
		hs := byRoute[r]
		if len(hs) != width {
			return d.Response{}, d.Fail(503, "INCOMPLETE_COVERAGE", "Подготовленные входы покрывают не все часы.")
		}
		for i, h := range hs {
			if !h.Time.Equal(q.Window.From.Add(time.Duration(i) * time.Hour)) {
				return d.Response{}, d.Fail(503, "INCOMPLETE_COVERAGE", "Некорректная часовая сетка.")
			}
		}
	}
	result := d.Response{
		ID:         CalculationID(desc),
		Descriptor: desc,
		Provenance: s.Provenance,
		Target:     s.Target,
		Frames:     []d.Frame{},
		Totals:     []d.RouteReading{},
		Notices:    append([]d.Notice{}, s.Limitations...),
		Visualization: d.Visualization{
			StopColumns: d.StopColumns{
				DataAnchor:     "stop_position",
				ValueMetric:    "load_index",
				ApproachEffect: "decorative",
				ApproachLength: 20,
				Profile:        "ramp_to_stop",
			},
			Metric:     "load_index",
			Min:        0,
			Max:        1.5,
			Overflow:   "show_overflow",
			Thresholds: []float64{0.75, 1, 1.25},
			Reference:  s.Provenance.Reference,
			Label:      "Интенсивность относительно исторического эталона; не заполненность салона",
		},
	}
	step := 1
	if q.Resolution == "day" {
		step = 24
	}
	for i := 0; i < width; i += step {
		frame := d.Frame{
			Window: d.Window{
				From: q.Window.From.Add(time.Duration(i) * time.Hour),
				To:   q.Window.From.Add(time.Duration(i+step) * time.Hour),
			},
			Routes: []d.RouteReading{},
		}
		for _, r := range q.RouteIDs {
			frame.Routes = append(frame.Routes, routeReading(r, byRoute[r][i:i+step], evaluated[r][i:i+step], s.Target))
		}
		result.Frames = append(result.Frames, frame)
	}
	for _, r := range q.RouteIDs {
		result.Totals = append(result.Totals, routeReading(r, byRoute[r], evaluated[r], s.Target))
	}
	if err := ValidateResponseNumbers(result); err != nil {
		return d.Response{}, d.Caused(503, "INVALID_DERIVED_METRICS", "Расчёт дал недопустимые значения.", "calculation", err)
	}
	return result, nil
}
