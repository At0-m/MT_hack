package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"math"
	"sort"
	"time"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
	"tramflow/internal/inference"
)

func finitePositive(p *float64) bool {
	return p == nil || (!math.IsNaN(*p) && !math.IsInf(*p, 0) && *p >= 0)
}
func ValidateBundle(ctx context.Context, b *d.Bundle, c *contract.Contract, models *inference.Manager, allowSynthetic bool) error {
	s := b.Snapshot
	p := s.Provenance
	raw, _ := json.Marshal(s)
	if err := c.Validate("Snapshot", raw); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	if p.Policy != "scenario-formulas-v1" || s.Target != 0.75 || s.Quantile != 0.9 {
		return fmt.Errorf("unsupported scenario policy")
	}
	if p.Origin.IsZero() || p.CompleteThrough.IsZero() || b.Model.TrainCutoff.IsZero() || p.CompleteThrough.After(p.Origin) || b.Model.TrainCutoff.After(p.CompleteThrough) || b.Model.Version != p.Model || b.Model.Schema != p.Features {
		return fmt.Errorf("version/cutoff mismatch")
	}
	if _, err := engine.DefaultSelection(s); err != nil {
		return err
	}
	if (p.StopModel == "") != (p.StopFeatures == "") {
		return fmt.Errorf("stop model and schema must be supplied together")
	}
	synthetic := p.Mode == "synthetic_mock"
	if synthetic && !allowSynthetic {
		return fmt.Errorf("synthetic publication requires explicit --allow-synthetic")
	}
	if len(b.Routes) != len(s.Coverage) {
		return fmt.Errorf("route catalog does not match coverage")
	}
	catalog := map[string]bool{}
	for _, r := range b.Routes {
		if err := c.Validate("RouteDetail", r.Detail); err != nil {
			return fmt.Errorf("route: %w", err)
		}
		var detail struct {
			Route struct {
				ID string `json:"route_id"`
			} `json:"route"`
			Network string `json:"network_version"`
		}
		_ = json.Unmarshal(r.Detail, &detail)
		if detail.Network != p.Network || catalog[detail.Route.ID] {
			return fmt.Errorf("route network mismatch or duplicate")
		}
		catalog[detail.Route.ID] = true
		if len(r.Geometry) == 0 || string(r.Geometry) == "null" {
			return fmt.Errorf("route geometry is required for map serving")
		}
		if len(r.Geometry) > 0 && string(r.Geometry) != "null" {
			if err := c.Validate("RouteGeometry", r.Geometry); err != nil {
				return fmt.Errorf("geometry: %w", err)
			}
			var geo struct {
				Network  string `json:"network_version"`
				Route    string `json:"route_id"`
				Features []struct {
					Geometry struct {
						Type        string          `json:"type"`
						Coordinates json.RawMessage `json:"coordinates"`
					} `json:"geometry"`
				} `json:"features"`
			}
			_ = json.Unmarshal(r.Geometry, &geo)
			if geo.Network != p.Network || geo.Route != detail.Route.ID {
				return fmt.Errorf("geometry version mismatch")
			}
			for _, f := range geo.Features {
				var coords any
				_ = json.Unmarshal(f.Geometry.Coordinates, &coords)
				if !validCoordinates(coords) {
					return fmt.Errorf("invalid WGS84 coordinates")
				}
			}
		}
	}
	if err := c.Validate("QualityResponse", b.Quality); err != nil {
		return fmt.Errorf("quality: %w", err)
	}
	if err := c.Validate("SourceList", b.Sources); err != nil {
		return fmt.Errorf("sources: %w", err)
	}
	for _, raw := range []json.RawMessage{b.Quality, b.Sources} {
		var report map[string]any
		_ = json.Unmarshal(raw, &report)
		if report["forecast_snapshot_id"] != p.SnapshotID {
			return fmt.Errorf("report snapshot mismatch")
		}
	}
	if err := validateGeography(b.Routes); err != nil {
		return err
	}
	if err := validateWeatherMetadata(*b); err != nil {
		return err
	}
	sort.Slice(b.Hours, func(i, j int) bool {
		if b.Hours[i].RouteID == b.Hours[j].RouteID {
			return b.Hours[i].Time.Before(b.Hours[j].Time)
		}
		return b.Hours[i].RouteID < b.Hours[j].RouteID
	})
	actual := map[string]d.Hour{}
	expected := 0
	for _, h := range b.Hours {
		key := h.RouteID + "|" + h.Time.UTC().Format(time.RFC3339)
		if _, ok := actual[key]; ok {
			return fmt.Errorf("duplicate hourly input")
		}
		actual[key] = h
		if err := engine.ValidateHourNumbers(h); err != nil {
			return fmt.Errorf("hour %s: %w", key, err)
		}
		if !catalog[h.RouteID] || h.FeaturesAvailableAt.IsZero() || h.FeaturesAvailableAt.After(p.Origin) || h.Time.Unix()%3600 != 0 || h.Time.Nanosecond() != 0 {
			return fmt.Errorf("invalid hourly input or future features")
		}
		if !finitePositive(h.Fleet) || !finitePositive(h.Reference) || !finitePositive(h.Typical) || (h.Fleet != nil && *h.Fleet > 200) {
			return fmt.Errorf("invalid profile")
		}
		if h.Source != "observed_vehicle_profile" && h.Source != "manual_plan" && h.Source != "unavailable" {
			return fmt.Errorf("invalid fleet source")
		}
		if (h.Source == "unavailable") != (h.Fleet == nil) || (h.Source == "unavailable" && h.Proxy) || (h.Source == "manual_plan" && h.Proxy) || (h.Source == "observed_vehicle_profile" && !h.Proxy) {
			return fmt.Errorf("contradictory fleet source/value/proxy")
		}
		if synthetic {
			if h.SyntheticBoardings == nil || !finitePositive(h.SyntheticBoardings) || *h.SyntheticBoardings > engine.MaxHourlyBoardings {
				return fmt.Errorf("synthetic prediction missing")
			}
		} else if h.SyntheticBoardings != nil {
			return fmt.Errorf("synthetic prediction in real publication")
		}
		for _, v := range h.Features {
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > math.MaxFloat32 {
				return fmt.Errorf("nonfinite feature")
			}
		}
		if len(h.Features) > 258 || len(encode(h.Features)) > 65536 {
			return fmt.Errorf("feature blob limit exceeded")
		}
		for name := range h.Features {
			if len(name) > 128 {
				return fmt.Errorf("feature name limit exceeded")
			}
		}
		weather, _ := json.Marshal(h.Weather)
		if err := validateWeatherMode(h.Weather); err != nil {
			return err
		}
		if err := c.Validate("WeatherPoint", weather); err != nil {
			return fmt.Errorf("weather: %w", err)
		}
		if !h.Weather.Window.From.Equal(h.Time) || !h.Weather.Window.To.Equal(h.Time.Add(time.Hour)) {
			return fmt.Errorf("weather hour mismatch")
		}
		if h.Weather.Mode != "missing" && (h.Weather.AvailableAt == nil || h.Weather.AvailableAt.After(p.Origin)) {
			return fmt.Errorf("weather not available at origin")
		}
		if !synthetic && h.Weather.Mode == "forecast" && (!h.Weather.Verified || h.Weather.RunAt == nil || h.Weather.RunAt.After(p.Origin)) {
			return fmt.Errorf("weather issue time unverified")
		}
	}
	for _, cov := range s.Coverage {
		// Stop predictions require a separate model and prepared stop inputs.
		// Reject unsupported publication rather than repeating route predictions.
		if len(cov.Scopes) != 1 || cov.Scopes[0] != "route" || cov.StopStatus != "unavailable" {
			return fmt.Errorf("stop model adapter is not installed: publish route-only coverage")
		}
		if !catalog[cov.RouteID] {
			return fmt.Errorf("coverage route missing")
		}
		if err := engine.ValidateWindow(cov.Window); err != nil {
			return err
		}
		if cov.Window.From.Before(p.Origin) || cov.Window.To.Sub(p.Origin) > time.Duration(cov.MaxLead)*time.Hour {
			return fmt.Errorf("coverage outside lead range")
		}
		for t := cov.Window.From; t.Before(cov.Window.To); t = t.Add(time.Hour) {
			expected++
			if _, ok := actual[cov.RouteID+"|"+t.UTC().Format(time.RFC3339)]; !ok {
				return fmt.Errorf("incomplete hourly coverage")
			}
		}
	}
	if expected != len(b.Hours) || expected > 7440 {
		return fmt.Errorf("unexpected input rows or input limit exceeded")
	}
	if _, err := models.Prepare(ctx, b.Model, synthetic); err != nil {
		return err
	}
	// Verify the entire published grid before touching the active pointer.
	hours := append([]d.Hour(nil), b.Hours...)
	for start := 0; start < len(hours); start += 128 {
		end := min(start+128, len(hours))
		predictions, err := models.Predict(ctx, b.Model, s, hours[start:end])
		if err != nil {
			return fmt.Errorf("candidate inference: %w", err)
		}
		for offset, prediction := range predictions {
			if prediction > engine.MaxHourlyBoardings || math.IsNaN(prediction) || math.IsInf(prediction, 0) || prediction < 0 {
				return fmt.Errorf("prediction outside numeric admission bounds")
			}
			hours[start+offset].Boardings = prediction
		}
	}
	return validateCapabilities(s, hours, c)
}
func validCoordinates(v any) bool {
	a, ok := v.([]any)
	if !ok || len(a) == 0 {
		return false
	}
	if x, ok := a[0].(float64); ok {
		if len(a) != 2 {
			return false
		}
		y, ok := a[1].(float64)
		return ok && x >= -180 && x <= 180 && y >= -90 && y <= 90
	}
	for _, sub := range a {
		if !validCoordinates(sub) {
			return false
		}
	}
	return true
}
func encode(v any) []byte { b, _ := json.Marshal(v); return b }
func claim(ctx context.Context, tx pgx.Tx, kind, version string, payload any) error {
	if !engine.IDPattern.MatchString(version) {
		return fmt.Errorf("invalid version")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(encode(payload)))
	_, err := tx.Exec(ctx, "INSERT INTO data_versions(kind,version,digest) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", kind, version, digest)
	if err != nil {
		return err
	}
	var stored string
	if err = tx.QueryRow(ctx, "SELECT digest FROM data_versions WHERE kind=$1 AND version=$2", kind, version).Scan(&stored); err != nil {
		return err
	}
	if stored != digest {
		return fmt.Errorf("immutable %s version conflict: %s", kind, version)
	}
	return nil
}
func (s *Store) Publish(ctx context.Context, b d.Bundle) error {
	if _, err := engine.DefaultSelection(b.Snapshot); err != nil {
		return err
	}
	if err := validateGeography(b.Routes); err != nil {
		return err
	}
	if err := validateWeatherMetadata(b); err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741853)"); err != nil {
		return err
	}
	var current string
	err = tx.QueryRow(ctx, "SELECT snapshot_id FROM active_forecast_snapshot WHERE slot=1").Scan(&current)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if current != b.ExpectedPrevious {
		return fmt.Errorf("active snapshot changed: expected %q, found %q", b.ExpectedPrevious, current)
	}
	p := b.Snapshot.Provenance
	history, supply, reference, weather := []any{}, []any{}, []any{}, []any{}
	for _, h := range b.Hours {
		history = append(history, []any{
			h.RouteID,
			h.Time.UTC(),
			p.Origin.UTC(),
			h.Features,
			h.FeaturesAvailableAt.UTC(),
			h.SyntheticBoardings,
		})
		supply = append(supply, []any{h.RouteID, h.Time.UTC(), h.Fleet, h.Source, h.Proxy})
		reference = append(reference, []any{h.RouteID, h.Time.UTC(), h.Reference, h.Typical})
		weather = append(weather, []any{h.RouteID, h.Time.UTC(), h.Weather})
	}
	var conflicts int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM models WHERE metadata->>'feature_schema_version'=$1
 AND COALESCE(metadata->>'schema_sha256','')<>$2`, p.Features, b.Model.SchemaSHA256).Scan(&conflicts); err != nil {
		return err
	}
	if conflicts > 0 {
		return fmt.Errorf("feature schema artifact changed under existing version")
	}
	for _, v := range []struct {
		kind, version string
		payload       any
	}{
		{"network", p.Network, b.Routes},
		{"history", p.History, history},
		{"supply", p.Supply, supply},
		{"reference", p.Reference, reference},
		{"weather", p.Weather, []any{b.WeatherMetadata, weather}},
		{"features", p.Features, b.Model.Columns},
		{"feature_schema_artifact", p.Features, featureSchemaClaim(b.Model)},
		{"model", p.Model, b.Model},
	} {
		if err = claim(ctx, tx, v.kind, v.version, v.payload); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, "INSERT INTO models(version,metadata) VALUES($1,$2) ON CONFLICT DO NOTHING", p.Model, encode(b.Model)); err != nil {
		return err
	}
	for _, r := range b.Routes {
		var detail struct {
			Route struct {
				ID string `json:"route_id"`
			} `json:"route"`
		}
		_ = json.Unmarshal(r.Detail, &detail)
		if _, err = tx.Exec(ctx, "INSERT INTO routes(network_version,route_id,detail) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", p.Network, detail.Route.ID, r.Detail); err != nil {
			return err
		}
		if len(r.Geometry) > 0 && string(r.Geometry) != "null" {
			if err = importGeometry(ctx, tx, p.Network, detail.Route.ID, r.Geometry); err != nil {
				return err
			}
		}
	}
	if _, err = tx.Exec(ctx, "INSERT INTO weather_snapshots(version,metadata) VALUES($1,$2) ON CONFLICT DO NOTHING", p.Weather, encode(b.WeatherMetadata)); err != nil {
		return err
	}
	if err = insertHours(ctx, tx, p, b.Hours); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO forecast_snapshots(id,metadata,model_version) VALUES($1,$2,$3)", p.SnapshotID, encode(b.Snapshot), p.Model); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO snapshot_reports VALUES($1,$2,$3)", p.SnapshotID, b.Quality, b.Sources); err != nil {
		return err
	}
	if current != "" {
		if _, err = tx.Exec(ctx, "UPDATE forecast_snapshots SET deactivated_at=now() WHERE id=$1", current); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, "INSERT INTO active_forecast_snapshot VALUES(1,$1) ON CONFLICT(slot) DO UPDATE SET snapshot_id=EXCLUDED.snapshot_id", p.SnapshotID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func importGeometry(ctx context.Context, tx pgx.Tx, network, route string, raw []byte) error {
	var geo struct {
		Features []struct {
			ID       string          `json:"id"`
			Geometry json.RawMessage `json:"geometry"`
			Props    struct {
				Kind      string `json:"kind"`
				Route     string `json:"route_id"`
				Pattern   string `json:"route_pattern_id"`
				RouteStop string `json:"route_stop_id"`
				Stop      string `json:"stop_id"`
				Name      string `json:"name"`
				Sequence  int    `json:"sequence"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &geo); err != nil {
		return err
	}
	for _, f := range geo.Features {
		if f.Props.Route != route {
			return fmt.Errorf("feature route mismatch")
		}
		if f.Props.Kind == "route_line" {
			if f.ID != f.Props.Pattern {
				return fmt.Errorf("pattern ID mismatch")
			}
			if _, err := tx.Exec(ctx, "INSERT INTO route_patterns VALUES($1,$2,$3,ST_SetSRID(ST_GeomFromGeoJSON($4),4326)) ON CONFLICT DO NOTHING", network, route, f.Props.Pattern, string(f.Geometry)); err != nil {
				return err
			}
		}
	}
	for _, f := range geo.Features {
		if f.Props.Kind == "stop" {
			if f.ID != f.Props.RouteStop {
				return fmt.Errorf("stop ID mismatch")
			}
			result, err := tx.Exec(ctx, `INSERT INTO stops VALUES($1,$2,$3,ST_SetSRID(ST_GeomFromGeoJSON($4),4326))
 ON CONFLICT(network_version,stop_id) DO UPDATE SET name=EXCLUDED.name
 WHERE stops.name=EXCLUDED.name AND ST_Equals(stops.location,EXCLUDED.location)`, network, f.Props.Stop, f.Props.Name, string(f.Geometry))
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return fmt.Errorf("conflicting stored stop %s", f.Props.Stop)
			}
			if _, err := tx.Exec(ctx, "INSERT INTO route_stops(network_version,route_id,pattern_id,route_stop_id,stop_id,sequence) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING", network, route, f.Props.Pattern, f.Props.RouteStop, f.Props.Stop, f.Props.Sequence); err != nil {
				return err
			}
		}
	}
	return nil
}

// Keep tombstones for 410; retain dependencies so pinned old calculations remain reproducible.
func (s *Store) Expire(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `UPDATE forecast_snapshots s SET deleted_at=now() WHERE deleted_at IS NULL AND deactivated_at IS NOT NULL AND deactivated_at+make_interval(hours => (metadata->>'retention_hours_after_deactivation')::int)<now() AND NOT EXISTS(SELECT 1 FROM active_forecast_snapshot a WHERE a.snapshot_id=s.id)`)
	return err
}
