CREATE TABLE weather_snapshots(
 version text PRIMARY KEY,
 metadata jsonb NOT NULL
);
ALTER TABLE route_patterns ADD CONSTRAINT route_patterns_route_identity
 UNIQUE(network_version,route_id,pattern_id);
ALTER TABLE route_stops ADD CONSTRAINT route_stop_pattern_belongs_to_route
 FOREIGN KEY(network_version,route_id,pattern_id)
 REFERENCES route_patterns(network_version,route_id,pattern_id);
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='tramflow_api') THEN
  GRANT SELECT ON weather_snapshots TO tramflow_api;
 END IF;
END $$;
INSERT INTO schema_migrations(version) VALUES(3);
