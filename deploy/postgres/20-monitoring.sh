#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set=monitor_password="$MONITOR_DB_PASSWORD" <<'SQL'
CREATE ROLE tramflow_monitor LOGIN PASSWORD :'monitor_password';
GRANT pg_monitor TO tramflow_monitor;
GRANT CONNECT ON DATABASE tramflow TO tramflow_monitor;
SQL
