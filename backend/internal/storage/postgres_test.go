package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"
	"tramflow/internal/auth"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
	"tramflow/internal/inference"
)

func TestPostgresPublicationAndReproducibility(t *testing.T) {
	address := os.Getenv("TEST_DATABASE_URL")
	if address == "" {
		t.Skip("set TEST_DATABASE_URL to run real PostgreSQL/PostGIS integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := Open(ctx, address, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Close()
	dbName := fmt.Sprintf("tramflow_test_%d", time.Now().UnixNano())
	if _, err = admin.Pool.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + dbName
	s, err := Open(ctx, u.String(), 8)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		s.Pool.Close()
		_, _ = admin.Pool.Exec(context.Background(), "DROP DATABASE "+dbName+" WITH (FORCE)")
	}()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("migration not idempotent", err)
	}
	if err = s.CheckSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.ProvisionUser(ctx, "dispatcher", "test-password-long"); err != nil {
		t.Fatal(err)
	}
	replica, err := Open(ctx, u.String(), 4)
	if err != nil {
		t.Fatal(err)
	}
	defer replica.Pool.Close()
	authA, authB := auth.New(s), auth.New(replica)
	_, token, err := authA.Login(ctx, "dispatcher", "test-password-long")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = authB.Resolve(ctx, token); err != nil {
		t.Fatal("session unavailable on second replica", err)
	}
	if err = authB.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err = authA.Resolve(ctx, token); err == nil {
		t.Fatal("revoked session accepted on first replica")
	}
	raw, err := os.ReadFile("../../testdata/demo-bundle.json")
	if err != nil {
		t.Fatal(err)
	}
	var a d.Bundle
	_ = json.Unmarshal(raw, &a)
	origin := a.Snapshot.Provenance.Origin
	hs := []d.Hour{}
	for _, h := range a.Hours {
		if h.Time.Before(origin.Add(24 * time.Hour)) {
			hs = append(hs, h)
		}
	}
	a.Hours = hs
	for i := range a.Snapshot.Coverage {
		a.Snapshot.Coverage[i].Window.To = origin.Add(24 * time.Hour)
		a.Snapshot.Coverage[i].MaxLead = 24
	}
	c, err := contract.Load("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m := inference.New("../../artifacts")
	defer m.Close()
	if err = ValidateBundle(ctx, &a, c, m, true); err != nil {
		t.Fatal(err)
	}
	if err = s.Publish(ctx, a); err != nil {
		t.Fatal(err)
	}
	geo, err := s.Geometry(ctx, a.Snapshot.Provenance.Network, "demo-01")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Validate("RouteGeometry", geo); err != nil {
		t.Fatal(err)
	}
	desc := d.Descriptor{
		Kind: "forecast",
		Selection: d.Selection{
			SnapshotID:    a.Snapshot.Provenance.SnapshotID,
			RouteIDs:      []string{"demo-01"},
			View:          "day",
			Resolution:    "hour",
			SpatialDetail: "route",
			Window:        d.Window{From: origin, To: origin.Add(24 * time.Hour)},
		},
	}
	service := func() *engine.Service { return &engine.Service{Repo: s, Models: m, Cache: engine.NewCache(4096)} }
	first, err := service().Calculate(ctx, desc)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service().Calculate(ctx, desc)
	if err != nil {
		t.Fatal(err)
	}
	if string(encode(first)) != string(encode(second)) {
		t.Fatal("cold replica changed result")
	}
	clone := func(id string) d.Bundle {
		var b d.Bundle
		_ = json.Unmarshal(encode(a), &b)
		b.ExpectedPrevious = a.Snapshot.Provenance.SnapshotID
		b.Snapshot.Provenance.SnapshotID = id
		for _, r := range []*json.RawMessage{&b.Quality, &b.Sources} {
			var data map[string]any
			_ = json.Unmarshal(*r, &data)
			data["forecast_snapshot_id"] = id
			*r = encode(data)
		}
		return b
	}
	bad := clone("bad-candidate")
	bad.Model.Path = "missing.onnx"
	bad.Model.Release = "published"
	bad.Snapshot.Provenance.Mode = "replay"
	if ValidateBundle(ctx, &bad, c, m, false) == nil {
		t.Fatal("bad model accepted")
	}
	active, _, err := s.Active(ctx)
	if err != nil || active.Provenance.SnapshotID != a.Snapshot.Provenance.SnapshotID {
		t.Fatal("failed candidate damaged active")
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []string{"concurrent-b", "concurrent-c"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			results <- s.Publish(ctx, clone(id))
		}(id)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("CAS publication count", success)
	}
	if _, _, err = s.Snapshot(ctx, a.Snapshot.Provenance.SnapshotID); err != nil {
		t.Fatal("retained snapshot unavailable", err)
	}
	if _, err = s.Pool.Exec(ctx, "UPDATE forecast_snapshots SET deactivated_at=now()-interval '25 hours' WHERE id=$1", a.Snapshot.Provenance.SnapshotID); err != nil {
		t.Fatal(err)
	}
	if err = s.Expire(ctx); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Snapshot(ctx, a.Snapshot.Provenance.SnapshotID)
	var e *d.Error
	if !errors.As(err, &e) || e.Status != 410 {
		t.Fatal("expired snapshot must return 410", err)
	}
	_, _, err = s.Snapshot(ctx, "unknown")
	if !errors.As(err, &e) || e.Status != 404 {
		t.Fatal(err)
	}
	s.Pool.Close()
	_, _, err = s.Snapshot(ctx, "unknown")
	if !errors.As(err, &e) || e.Status != 503 {
		t.Fatal("database failure must not become missing snapshot", err)
	}
}
