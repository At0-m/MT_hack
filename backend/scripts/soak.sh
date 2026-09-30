#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$BACKEND_DIR"

DURATION_MINUTES="${DURATION_MINUTES:-30}"
SOAK_CONCURRENCY="${SOAK_CONCURRENCY:-16}"
BATCH_REQUESTS="${BATCH_REQUESTS:-5000}"

if [[ ! -f .env ]]; then
  echo '.env not found. Run ./scripts/dev.sh first.' >&2
  exit 1
fi
set -a
# shellcheck disable=SC1091
source ./.env
set +a

"$SCRIPT_DIR/wait-ready.sh" http://localhost:8080
docker compose --profile tools build loadtest >/dev/null

run_id="soak-$(date -u +%Y%m%dT%H%M%SZ)"
result_dir="benchmarks/results/$run_id"
mkdir -p "$result_dir"

stats_pid=''
cleanup() {
  if [[ -n "$stats_pid" ]] && kill -0 "$stats_pid" >/dev/null 2>&1; then
    kill "$stats_pid" >/dev/null 2>&1 || true
    wait "$stats_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM
"$SCRIPT_DIR/collect-stats.sh" "$result_dir/system.csv" &
stats_pid=$!

start_epoch="$(date +%s)"
end_epoch=$(( start_epoch + DURATION_MINUTES * 60 ))
iteration=0
failed=0

while (( $(date +%s) < end_epoch )); do
  iteration=$((iteration + 1))
  output="batch-$(printf '%04d' "$iteration").json"
  echo "Soak batch $iteration: c=$SOAK_CONCURRENCY n=$BATCH_REQUESTS"
  if ! docker compose --profile tools run --rm --no-deps \
    --user "$(id -u):$(id -g)" \
    loadtest \
    -url http://api:8080 \
    -n "$BATCH_REQUESTS" \
    -c "$SOAK_CONCURRENCY" \
    -output "/results/$run_id/$output"; then
    failed=1
    echo "Soak loadtest failed at batch $iteration; stopping." >&2
    break
  fi
done

cleanup
stats_pid=''

end_actual="$(date +%s)"
{
  echo "started_utc=$(date -u -d "@$start_epoch" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "$start_epoch")"
  echo "duration_requested_minutes=$DURATION_MINUTES"
  echo "duration_actual_seconds=$((end_actual - start_epoch))"
  echo "concurrency=$SOAK_CONCURRENCY"
  echo "batch_requests=$BATCH_REQUESTS"
  echo "completed_batches=$iteration"
  echo "failed=$failed"
} > "$result_dir/soak-summary.txt"

echo "Soak artifacts: $result_dir"
if (( failed != 0 )); then
  exit 1
fi
