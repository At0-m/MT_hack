package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
	"tramflow/migrations"
)

type Store struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string, max int32) (*Store, error) {
	c, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	c.MaxConns = max
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = 3 * time.Second
	c.ConnConfig.RuntimeParams["statement_timeout"] = "5000"
	c.ConnConfig.RuntimeParams["application_name"] = "tramflow"
	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	return &Store{p}, nil
}
func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741852)"); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return err
	}
	var v int
	if exists {
		if err = tx.QueryRow(ctx, "SELECT COALESCE(max(version),0) FROM schema_migrations").Scan(&v); err != nil {
			return err
		}
	}
	if v > 5 {
		return fmt.Errorf("unsupported database schema %d", v)
	}
	for i, name := range []string{"001_initial.sql", "002_sessions_anchors.sql", "003_weather_geography.sql", "004_login_limits.sql", "005_calendar_flags.sql"} {
		if i+1 <= v {
			continue
		}
		sql, err := migrations.Files.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (s *Store) CheckSchema(ctx context.Context) error {
	var v int
	var ext string
	err := s.Pool.QueryRow(ctx, "SELECT (SELECT max(version) FROM schema_migrations),postgis_version()").Scan(&v, &ext)
	if err != nil {
		return err
	}
	if v != 5 || ext == "" {
		return fmt.Errorf("schema not ready")
	}
	return nil
}
func dependency(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return d.Caused(504, "DEPENDENCY_TIMEOUT", "Превышено время ожидания данных.", "postgres", err)
	}
	return d.Caused(503, "DATABASE_UNAVAILABLE", "Хранилище временно недоступно.", "postgres", err)
}
func decodeSnapshot(meta, model []byte) (d.Snapshot, d.Model, error) {
	var s d.Snapshot
	var m d.Model
	if err := json.Unmarshal(meta, &s); err != nil {
		return s, m, dependency(err)
	}
	if err := json.Unmarshal(model, &m); err != nil {
		return s, m, dependency(err)
	}
	return s, m, nil
}
func (s *Store) Snapshot(ctx context.Context, id string) (d.Snapshot, d.Model, error) {
	var meta, model []byte
	var deleted *time.Time
	err := s.Pool.QueryRow(ctx, "SELECT s.metadata,m.metadata,s.deleted_at FROM forecast_snapshots s JOIN models m ON m.version=s.model_version WHERE s.id=$1", id).Scan(&meta, &model, &deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Snapshot{}, d.Model{}, d.Fail(404, "SNAPSHOT_NOT_FOUND", "Версия прогноза не найдена.")
	}
	if err != nil {
		return d.Snapshot{}, d.Model{}, dependency(err)
	}
	if deleted != nil {
		return d.Snapshot{}, d.Model{}, d.Fail(410, "SNAPSHOT_EXPIRED", "Версия прогноза удалена после срока хранения.")
	}
	return decodeSnapshot(meta, model)
}
func (s *Store) Active(ctx context.Context) (d.Snapshot, d.Model, error) {
	var meta, model []byte
	err := s.Pool.QueryRow(ctx, "SELECT s.metadata,m.metadata FROM active_forecast_snapshot a JOIN forecast_snapshots s ON s.id=a.snapshot_id JOIN models m ON m.version=s.model_version WHERE a.slot=1 AND s.deleted_at IS NULL").Scan(&meta, &model)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Snapshot{}, d.Model{}, d.Fail(503, "NO_ACTIVE_SNAPSHOT", "Опубликованный прогноз недоступен.")
	}
	if err != nil {
		return d.Snapshot{}, d.Model{}, dependency(err)
	}
	return decodeSnapshot(meta, model)
}
func (s *Store) Hours(ctx context.Context, snap d.Snapshot, q engine.SelectionInput) ([]d.Hour, error) {
	p := snap.Provenance
	rows, err := s.Pool.Query(ctx, `SELECT f.route_id,f.target_hour,f.calendar_flags,f.available_at,f.synthetic_boardings,
 sp.fleet,sp.source,sp.is_proxy,r.reference,r.typical,w.point
 FROM prepared_features f
 JOIN supply_profiles sp ON sp.version=$5 AND sp.route_id=f.route_id AND sp.target_hour=f.target_hour
 JOIN reference_profiles r ON r.version=$6 AND r.route_id=f.route_id AND r.target_hour=f.target_hour
 JOIN weather_points w ON w.version=$7 AND w.route_id=f.route_id AND w.target_hour=f.target_hour
 WHERE f.history_version=$1 AND f.schema_version=$2 AND f.origin=$3 AND f.route_id=ANY($4::text[])
 AND f.target_hour >= $8 AND f.target_hour < $9 ORDER BY f.route_id,f.target_hour`, p.History, p.Features, p.Origin, q.Routes, p.Supply, p.Reference, p.Weather, q.Window.From, q.Window.To)
	if err != nil {
		return nil, dependency(err)
	}
	defer rows.Close()
	out := []d.Hour{}
	for rows.Next() {
		var h d.Hour
		var features, weather []byte
		if err = rows.Scan(&h.RouteID, &h.Time, &features, &h.FeaturesAvailableAt, &h.SyntheticBoardings, &h.Fleet, &h.Source, &h.Proxy, &h.Reference, &h.Typical, &weather); err != nil {
			return nil, dependency(err)
		}
		if err = json.Unmarshal(features, &h.Features); err != nil {
			return nil, dependency(err)
		}
		if err = json.Unmarshal(weather, &h.Weather); err != nil {
			return nil, dependency(err)
		}
		h.Time = h.Time.UTC()
		out = append(out, h)
	}
	if err = rows.Err(); err != nil {
		return nil, dependency(err)
	}
	return out, nil
}
func (s *Store) Routes(ctx context.Context, network string) ([]json.RawMessage, error) {
	rows, err := s.Pool.Query(ctx, "SELECT detail->'route' FROM routes WHERE network_version=$1 ORDER BY route_id", network)
	if err != nil {
		return nil, dependency(err)
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, dependency(err)
		}
		out = append(out, json.RawMessage(b))
	}
	if err = rows.Err(); err != nil {
		return nil, dependency(err)
	}
	if len(out) == 0 {
		return nil, d.Fail(404, "NETWORK_NOT_FOUND", "Версия сети не найдена.")
	}
	return out, nil
}
func (s *Store) Route(ctx context.Context, network, route string) (json.RawMessage, error) {
	var b []byte
	err := s.Pool.QueryRow(ctx, "SELECT detail FROM routes WHERE network_version=$1 AND route_id=$2", network, route).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, d.Fail(404, "ROUTE_NOT_FOUND", "Маршрут не найден.")
	}
	if err != nil {
		return nil, dependency(err)
	}
	return b, nil
}
func (s *Store) Geometry(ctx context.Context, network, route string) (json.RawMessage, error) {
	detail, err := s.Route(ctx, network, route)
	if err != nil {
		return nil, err
	}
	var info struct {
		Source string `json:"source_id"`
	}
	_ = json.Unmarshal(detail, &info)
	const query = `
		SELECT feature FROM (
			SELECT jsonb_build_object(
				'type', 'Feature', 'id', pattern_id,
				'geometry', ST_AsGeoJSON(geometry)::jsonb,
				'properties', jsonb_build_object(
					'kind', 'route_line', 'route_id', route_id,
					'route_pattern_id', pattern_id
				)
			) AS feature
			FROM route_patterns WHERE network_version=$1 AND route_id=$2
			UNION ALL
			SELECT jsonb_build_object(
				'type', 'Feature', 'id', rs.route_stop_id,
				'geometry', ST_AsGeoJSON(st.location)::jsonb,
				'properties', jsonb_build_object(
					'kind', 'stop', 'route_id', rs.route_id,
					'route_pattern_id', rs.pattern_id,
					'route_stop_id', rs.route_stop_id, 'stop_id', rs.stop_id,
					'name', st.name, 'sequence', rs.sequence
				)
			)
			FROM route_stops rs JOIN stops st USING(network_version,stop_id)
			WHERE rs.network_version=$1 AND rs.route_id=$2
		) AS features
		ORDER BY feature->>'id'`
	rows, err := s.Pool.Query(ctx, query, network, route)
	if err != nil {
		return nil, dependency(err)
	}
	defer rows.Close()
	features := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, dependency(err)
		}
		features = append(features, b)
	}
	if err = rows.Err(); err != nil {
		return nil, dependency(err)
	}
	if len(features) == 0 {
		return nil, d.Fail(404, "GEOMETRY_UNAVAILABLE", "Геометрия маршрута отсутствует.")
	}
	b, err := json.Marshal(map[string]any{
		"type":            "FeatureCollection",
		"network_version": network,
		"route_id":        route,
		"source_id":       info.Source,
		"features":        features,
	})
	return b, err
}
func (s *Store) Report(ctx context.Context, id, kind string) (json.RawMessage, error) {
	column := "quality"
	if kind == "data-sources" {
		column = "sources"
	}
	var b []byte
	err := s.Pool.QueryRow(ctx, "SELECT "+column+" FROM snapshot_reports WHERE snapshot_id=$1", id).Scan(&b)
	if err != nil {
		return nil, dependency(err)
	}
	return b, nil
}
