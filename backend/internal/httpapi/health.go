package httpapi

import (
	"net/http"
	"tramflow/internal/engine"
)

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.CheckSchema(r.Context()); err != nil {
		s.fail(w, err)
		return
	}
	snap, model, err := s.Store.Active(r.Context())
	if err == nil {
		_, err = s.Models.Prepare(r.Context(), model, snap.Provenance.Mode == "synthetic_mock")
	}
	if err == nil {
		_, err = engine.DefaultSelection(snap)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.write(w, r, map[string]any{
		"status":    "ok",
		"component": "api",
		"checks": []any{
			map[string]any{"name": "postgres_postgis_schema", "ok": true},
			map[string]any{"name": "published_snapshot_model", "ok": true},
		},
	}, false)
}
