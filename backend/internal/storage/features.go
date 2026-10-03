package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/telemetry"
)

// Only requested misses are fetched; both caller and repository bound the batch.
func (s *Store) Features(ctx context.Context, snap d.Snapshot, hours []d.Hour) ([]d.Hour, error) {
	done := telemetry.Timer("storage_operation_duration_seconds", telemetry.Labels{"operation": "features"})
	defer done()
	if len(hours) == 0 {
		return []d.Hour{}, nil
	}
	if len(hours) > 128 {
		return nil, fmt.Errorf("feature batch exceeds 128 rows")
	}
	routes, times := make([]string, len(hours)), make([]time.Time, len(hours))
	for i, h := range hours {
		routes[i], times[i] = h.RouteID, h.Time
	}
	p := snap.Provenance
	rows, err := s.Pool.Query(ctx, `SELECT k.ordinality,f.features,f.available_at
 FROM unnest($4::text[],$5::timestamptz[]) WITH ORDINALITY k(route_id,target_hour,ordinality)
 JOIN prepared_features f ON f.history_version=$1 AND f.schema_version=$2 AND f.origin=$3
 AND f.route_id=k.route_id AND f.target_hour=k.target_hour ORDER BY k.ordinality`, p.History, p.Features, p.Origin, routes, times)
	if err != nil {
		return nil, dependency(err)
	}
	defer rows.Close()
	out := append([]d.Hour(nil), hours...)
	count := 0
	for rows.Next() {
		var index int
		var raw []byte
		var available time.Time
		if err := rows.Scan(&index, &raw, &available); err != nil {
			return nil, dependency(err)
		}
		if index != count+1 {
			return nil, d.Fail(503, "INCOMPLETE_FEATURES", "Подготовленные признаки неполны.")
		}
		out[count].Features = nil // Do not expand the lightweight map shared with Hours.
		if err := json.Unmarshal(raw, &out[count].Features); err != nil {
			return nil, dependency(err)
		}
		out[count].FeaturesAvailableAt = available
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, dependency(err)
	}
	if count != len(hours) {
		return nil, d.Fail(503, "INCOMPLETE_FEATURES", "Подготовленные признаки неполны.")
	}
	return out, nil
}
