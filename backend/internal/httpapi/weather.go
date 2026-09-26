package httpapi

import (
	"net/http"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
)

func (s *Server) weather(w http.ResponseWriter, r *http.Request) {
	id, err := param(r, "forecast_snapshot_id")
	if err != nil {
		s.fail(w, err)
		return
	}
	route, err := param(r, "route_id")
	if err != nil {
		s.fail(w, err)
		return
	}
	q := r.URL.Query()
	from, e1 := time.Parse(time.RFC3339, q.Get("from"))
	to, e2 := time.Parse(time.RFC3339, q.Get("to"))
	if e1 != nil || e2 != nil || len(q["from"]) != 1 || len(q["to"]) != 1 {
		s.problem(w, 400, "INVALID_QUERY", "Нужны from/to в RFC3339.")
		return
	}
	snap, _, err := s.Store.Snapshot(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	window := d.Window{From: from, To: to}
	if err = engine.ValidateWeatherWindow(route, window, snap); err != nil {
		s.fail(w, err)
		return
	}
	hours, err := s.Store.Hours(r.Context(), snap, engine.SelectionInput{Routes: []string{route}, Window: window})
	if err != nil {
		s.fail(w, err)
		return
	}
	if len(hours) != int(to.Sub(from)/time.Hour) {
		s.problem(w, 503, "INCOMPLETE_COVERAGE", "Погода покрывает не все часы.")
		return
	}
	points := []d.WeatherPoint{}
	for _, h := range hours {
		points = append(points, h.Weather)
	}
	provider := "open-meteo"
	if snap.Provenance.Mode == "synthetic_mock" {
		provider = "synthetic_mock"
	}
	s.write(w, r, map[string]any{
		"weather_snapshot_id":  snap.Provenance.Weather,
		"forecast_snapshot_id": id,
		"route_id":             route,
		"location_id":          route,
		"provider":             provider,
		"retrieved_at":         snap.Created.UTC(),
		"source_id":            snap.Provenance.Weather,
		"window":               d.Window{From: from.UTC(), To: to.UTC()},
		"points":               points,
		"notices":              snap.Limitations,
	}, true)
}
