#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$BACKEND_DIR"

OUTPUT="${1:-benchmarks/results/system.csv}"
INTERVAL_SECONDS="${STATS_INTERVAL_SECONDS:-1}"
mkdir -p "$(dirname -- "$OUTPUT")"

api_id="$(docker compose ps -q api)"
db_id="$(docker compose ps -q db)"
if [[ -z "$api_id" || -z "$db_id" ]]; then
  echo 'api and db containers must be running before stats collection.' >&2
  exit 1
fi

printf 'timestamp,container,cpu_percent,memory_usage,memory_percent,pids\n' > "$OUTPUT"

while :; do
  timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  while IFS= read -r row; do
    [[ -n "$row" ]] && printf '%s,%s\n' "$timestamp" "$row" >> "$OUTPUT"
  done < <(docker stats --no-stream \
    --format '{{.Name}},{{.CPUPerc}},{{.MemUsage}},{{.MemPerc}},{{.PIDs}}' \
    "$api_id" "$db_id")
  sleep "$INTERVAL_SECONDS"
done
