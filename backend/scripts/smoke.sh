#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$BACKEND_DIR"

if [[ ! -f .env ]]; then
  "$SCRIPT_DIR/init-env.sh"
fi
set -a
# shellcheck disable=SC1091
source ./.env
set +a

"$SCRIPT_DIR/wait-ready.sh" http://localhost:8080
curl -fsS http://localhost:8080/health/live >/dev/null
curl -fsS http://localhost:8080/health/ready >/dev/null

run_id="smoke-$(date -u +%Y%m%dT%H%M%SZ)"
result_dir="benchmarks/results/$run_id"
mkdir -p "$result_dir"

docker compose --profile tools build loadtest >/dev/null
docker compose --profile tools run --rm --no-deps \
  --user "$(id -u):$(id -g)" \
  loadtest \
  -url http://api:8080 \
  -n "${SMOKE_REQUESTS:-10}" \
  -c "${SMOKE_CONCURRENCY:-1}" \
  -output "/results/$run_id/summary.json"

echo "Smoke test passed: $result_dir/summary.json"
