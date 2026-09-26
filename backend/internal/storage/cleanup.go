package storage

import (
	"context"
	"fmt"
)

// Explicit offline maintenance. Tombstones, manifests and immutable digests stay
// available; large profiles disappear only when no retained snapshot uses them.
// One minute after tombstoning protects requests admitted before expiry.
func (s *Store) CollectData(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741853)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM user_sessions WHERE expires_at<=now()`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM login_limits WHERE window_start<=now()-interval '1 minute'`); err != nil {
		return err
	}
	const retained = `SELECT metadata->'provenance'->>%s FROM forecast_snapshots
 WHERE deleted_at IS NULL OR deleted_at>now()-interval '1 minute'`
	// Identifiers and keys come solely from this fixed internal table.
	for _, item := range []struct{ table, column, key string }{
		{"prepared_features", "history_version", "history_version"},
		{"supply_profiles", "version", "supply_profile_version"},
		{"reference_profiles", "version", "reference_version"},
		{"weather_points", "version", "weather_snapshot_id"},
		{"weather_snapshots", "version", "weather_snapshot_id"},
		{"route_stops", "network_version", "network_version"},
		{"stops", "network_version", "network_version"},
		{"route_patterns", "network_version", "network_version"},
		{"routes", "network_version", "network_version"},
	} {
		query := "DELETE FROM " + item.table + " WHERE " + item.column + " NOT IN (" + fmt.Sprintf(retained, "'"+item.key+"'") + ")"
		if _, err = tx.Exec(ctx, query); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
