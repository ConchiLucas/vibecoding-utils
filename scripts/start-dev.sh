#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BACKEND_PORT_START="${BACKEND_PORT:-8009}"
FRONTEND_PORT_START="${FRONTEND_PORT:-5176}"
CONFIG_PATH="${CONFIG_PATH:-/private/tmp/easy-deploy-dev-config.yaml}"
GOCACHE_DIR="${GOCACHE:-/private/tmp/easy-deploy-go-build-cache}"

port_in_use() {
  if ! command -v lsof >/dev/null 2>&1; then
    return 1
  fi
  lsof -iTCP:"$1" -sTCP:LISTEN -n -P >/dev/null 2>&1
}

require_free_port() {
  local port="$1"
  local label="$2"
  if port_in_use "$port"; then
    echo "${label} port ${port} is already in use. Stop the existing dev service before restarting." >&2
    exit 1
  fi
}

BACKEND_PORT="$BACKEND_PORT_START"
FRONTEND_PORT="$FRONTEND_PORT_START"

require_free_port "$BACKEND_PORT" "Backend"
require_free_port "$FRONTEND_PORT" "Frontend"

mkdir -p "$(dirname "$CONFIG_PATH")" "$GOCACHE_DIR"

awk -v port="$BACKEND_PORT" '
  /^system:/ { in_system = 1 }
  /^[^[:space:]].*:/ && $0 !~ /^system:/ { in_system = 0 }
  in_system && /^[[:space:]]+addr:/ {
    print "    addr: " port
    next
  }
  { print }
' "$ROOT_DIR/server/config.template.yaml" > "$CONFIG_PATH"

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT INT TERM

echo "Backend:  http://localhost:${BACKEND_PORT}"
echo "Frontend: http://localhost:${FRONTEND_PORT}"
echo "Config:   ${CONFIG_PATH}"

if [ ! -d "$ROOT_DIR/web-react/node_modules" ]; then
  echo "Installing frontend dependencies..."
  (cd "$ROOT_DIR/web-react" && npm ci)
fi

(
  cd "$ROOT_DIR/server"
  env GOCACHE="$GOCACHE_DIR" go run ./cmd/http -c "$CONFIG_PATH"
) &
BACKEND_PID=$!

(
  cd "$ROOT_DIR/web-react"
  env VITE_BASE_API="http://localhost:${BACKEND_PORT}" npm run dev -- --host 0.0.0.0 --port "$FRONTEND_PORT" --strictPort
) &
FRONTEND_PID=$!

PIDS=("$BACKEND_PID" "$FRONTEND_PID")

while true; do
  for pid in "${PIDS[@]}"; do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      wait "$pid"
      exit $?
    fi
  done
  sleep 1
done
