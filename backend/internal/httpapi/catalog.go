package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"tramflow/internal/engine"
)

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !engine.IDPattern.MatchString(id) {
		s.problem(w, 400, "INVALID_ID", "Некорректный идентификатор.")
		return
	}
	snap, _, err := s.Store.Snapshot(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, snap, true)
}

func (s *Server) routes(w http.ResponseWriter, r *http.Request) {
	network, err := param(r, "network_version")
	if err != nil {
		s.fail(w, err)
		return
	}
	items, err := s.Store.Routes(r.Context(), network)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, map[string]any{"network_version": network, "items": items}, true)
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) { s.routeOperation(w, r, false) }

func (s *Server) geometry(w http.ResponseWriter, r *http.Request) { s.routeOperation(w, r, true) }

func (s *Server) routeOperation(w http.ResponseWriter, r *http.Request, geometry bool) {
	network, err := param(r, "network_version")
	if err != nil {
		s.fail(w, err)
		return
	}
	id := r.PathValue("id")
	if !engine.IDPattern.MatchString(id) {
		s.problem(w, 400, "INVALID_ID", "Некорректный маршрут.")
		return
	}
	var result json.RawMessage
	if geometry {
		result, err = s.Store.Geometry(r.Context(), network, id)
	} else {
		result, err = s.Store.Route(r.Context(), network, id)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, result, true)
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	id, err := param(r, "forecast_snapshot_id")
	if err != nil {
		s.fail(w, err)
		return
	}
	if _, _, err = s.Store.Snapshot(r.Context(), id); err != nil {
		s.fail(w, err)
		return
	}
	kind := "model-quality"
	if strings.HasSuffix(r.URL.Path, "data-sources") {
		kind = "data-sources"
	}
	raw, err := s.Store.Report(r.Context(), id, kind)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, raw, true)
}
