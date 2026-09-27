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
	checkMigrationIntegrity(t, s, ctx)
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
	// This test intentionally keeps one day: it must advertise only that view.
	a.Snapshot.Views = []string{"day"}
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
	beforeWeather, err := s.WeatherMetadata(ctx, a.Snapshot.Provenance.Weather)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"shifted", "no-day", "no-hour"} {
		candidate := clone("bad-" + kind)
		switch kind {
		case "shifted":
			for i := range candidate.Snapshot.Coverage {
				candidate.Snapshot.Coverage[i].Window.From = origin.Add(12 * time.Hour)
				candidate.Snapshot.Coverage[i].Window.To = origin.Add(36 * time.Hour)
				candidate.Snapshot.Coverage[i].MaxLead = 36
			}
		case "no-day":
			candidate.Snapshot.Views = []string{"week"}
		case "no-hour":
			for i := range candidate.Snapshot.Coverage {
				candidate.Snapshot.Coverage[i].Resolutions = []string{"day"}
			}
		}
		if err := s.Publish(ctx, candidate); err == nil {
			t.Fatal("unusable candidate activated", kind)
		}
		active, _, err := s.Active(ctx)
		if err != nil || active.Provenance.SnapshotID != a.Snapshot.Provenance.SnapshotID {
			t.Fatal("active changed after rejected candidate")
		}
	}
	// Enforce pattern ownership even for writes outside the normal bundle importer.
	_, err = s.Pool.Exec(ctx, `INSERT INTO route_stops(network_version,route_id,pattern_id,route_stop_id,stop_id,sequence)
 VALUES($1,'demo-02','demo-01-out','invalid-cross-route','demo-01-stop-1',99)`, a.Snapshot.Provenance.Network)
	if err == nil {
		t.Fatal("cross-route pattern accepted")
	}
	_, err = s.Pool.Exec(ctx, `SELECT 1`) // Confirm failed single statement did not break the connection.
	if err != nil {
		t.Fatal(err)
	}
	schemaMutation := clone("bad-schema-mutation")
	schemaMutation.Model.Version = "different-model"
	schemaMutation.Snapshot.Provenance.Model = schemaMutation.Model.Version
	schemaMutation.Model.SchemaSHA256 = "different-schema-checksum"
	if err := s.Publish(ctx, schemaMutation); err == nil {
		t.Fatal("same feature schema ID accepted another artifact")
	}
	// The metadata path must never expand full feature blobs, even on a cold miss.
	profiles, err := s.Hours(ctx, a.Snapshot, engine.SelectionInput{Routes: []string{"demo-01"}, Window: desc.Selection.Window})
	if err != nil || len(profiles) != 24 {
		t.Fatal(err)
	}
	if len(profiles[0].Features) != 0 {
		t.Fatal("full feature map fetched on metadata path")
	}
	loaded, err := s.Features(ctx, a.Snapshot, profiles[:2])
	if err != nil || len(loaded[0].Features) != 2 || len(profiles[0].Features) != 0 {
		t.Fatal("feature batch aliases lightweight profiles", err)
	}
	// Account throttling is shared between database pools, independent of client IP.
	for i := 0; i < 120; i++ {
		store := s
		if i%2 == 1 {
			store = replica
		}
		allowed, err := store.AllowLogin(ctx, auth.Hash("shared-account"))
		if err != nil || !allowed {
			t.Fatal(i, err)
		}
	}
	if allowed, err := replica.AllowLogin(ctx, auth.Hash("shared-account")); err != nil || allowed {
		t.Fatal("replica bypasses account limiter", err)
	}
	userID := auth.Hash("dispatcher")[:32]
	_, verifiedHash, err := s.User(ctx, "dispatcher")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pool.Exec(ctx, "INSERT INTO user_sessions(token_hash,user_id,expires_at) VALUES('expired-review',$1,now()-interval '1 hour')", userID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 22; i++ {
		if err := s.CreateSession(ctx, fmt.Sprintf("review-session-%d", i), userID, time.Now().Add(time.Hour), verifiedHash); err != nil {
			t.Fatal(err)
		}
	}
	var sessionCount int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE user_id=$1", userID).Scan(&sessionCount); err != nil || sessionCount != 20 {
		t.Fatal("session cleanup/cap", sessionCount, err)
	}
	if err := s.ProvisionUser(ctx, "dispatcher", "new-password-long"); err != nil {
		t.Fatal(err)
	}
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE user_id=$1", userID).Scan(&sessionCount); err != nil || sessionCount != 0 {
		t.Fatal("password rotation did not revoke sessions", sessionCount, err)
	}
	if err := s.CreateSession(ctx, "stale-password-session", userID, time.Now().Add(time.Hour), verifiedHash); err == nil {
		t.Fatal("session created from password verified before rotation")
	}
	for range 11 {
		if _, _, err := authA.Login(ctx, "dispatcher", "invalid-password"); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if _, _, err := authB.Login(ctx, "dispatcher", "new-password-long"); err != nil {
		t.Fatal("targeted account lockout persists across replicas", err)
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
			candidate := clone(id)
			candidate.Snapshot.Created = candidate.Snapshot.Created.Add(2 * time.Hour)
			results <- s.Publish(ctx, candidate)
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
	afterWeather, err := s.WeatherMetadata(ctx, a.Snapshot.Provenance.Weather)
	if err != nil || string(encode(beforeWeather)) != string(encode(afterWeather)) {
		t.Fatal("forecast publication changed weather metadata")
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
	if _, err := s.Pool.Exec(ctx, `INSERT INTO supply_profiles(version,route_id,target_hour,fleet,source,is_proxy)
 VALUES('orphan-review','demo-01',$1,1,'manual_plan',false)`, origin); err != nil {
		t.Fatal(err)
	}
	if err := s.CollectData(ctx); err != nil {
		t.Fatal(err)
	}
	var orphanCount int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM supply_profiles WHERE version='orphan-review'").Scan(&orphanCount); err != nil || orphanCount != 0 {
		t.Fatal("orphan profiles not collected", err)
	}
	if profiles, err := s.Hours(ctx, a.Snapshot, engine.SelectionInput{Routes: []string{"demo-01"}, Window: desc.Selection.Window}); err != nil || len(profiles) != 24 {
		t.Fatal("GC removed versions shared by live snapshot", err)
	}
	s.Pool.Close()
	_, _, err = s.Snapshot(ctx, "unknown")
	if !errors.As(err, &e) || e.Status != 503 {
		t.Fatal("database failure must not become missing snapshot", err)
	}
}
