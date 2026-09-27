#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
ENV_PATH="$BACKEND_DIR/.env"

if [[ -f "$ENV_PATH" ]]; then
  echo '.env already exists; preserved.'
  exit 0
fi

new_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 24
    return
  fi
  LC_ALL=C od -An -N24 -tx1 /dev/urandom | tr -d ' \n'
}

umask 077
cat > "$ENV_PATH" <<EOF_ENV
DB_PASSWORD=$(new_secret)
API_DB_PASSWORD=$(new_secret)
API_USER=dispatcher
API_PASSWORD=$(new_secret)
CORS_ORIGIN=http://localhost:5173
TRUSTED_PROXY_CIDRS=
EOF_ENV
chmod 600 "$ENV_PATH"

echo 'Created .env with random local credentials. Do not commit it.'
