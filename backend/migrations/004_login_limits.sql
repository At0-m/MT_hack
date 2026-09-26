CREATE TABLE login_limits(
 account_hash text PRIMARY KEY,
 window_start timestamptz NOT NULL,
 attempts integer NOT NULL
);
CREATE INDEX login_limits_expiry ON login_limits(window_start);
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='tramflow_api') THEN
  GRANT SELECT,INSERT,UPDATE,DELETE ON login_limits TO tramflow_api;
 END IF;
END $$;
INSERT INTO schema_migrations(version) VALUES(4);
