CREATE TABLE app_users(user_id text PRIMARY KEY,username text UNIQUE NOT NULL,display_name text NOT NULL,password_hash text NOT NULL,disabled boolean NOT NULL DEFAULT false);
CREATE TABLE user_sessions(token_hash text PRIMARY KEY,user_id text NOT NULL REFERENCES app_users(user_id),expires_at timestamptz NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX user_sessions_expiry ON user_sessions(expires_at);
ALTER TABLE route_stops ADD COLUMN anchor geometry(Point,4326);
ALTER TABLE route_stops ADD COLUMN anchor_kind text NOT NULL DEFAULT 'stop_position' CHECK(anchor_kind IN ('stop_position','approach_20m_before_stop'));
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='tramflow_api') THEN
   GRANT SELECT ON app_users TO tramflow_api;
   GRANT SELECT, INSERT, DELETE ON user_sessions TO tramflow_api;
 END IF;
END $$;
INSERT INTO schema_migrations(version) VALUES(2);
