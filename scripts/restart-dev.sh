#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT_MIN="${PORT_MIN:-20000}"
PORT_MAX="${PORT_MAX:-30000}"
PID_FILE="${PID_FILE:-/private/tmp/vibecoding-utils-dev.pids}"
LOG_DIR="${LOG_DIR:-/private/tmp/vibecoding-utils-dev-logs}"
CONFIG_PATH="${CONFIG_PATH:-/private/tmp/vibecoding-utils-dev-config.yaml}"
GOCACHE_DIR="${GOCACHE:-/private/tmp/easy-deploy-go-build-cache}"

usage() {
  echo "Usage: $0 [restart|start|stop]"
  echo
  echo "Environment:"
  echo "  PORT_MIN=20000"
  echo "  PORT_MAX=30000"
  echo "  BACKEND_PORT=<fixed backend port>"
  echo "  FRONTEND_PORT=<fixed frontend port>"
  echo "  PID_FILE=/private/tmp/vibecoding-utils-dev.pids"
  echo "  CONFIG_PATH=/private/tmp/vibecoding-utils-dev-config.yaml"
}

is_positive_integer() {
  case "$1" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

validate_port_range() {
  if ! is_positive_integer "$PORT_MIN" || ! is_positive_integer "$PORT_MAX"; then
    echo "PORT_MIN and PORT_MAX must be positive integers" >&2
    exit 1
  fi
  if [ "$PORT_MIN" -lt 1 ] || [ "$PORT_MAX" -gt 65535 ] || [ "$PORT_MIN" -gt "$PORT_MAX" ]; then
    echo "Invalid port range: ${PORT_MIN}-${PORT_MAX}" >&2
    exit 1
  fi
}

port_in_use() {
  if ! command -v lsof >/dev/null 2>&1; then
    return 1
  fi
  lsof -iTCP:"$1" -sTCP:LISTEN -n -P >/dev/null 2>&1
}

port_is_avoided() {
  local port="$1"
  shift || true
  for avoided in "$@"; do
    if [ "$port" = "$avoided" ]; then
      return 0
    fi
  done
  return 1
}

pick_random_port() {
  local candidate
  local span=$((PORT_MAX - PORT_MIN + 1))

  for _ in $(seq 1 200); do
    candidate=$((PORT_MIN + RANDOM % span))
    if ! port_in_use "$candidate" && ! port_is_avoided "$candidate" "$@"; then
      echo "$candidate"
      return 0
    fi
  done

  candidate="$PORT_MIN"
  while [ "$candidate" -le "$PORT_MAX" ]; do
    if ! port_in_use "$candidate" && ! port_is_avoided "$candidate" "$@"; then
      echo "$candidate"
      return 0
    fi
    candidate=$((candidate + 1))
  done

  echo "No free port in ${PORT_MIN}-${PORT_MAX}" >&2
  exit 1
}

wait_for_exit() {
  local pid="$1"
  local label="$2"

  for _ in $(seq 1 30); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done

  echo "${label} pid ${pid} did not exit after TERM, sending KILL"
  kill -KILL "$pid" >/dev/null 2>&1 || true
}

stop_recorded_processes() {
  if [ ! -f "$PID_FILE" ]; then
    return 0
  fi

  echo "Stopping recorded dev processes from ${PID_FILE}"
  while read -r label pid; do
    if [ -z "${pid:-}" ]; then
      continue
    fi
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "Stopping ${label} pid ${pid}"
      kill "$pid" >/dev/null 2>&1 || true
      wait_for_exit "$pid" "$label"
    fi
  done < "$PID_FILE"

  rm -f "$PID_FILE"
}

write_backend_config() {
  mkdir -p "$(dirname "$CONFIG_PATH")" "$GOCACHE_DIR" "$LOG_DIR"

  awk -v port="$BACKEND_PORT" '
    /^system:/ { in_system = 1 }
    /^[^[:space:]].*:/ && $0 !~ /^system:/ { in_system = 0 }
    in_system && /^[[:space:]]+addr:/ {
      print "    addr: " port
      next
    }
    { print }
  ' "$ROOT_DIR/server/config.template.yaml" > "$CONFIG_PATH"
}

wait_for_http() {
  local url="$1"
  local label="$2"

  if ! command -v curl >/dev/null 2>&1; then
    return 0
  fi

  for _ in $(seq 1 80); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "${label} is ready: ${url}"
      return 0
    fi
    sleep 0.25
  done

  echo "${label} did not become ready in time: ${url}" >&2
  return 1
}

start_services() {
  validate_port_range

  BACKEND_PORT="${BACKEND_PORT:-$(pick_random_port)}"
  FRONTEND_PORT="${FRONTEND_PORT:-$(pick_random_port "$BACKEND_PORT")}"

  write_backend_config

  if [ ! -d "$ROOT_DIR/web-react/node_modules" ]; then
    echo "Installing frontend dependencies..."
    (cd "$ROOT_DIR/web-react" && npm ci)
  fi

  echo "Backend:  http://localhost:${BACKEND_PORT}"
  echo "Frontend: http://localhost:${FRONTEND_PORT}"
  echo "Config:   ${CONFIG_PATH}"
  echo "Logs:     ${LOG_DIR}"

  (
    cd "$ROOT_DIR/server"
    env GOCACHE="$GOCACHE_DIR" go run ./cmd/http -c "$CONFIG_PATH"
  ) >"${LOG_DIR}/backend.log" 2>&1 &
  BACKEND_PID=$!

  (
    cd "$ROOT_DIR/web-react"
    env VITE_BASE_API="http://localhost:${BACKEND_PORT}" npm run dev -- --host 0.0.0.0 --port "$FRONTEND_PORT" --strictPort
  ) >"${LOG_DIR}/frontend.log" 2>&1 &
  FRONTEND_PID=$!

  {
    echo "backend ${BACKEND_PID}"
    echo "frontend ${FRONTEND_PID}"
  } > "$PID_FILE"

  cleanup() {
    stop_recorded_processes >/dev/null 2>&1 || true
  }
  trap cleanup EXIT INT TERM

  wait_for_http "http://127.0.0.1:${BACKEND_PORT}/health" "Backend" || true
  wait_for_http "http://127.0.0.1:${FRONTEND_PORT}/" "Frontend" || true

  echo
  echo "Dev services are running."
  echo "Open: http://localhost:${FRONTEND_PORT}/projects"
  echo "Tail logs:"
  echo "  tail -f ${LOG_DIR}/backend.log"
  echo "  tail -f ${LOG_DIR}/frontend.log"
  echo

  while true; do
    for pid in "$BACKEND_PID" "$FRONTEND_PID"; do
      if ! kill -0 "$pid" >/dev/null 2>&1; then
        echo "A dev process exited. Recent backend log:"
        tail -n 40 "${LOG_DIR}/backend.log" || true
        echo "Recent frontend log:"
        tail -n 40 "${LOG_DIR}/frontend.log" || true
        wait "$pid"
        exit $?
      fi
    done
    sleep 1
  done
}

ACTION="${1:-restart}"
case "$ACTION" in
  restart)
    stop_recorded_processes
    start_services
    ;;
  start)
    start_services
    ;;
  stop)
    stop_recorded_processes
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
