#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${API_USER:?API_USER is required}"
: "${API_PASSWORD:?API_PASSWORD is required}"

TARGET_SNAPSHOT="${DEMO_SNAPSHOT_ID:-stress-october-2026-v1}"

echo "[seed] waiting for PostgreSQL..."

attempt=1
max_attempts=30

while ! pg_isready \
    -h db \
    -p 5432 \
    -U tramflow_publisher \
    -d tramflow >/dev/null 2>&1
do
    if [ "$attempt" -ge "$max_attempts" ]; then
        echo "[seed] PostgreSQL did not become ready"
        exit 1
    fi

    echo "[seed] PostgreSQL not ready; attempt ${attempt}/${max_attempts}"
    attempt=$((attempt + 1))
    sleep 2
done

echo "[seed] PostgreSQL ready"

echo "[seed] applying migrations and provisioning demo user..."

/app/publisher \
    --migrate \
    --create-user "${API_USER}"

echo "[seed] checking active snapshot..."

ACTIVE_SNAPSHOT="$(
    psql "${DATABASE_URL}" \
        -X \
        -A \
        -t \
        -v ON_ERROR_STOP=1 \
        -c 'SELECT snapshot_id FROM active_forecast_snapshot WHERE slot=1' \
        | tr -d '[:space:]'
)"

if [ -z "${ACTIVE_SNAPSHOT}" ]; then
    echo "[seed] no active snapshot; publishing ${TARGET_SNAPSHOT}..."

    /app/publisher \
        --bundle /data/demo-bundle.json \
        --allow-synthetic

    echo "[seed] snapshot ${TARGET_SNAPSHOT} published"

elif [ "${ACTIVE_SNAPSHOT}" = "${TARGET_SNAPSHOT}" ]; then
    echo "[seed] snapshot ${TARGET_SNAPSHOT} is already active; skipping publication"

else
    echo "[seed] ERROR: another snapshot is already active"
    echo "[seed] expected: ${TARGET_SNAPSHOT}"
    echo "[seed] active:   ${ACTIVE_SNAPSHOT}"
    echo "[seed] refusing to overwrite it automatically"
    exit 1
fi

echo "[seed] completed successfully"
