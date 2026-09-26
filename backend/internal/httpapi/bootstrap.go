package httpapi

import (
	"net/http"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
)

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	snapshot, _, err := s.Store.Active(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	routes, err := s.Store.Routes(r.Context(), snapshot.Provenance.Network)
	if err != nil {
		s.fail(w, err)
		return
	}
	selection, err := engine.DefaultSelection(snapshot)
	if err != nil {
		s.fail(w, err)
		return
	}
	response := d.Bootstrap{
		APIVersion: "1.2.0",
		Timezone:   "Europe/Moscow",
		Locale:     "ru-RU",
		Snapshot:   snapshot,
		Routes:     routes,
		Selection:  selection,
		Map: d.MapConfiguration{
			Renderer:     "maplibre",
			StyleURL:     "https://tiles.openfreemap.org/styles/liberty",
			FallbackPath: "/map-fallback.json",
			Attribution:  "© OpenStreetMap contributors / OpenFreeMap",
			Center:       d.Position{Longitude: 37.6173, Latitude: 55.7558},
			Zoom:         11,
		},
		Limits: d.Limits{
			Routes:              10,
			WindowHours:         744,
			HourlyCells:         7440,
			BodyBytes:           65536,
			ExportRows:          960,
			ExportBytes:         4194304,
			TimeoutMS:           int(s.Timeout.Milliseconds()),
			ConcurrentInference: 2,
			ResponseCells:       960,
			CustomDays:          engine.MaxCustomDays,
			StopCells:           960,
		},
		Capabilities: d.Capabilities{
			Scopes:             []string{"route"},
			ScenarioFleet:      true,
			ScenarioFactors:    []string{"weather", "event", "season"},
			CSV:                true,
			Summary:            true,
			CustomPeriod:       true,
			SessionAuth:        true,
			DecorativeApproach: true,
		},
	}
	s.write(w, r, response, false)
}
