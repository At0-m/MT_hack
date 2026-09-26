package httpapi

import (
	"net/http"
	d "tramflow/internal/domain"
)

func (s *Server) forecast(w http.ResponseWriter, r *http.Request) { s.calculate(w, r, false) }

func (s *Server) scenario(w http.ResponseWriter, r *http.Request) { s.calculate(w, r, true) }

func (s *Server) calculate(w http.ResponseWriter, r *http.Request, scenario bool) {
	var q d.Query
	name, kind := "ForecastQuery", "forecast"
	if scenario {
		name, kind = "ScenarioQuery", "scenario"
	}
	if !s.body(w, r, name, &q) {
		return
	}
	result, err := s.Service.Calculate(r.Context(), d.Descriptor{Kind: kind, Selection: q.Selection, Overrides: q.Overrides})
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, result, false)
}
