package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
	"tramflow/internal/telemetry"
)

func validateWeatherMetadata(b d.Bundle) error {
	w := b.WeatherMetadata
	if w.ID != b.Snapshot.Provenance.Weather || !engine.IDPattern.MatchString(w.SourceID) || w.RetrievedAt.IsZero() || w.RetrievedAt.After(b.Snapshot.Created) {
		return fmt.Errorf("invalid weather snapshot metadata")
	}
	if w.Provider != "open-meteo" && w.Provider != "synthetic_mock" {
		return fmt.Errorf("unsupported weather provider")
	}
	if (w.Provider == "synthetic_mock") != (b.Snapshot.Provenance.Mode == "synthetic_mock") {
		return fmt.Errorf("weather provider/runtime mismatch")
	}
	for _, c := range b.Snapshot.Coverage {
		if !engine.IDPattern.MatchString(w.Locations[c.RouteID]) {
			return fmt.Errorf("weather location missing for %s", c.RouteID)
		}
	}
	var report struct {
		Sources []struct {
			ID        string    `json:"source_id"`
			Category  string    `json:"category"`
			Version   string    `json:"version"`
			Retrieved time.Time `json:"retrieved_at"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(b.Sources, &report); err != nil {
		return err
	}
	seen, found := map[string]bool{}, false
	for _, source := range report.Sources {
		if seen[source.ID] {
			return fmt.Errorf("duplicate source ID")
		}
		seen[source.ID] = true
		if source.ID == w.SourceID {
			if source.Category != "weather" || source.Version != w.ID || !source.Retrieved.Equal(w.RetrievedAt) {
				return fmt.Errorf("weather source metadata mismatch")
			}
			found = true
		}
	}
	if !found {
		return fmt.Errorf("weather source not found in SourceList")
	}
	return nil
}

func (s *Store) WeatherMetadata(ctx context.Context, version string) (d.WeatherSnapshot, error) {
	done := telemetry.Timer("storage_operation_duration_seconds", telemetry.Labels{"operation": "weather"})
	defer done()
	var raw []byte
	var metadata d.WeatherSnapshot
	err := s.Pool.QueryRow(ctx, "SELECT metadata FROM weather_snapshots WHERE version=$1", version).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return metadata, d.Fail(503, "WEATHER_METADATA_UNAVAILABLE", "Метаданные погодного снимка недоступны.")
	}
	if err != nil {
		return metadata, dependency(err)
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return metadata, dependency(err)
	}
	return metadata, nil
}
