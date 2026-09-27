#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set=api_password="$API_DB_PASSWORD" <<'SQL'
CREATE ROLE tramflow_api LOGIN PASSWORD :'api_password';
GRANT CONNECT ON DATABASE tramflow TO tramflow_api;
GRANT USAGE ON SCHEMA public TO tramflow_api;
ALTER DEFAULT PRIVILEGES FOR ROLE tramflow_publisher IN SCHEMA public GRANT SELECT ON TABLES TO tramflow_api;
SQL
