#!/usr/bin/env bash
set -Eeuo pipefail

BASE_URL="${1:-http://localhost:8080}"
TIMEOUT_SECONDS="${READY_TIMEOUT_SECONDS:-120}"
INTERVAL_SECONDS="${READY_INTERVAL_SECONDS:-2}"

if ! command -v curl >/dev/null 2>&1; then
  echo 'curl is required for readiness checks.' >&2
  exit 1
fi

start=$SECONDS
while (( SECONDS - start < TIMEOUT_SECONDS )); do
  if curl -fsS --max-time 3 "$BASE_URL/health/ready" >/dev/null 2>&1; then
    echo "Ready: $BASE_URL/health/ready"
    exit 0
  fi
  sleep "$INTERVAL_SECONDS"
done

echo "Timed out after ${TIMEOUT_SECONDS}s waiting for $BASE_URL/health/ready" >&2
exit 1
