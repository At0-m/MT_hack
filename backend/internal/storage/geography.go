package storage

import (
	"encoding/json"
	"fmt"
	"math"
	d "tramflow/internal/domain"
)

type importedStop struct {
	ID       string `json:"route_stop_id"`
	Stop     string `json:"stop_id"`
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	Position struct {
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	} `json:"position"`
}

func samePosition(a, b importedStop) bool {
	return math.Abs(a.Position.Longitude-b.Position.Longitude) <= 1e-9 && math.Abs(a.Position.Latitude-b.Position.Latitude) <= 1e-9
}

// Detail and geometry must agree before either representation reaches storage.
func validateGeography(routes []d.RouteImport) error {
	physical := map[string]importedStop{}
	patterns, occurrences := map[string]bool{}, map[string]bool{}
	for _, route := range routes {
		var detail struct {
			Route struct {
				ID string `json:"route_id"`
			} `json:"route"`
			Patterns []struct {
				ID    string         `json:"route_pattern_id"`
				Stops []importedStop `json:"stops"`
			} `json:"patterns"`
		}
		if err := json.Unmarshal(route.Detail, &detail); err != nil {
			return err
		}
		stops := map[string]importedStop{}
		owners := map[string]string{}
		lines := map[string]bool{}
		for _, pattern := range detail.Patterns {
			if patterns[pattern.ID] {
				return fmt.Errorf("duplicate route pattern %s", pattern.ID)
			}
			patterns[pattern.ID], lines[pattern.ID] = true, true
			sequences := map[int]bool{}
			for _, stop := range pattern.Stops {
				if occurrences[stop.ID] || sequences[stop.Sequence] {
					return fmt.Errorf("duplicate stop occurrence or sequence")
				}
				occurrences[stop.ID], sequences[stop.Sequence] = true, true
				if old, ok := physical[stop.Stop]; ok && (old.Name != stop.Name || !samePosition(old, stop)) {
					return fmt.Errorf("conflicting physical stop %s", stop.Stop)
				}
				physical[stop.Stop], stops[stop.ID], owners[stop.ID] = stop, stop, pattern.ID
			}
		}
		if len(route.Geometry) == 0 || string(route.Geometry) == "null" {
			continue
		}
		var geo struct {
			Features []struct {
				ID       string `json:"id"`
				Geometry struct {
					Coordinates json.RawMessage `json:"coordinates"`
				} `json:"geometry"`
				Properties struct {
					Kind     string `json:"kind"`
					Route    string `json:"route_id"`
					Pattern  string `json:"route_pattern_id"`
					ID       string `json:"route_stop_id"`
					Stop     string `json:"stop_id"`
					Name     string `json:"name"`
					Sequence int    `json:"sequence"`
				} `json:"properties"`
			} `json:"features"`
		}
		if err := json.Unmarshal(route.Geometry, &geo); err != nil {
			return err
		}
		seenStops, seenLines := map[string]bool{}, map[string]bool{}
		for _, feature := range geo.Features {
			p := feature.Properties
			if p.Route != detail.Route.ID {
				return fmt.Errorf("feature route mismatch")
			}
			if p.Kind == "route_line" {
				if feature.ID != p.Pattern || !lines[p.Pattern] || seenLines[p.Pattern] {
					return fmt.Errorf("geometry pattern mismatch")
				}
				seenLines[p.Pattern] = true
				continue
			}
			stop, ok := stops[p.ID]
			if !ok || seenStops[p.ID] || feature.ID != p.ID || owners[p.ID] != p.Pattern || stop.Stop != p.Stop || stop.Name != p.Name || stop.Sequence != p.Sequence {
				return fmt.Errorf("geometry stop mismatch")
			}
			var coords []float64
			if err := json.Unmarshal(feature.Geometry.Coordinates, &coords); err != nil || len(coords) != 2 {
				return fmt.Errorf("invalid stop coordinates")
			}
			other := stop
			other.Position.Longitude, other.Position.Latitude = coords[0], coords[1]
			if !samePosition(stop, other) {
				return fmt.Errorf("detail/geometry stop position mismatch")
			}
			seenStops[p.ID] = true
		}
		if len(stops) != len(seenStops) || len(lines) != len(seenLines) {
			return fmt.Errorf("incomplete route geometry")
		}
	}
	return nil
}
