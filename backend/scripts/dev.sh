#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$BACKEND_DIR"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command not found: $1" >&2
    exit 1
  fi
}

need docker
need curl
if ! docker compose version >/dev/null 2>&1; then
  echo 'Docker Compose v2 is required: docker compose ...' >&2
  exit 1
fi

"$SCRIPT_DIR/init-env.sh"
set -a
# shellcheck disable=SC1091
source ./.env
set +a

mkdir -p artifacts testdata benchmarks/results

if [[ ! -f testdata/demo-bundle.json ]]; then
  echo 'Generating synthetic demo bundle and ONNX artifact inside Docker...'
  docker compose --profile tools build fixtures
  docker compose --profile tools run --rm --no-deps \
    --user "$(id -u):$(id -g)" \
    fixtures --onnx
fi

echo 'Starting PostgreSQL/PostGIS...'
docker compose up -d db

printf 'Waiting for PostgreSQL'
for _ in $(seq 1 60); do
  if docker compose exec -T db pg_isready -U tramflow_publisher -d tramflow >/dev/null 2>&1; then
    echo
    break
  fi
  printf '.'
  sleep 2
done
if ! docker compose exec -T db pg_isready -U tramflow_publisher -d tramflow >/dev/null 2>&1; then
  echo >&2
  echo 'PostgreSQL did not become ready.' >&2
  exit 1
fi

echo 'Applying migrations and provisioning API user...'
docker compose --profile tools run --rm publisher --migrate --create-user "$API_USER"

active="$(docker compose exec -T db psql -U tramflow_publisher -d tramflow -Atc \
  'SELECT snapshot_id FROM active_forecast_snapshot WHERE slot=1' | tr -d '[:space:]')"

if [[ -z "$active" ]]; then
  echo 'Publishing synthetic demo snapshot...'
  docker compose --profile tools run --rm publisher \
    --bundle /data/demo-bundle.json --allow-synthetic
else
  echo "Active snapshot already exists: $active"
fi

echo 'Building and starting API...'
docker compose up -d --build api
"$SCRIPT_DIR/wait-ready.sh" http://localhost:8080

echo
printf 'API: http://localhost:8080\n'
printf 'User: %s\n' "$API_USER"
printf 'Password is stored in %s/.env\n' "$BACKEND_DIR"
