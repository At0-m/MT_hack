package storage

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/inference"
)

func TestV3CandidateAdmission(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/demo-bundle.json")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := contract.Load("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	manager := inference.New("../../artifacts")
	defer manager.Close()
	for _, kind := range []string{"valid", "huge reference", "tiny reference", "huge typical", "huge weather", "huge boardings", "missing weather values", "climatology provider", "missing geometry", "unsupported month"} {
		t.Run(kind, func(t *testing.T) {
			var bundle d.Bundle
			if err := json.Unmarshal(raw, &bundle); err != nil {
				t.Fatal(err)
			}
			hour := &bundle.Hours[0]
			large, tiny := 1e308, math.SmallestNonzeroFloat64
			switch kind {
			case "huge reference":
				hour.Reference = &large
			case "tiny reference":
				hour.Reference = &tiny
			case "huge typical":
				hour.Typical = &large
			case "huge weather":
				hour.Weather.Temperature = &large
			case "huge boardings":
				hour.SyntheticBoardings = &large
			case "missing weather values":
				hour.Weather.Mode = "missing"
			case "climatology provider":
				hour.Weather.Mode = "climatology"
			case "missing geometry":
				bundle.Routes[0].Geometry = nil
			case "unsupported month":
				bundle.Snapshot.Views = []string{"day", "week", "month"}
				bundle.Snapshot.Coverage[0].Resolutions = []string{"hour"}
			}
			err := ValidateBundle(context.Background(), &bundle, spec, manager, true)
			if (kind == "valid") != (err == nil) {
				t.Fatal(kind, err)
			}
		})
	}
}

func TestWeatherModeSemantics(t *testing.T) {
	value, code, now := 1.0, 0, time.Now()
	for _, point := range []d.WeatherPoint{
		{Mode: "missing"},
		{Mode: "climatology", Temperature: &value, Precipitation: &value, AvailableAt: &now},
		{Mode: "forecast", Code: &code, RunAt: &now},
	} {
		if err := validateWeatherMode(point); err != nil {
			t.Fatal(err)
		}
	}
}

func checkMigrationIntegrity(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	_, digest, err := migrationSQL(3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version=4"); err != nil {
		t.Fatal(err)
	}
	if store.CheckSchema(ctx) == nil || store.Migrate(ctx) == nil {
		t.Fatal("migration gap accepted")
	}
	if _, err = store.Pool.Exec(ctx, "INSERT INTO schema_migrations(version,checksum) VALUES(4,$1)", digest); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Pool.Exec(ctx, "UPDATE schema_migrations SET checksum='changed' WHERE version=4"); err != nil {
		t.Fatal(err)
	}
	if store.CheckSchema(ctx) == nil || store.Migrate(ctx) == nil {
		t.Fatal("checksum drift accepted")
	}
	if _, err = store.Pool.Exec(ctx, "UPDATE schema_migrations SET checksum=$1 WHERE version=4", digest); err != nil {
		t.Fatal(err)
	}
	// Simulate schema 5's legacy migration ledger, then verify explicit adoption.
	if _, err = store.Pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version=6; ALTER TABLE schema_migrations DROP COLUMN checksum"); err != nil {
		t.Fatal(err)
	}
	if store.CheckSchema(ctx) == nil {
		t.Fatal("legacy schema accepted by new API")
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal("legacy migration adoption", err)
	}
	if err = store.CheckSchema(ctx); err != nil {
		t.Fatal(err)
	}
}
