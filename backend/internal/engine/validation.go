package engine

import (
	"math"
	"regexp"
	"sort"
	"time"
	d "tramflow/internal/domain"
)

var IDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,95}$`)

func ValidateWindow(w d.Window) error {
	if w.From.IsZero() || !w.From.Before(w.To) || w.From.Unix()%3600 != 0 || w.To.Unix()%3600 != 0 || w.From.Nanosecond() != 0 || w.To.Nanosecond() != 0 {
		return d.Fail(422, "INVALID_WINDOW", "Интервал должен состоять из целых часов, from < to.")
	}
	return nil
}
func Milli(f *float64) int64 {
	if f == nil {
		return 1000
	}
	return int64(math.Round(*f * 1000))
}
func FactorMillis(o *d.Overrides) [3]int64 {
	if o == nil || o.Factors == nil {
		return [3]int64{1000, 1000, 1000}
	}
	return [3]int64{Milli(o.Factors.Weather), Milli(o.Factors.Event), Milli(o.Factors.Season)}
}
func Validate(desc d.Descriptor, s d.Snapshot) error {
	q := desc.Selection
	if err := ValidateWindow(q.Window); err != nil {
		return err
	}
	if q.SnapshotID != s.Provenance.SnapshotID || !IDPattern.MatchString(q.SnapshotID) {
		return d.Fail(422, "INVALID_SELECTION", "Некорректная версия прогноза.")
	}
	if err := ValidateCalendar(q); err != nil {
		return err
	}
	if !contains(s.Views, q.View) && q.View != "custom" {
		return d.Fail(422, "UNSUPPORTED_VIEW", "Представление не опубликовано.")
	}
	if q.SpatialDetail != "route" && q.SpatialDetail != "route_stop" {
		return d.Fail(422, "UNSUPPORTED_SCOPE", "Укажите route или route_stop.")
	}
	width := int(q.Window.To.Sub(q.Window.From) / time.Hour)
	frames := width
	if q.Resolution == "day" {
		frames /= 24
	}
	if len(q.RouteIDs) < 1 || len(q.RouteIDs) > 10 || len(q.RouteIDs)*width > 7440 || len(q.RouteIDs)*frames > 960 {
		return d.Fail(422, "CELL_LIMIT", "Превышен лимит маршрутов или ячеек; выберите суточную агрегацию.")
	}
	seen := map[string]bool{}
	for _, r := range q.RouteIDs {
		if !IDPattern.MatchString(r) || seen[r] {
			return d.Fail(422, "INVALID_ROUTE", "Некорректные или повторяющиеся маршруты.")
		}
		seen[r] = true
		found := false
		for _, c := range s.Coverage {
			if c.RouteID == r {
				found = true
				if !contains(c.Scopes, q.SpatialDetail) || (q.SpatialDetail == "route_stop" && c.StopStatus == "unavailable") {
					return d.Fail(422, "STOP_FORECAST_UNAVAILABLE", "Прогноз по остановкам пока не опубликован.")
				}
				if q.Window.From.Before(c.Window.From) || q.Window.To.After(c.Window.To) || q.Window.From.Before(s.Provenance.Origin) || q.Window.To.Sub(s.Provenance.Origin) > time.Duration(c.MaxLead)*time.Hour || !contains(c.Resolutions, q.Resolution) {
					return d.Fail(422, "OUTSIDE_COVERAGE", "Окно выходит за опубликованное покрытие маршрута.")
				}
			}
		}
		if !found {
			return d.Fail(422, "ROUTE_UNSUPPORTED", "Маршрут отсутствует в закреплённом прогнозе.")
		}
	}
	if desc.Kind == "forecast" && desc.Overrides == nil {
		return nil
	}
	if desc.Kind != "scenario" || desc.Overrides == nil {
		return d.Fail(422, "INVALID_CALCULATION", "Некорректный тип расчёта.")
	}
	o := desc.Overrides
	if err := ValidateWindow(o.Window); err != nil {
		return err
	}
	if o.Window.From.Before(q.Window.From) || o.Window.To.After(q.Window.To) || (o.Fleet == nil && o.Factors == nil) {
		return d.Fail(422, "INVALID_SCENARIO_WINDOW", "Сценарий должен входить в окно запроса и содержать поправки.")
	}
	if o.Fleet != nil {
		f := o.Fleet
		if (f.Kind == "absolute" && (f.Absolute == nil || f.Delta != nil || *f.Absolute < 0 || *f.Absolute > 200)) || (f.Kind == "delta" && (f.Delta == nil || f.Absolute != nil || *f.Delta < -200 || *f.Delta > 200)) || (f.Kind != "absolute" && f.Kind != "delta") {
			return d.Fail(422, "INVALID_FLEET", "Выпуск должен быть absolute или delta в допустимом диапазоне.")
		}
	}
	if o.Factors != nil {
		for _, p := range []*float64{o.Factors.Weather, o.Factors.Event, o.Factors.Season} {
			if p != nil && (math.IsNaN(*p) || math.IsInf(*p, 0) || *p < 0.5 || *p > 2 || math.Abs(*p*1000-math.Round(*p*1000)) > 1e-9) {
				return d.Fail(422, "INVALID_FACTOR", "Поправка должна быть 0.5–2, не более трёх знаков.")
			}
		}
	}
	m := FactorMillis(o)
	product := m[0] * m[1] * m[2]
	if product < 250000000 || product > 4000000000 {
		return d.Fail(422, "FACTOR_PRODUCT_OUT_OF_RANGE", "Произведение поправок должно быть 0.25–4.")
	}
	return nil
}
func contains(a []string, x string) bool {
	for _, v := range a {
		if v == x {
			return true
		}
	}
	return false
}
func Normalize(desc d.Descriptor) d.Descriptor {
	desc.Selection.RouteIDs = append([]string(nil), desc.Selection.RouteIDs...)
	sort.Strings(desc.Selection.RouteIDs)
	desc.Selection.Window = d.Window{From: desc.Selection.Window.From.UTC(), To: desc.Selection.Window.To.UTC()}
	if desc.Overrides != nil {
		o := *desc.Overrides
		o.Window = d.Window{From: o.Window.From.UTC(), To: o.Window.To.UTC()}
		m := FactorMillis(&o)
		f := d.Factors{}
		w, e, s := float64(m[0])/1000, float64(m[1])/1000, float64(m[2])/1000
		f.Weather = &w
		f.Event = &e
		f.Season = &s
		o.Factors = &f
		desc.Overrides = &o
	}
	return desc
}
