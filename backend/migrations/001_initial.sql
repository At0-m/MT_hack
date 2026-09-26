CREATE EXTENSION IF NOT EXISTS postgis;
CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE models(version text PRIMARY KEY, metadata jsonb NOT NULL);
CREATE TABLE data_versions(kind text NOT NULL, version text NOT NULL, digest text NOT NULL, PRIMARY KEY(kind,version));
CREATE TABLE forecast_snapshots(
 id text PRIMARY KEY, metadata jsonb NOT NULL, model_version text NOT NULL REFERENCES models(version),
 published_at timestamptz NOT NULL DEFAULT now(), deactivated_at timestamptz, deleted_at timestamptz
);
CREATE TABLE active_forecast_snapshot(slot integer PRIMARY KEY CHECK(slot=1), snapshot_id text NOT NULL REFERENCES forecast_snapshots(id));
CREATE TABLE routes(network_version text NOT NULL, route_id text NOT NULL, detail jsonb NOT NULL,
 PRIMARY KEY(network_version,route_id));
CREATE TABLE route_patterns(network_version text NOT NULL, route_id text NOT NULL, pattern_id text NOT NULL,
 geometry geometry(Geometry,4326) NOT NULL CHECK(GeometryType(geometry) IN ('LINESTRING','MULTILINESTRING')),
 PRIMARY KEY(network_version,pattern_id), FOREIGN KEY(network_version,route_id) REFERENCES routes(network_version,route_id));
CREATE INDEX route_patterns_spatial ON route_patterns USING gist(geometry);
CREATE TABLE stops(network_version text NOT NULL, stop_id text NOT NULL, name text NOT NULL, location geometry(Point,4326) NOT NULL,
 PRIMARY KEY(network_version,stop_id));
CREATE INDEX stops_spatial ON stops USING gist(location);
CREATE TABLE route_stops(network_version text NOT NULL, route_id text NOT NULL, pattern_id text NOT NULL, route_stop_id text NOT NULL,
 stop_id text NOT NULL, sequence integer NOT NULL CHECK(sequence>0),
 PRIMARY KEY(network_version,route_stop_id), UNIQUE(network_version,pattern_id,sequence),
 FOREIGN KEY(network_version,pattern_id) REFERENCES route_patterns(network_version,pattern_id),
 FOREIGN KEY(network_version,stop_id) REFERENCES stops(network_version,stop_id),
 FOREIGN KEY(network_version,route_id) REFERENCES routes(network_version,route_id));
CREATE TABLE prepared_features(history_version text NOT NULL, schema_version text NOT NULL, origin timestamptz NOT NULL,
 route_id text NOT NULL,target_hour timestamptz NOT NULL, available_at timestamptz NOT NULL, features jsonb NOT NULL,
 synthetic_boardings double precision CHECK(synthetic_boardings>=0),
 PRIMARY KEY(history_version,schema_version,origin,route_id,target_hour));
CREATE TABLE supply_profiles(version text NOT NULL,route_id text NOT NULL,target_hour timestamptz NOT NULL,
 fleet double precision CHECK(fleet BETWEEN 0 AND 200), source text NOT NULL, is_proxy boolean NOT NULL,
 PRIMARY KEY(version,route_id,target_hour));
CREATE TABLE reference_profiles(version text NOT NULL,route_id text NOT NULL,target_hour timestamptz NOT NULL,
 reference double precision CHECK(reference>=0), typical double precision CHECK(typical>=0),
 PRIMARY KEY(version,route_id,target_hour));
CREATE TABLE weather_points(version text NOT NULL,route_id text NOT NULL,target_hour timestamptz NOT NULL, point jsonb NOT NULL,
 PRIMARY KEY(version,route_id,target_hour));
CREATE TABLE snapshot_reports(snapshot_id text PRIMARY KEY REFERENCES forecast_snapshots(id), quality jsonb NOT NULL, sources jsonb NOT NULL);
INSERT INTO schema_migrations(version) VALUES (1);
