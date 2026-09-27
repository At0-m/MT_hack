#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$BACKEND_DIR"

REQUESTS="${REQUESTS:-5000}"
WARMUP_REQUESTS="${WARMUP_REQUESTS:-500}"
WARMUP_CONCURRENCY="${WARMUP_CONCURRENCY:-8}"
CONCURRENCIES="${CONCURRENCIES:-4 8 16 24 32}"

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

run_id="$(date -u +%Y%m%dT%H%M%SZ)"
result_dir="benchmarks/results/$run_id"
mkdir -p "$result_dir"

api_id="$(docker compose ps -q api)"
db_id="$(docker compose ps -q db)"
if [[ -z "$api_id" || -z "$db_id" ]]; then
  echo 'api and db must be running. Run ./scripts/dev.sh first.' >&2
  exit 1
fi

{
  echo "date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "commit=$(git rev-parse HEAD 2>/dev/null || echo unknown)"
  echo "uname=$(uname -a)"
  echo "host_cpus=$(nproc 2>/dev/null || echo unknown)"
  echo "docker=$(docker --version)"
  echo "compose=$(docker compose version)"
  docker inspect "$api_id" --format 'api_limits NanoCPUs={{.HostConfig.NanoCpus}} MemoryBytes={{.HostConfig.Memory}}'
  docker inspect "$db_id" --format 'db_limits NanoCPUs={{.HostConfig.NanoCpus}} MemoryBytes={{.HostConfig.Memory}}'
  echo "requests_per_step=$REQUESTS"
  echo "concurrencies=$CONCURRENCIES"
} > "$result_dir/environment.txt"

run_loadtest() {
  local n="$1"
  local c="$2"
  local output="$3"
  docker compose --profile tools run --rm --no-deps \
    --user "$(id -u):$(id -g)" \
    loadtest \
    -url http://api:8080 \
    -n "$n" \
    -c "$c" \
    -output "/results/$run_id/$output"
}

echo "Warm-up: n=$WARMUP_REQUESTS c=$WARMUP_CONCURRENCY"
run_loadtest "$WARMUP_REQUESTS" "$WARMUP_CONCURRENCY" warmup.json

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
sleep 2

for c in $CONCURRENCIES; do
  echo '=================================='
  echo "Benchmark concurrency=$c requests=$REQUESTS"
  echo '=================================='
  output="load-c${c}.json"
  if ! run_loadtest "$REQUESTS" "$c" "$output"; then
    if [[ -s "$result_dir/$output" ]]; then
      echo "Loadtest at c=$c reported HTTP errors; result preserved and benchmark continues." >&2
    else
      echo "Loadtest at c=$c failed before producing a report." >&2
      exit 1
    fi
  fi
done

cleanup
stats_pid=''

extract_number() {
  local file="$1"
  local key="$2"
  sed -nE 's/^[[:space:]]*"'"$key"'"[[:space:]]*:[[:space:]]*([^,]+),?[[:space:]]*$/\1/p' "$file" | tr -d '"'
}

summary_csv="$result_dir/summary.csv"
printf 'concurrency,requests,rps,p50_ms,p95_ms,p99_ms,errors,elapsed_seconds\n' > "$summary_csv"
summary_md="$result_dir/summary.md"
cat > "$summary_md" <<'EOF_MD'
# Benchmark summary

| Concurrency | Requests | RPS | p50 ms | p95 ms | p99 ms | Errors |
|---:|---:|---:|---:|---:|---:|---:|
EOF_MD

for c in $CONCURRENCIES; do
  file="$result_dir/load-c${c}.json"
  requests="$(extract_number "$file" requests)"
  rps="$(extract_number "$file" rps)"
  p50="$(extract_number "$file" p50_ms)"
  p95="$(extract_number "$file" p95_ms)"
  p99="$(extract_number "$file" p99_ms)"
  errors="$(extract_number "$file" errors)"
  elapsed="$(extract_number "$file" elapsed_seconds)"
  printf '%s,%s,%s,%s,%s,%s,%s,%s\n' "$c" "$requests" "$rps" "$p50" "$p95" "$p99" "$errors" "$elapsed" >> "$summary_csv"
  printf '| %s | %s | %s | %s | %s | %s | %s |\n' "$c" "$requests" "$rps" "$p50" "$p95" "$p99" "$errors" >> "$summary_md"
done

cat >> "$summary_md" <<EOF_MD

Test environment: [environment.txt](environment.txt)  
Raw container metrics: [system.csv](system.csv)
EOF_MD

echo
echo "Benchmark finished: $result_dir"
echo "Summary: $summary_md"
