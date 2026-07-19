#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_BIN="${DOCKER_BIN:-docker}"

DOCKER_BIN="$DOCKER_BIN" "$ROOT_DIR/scripts/ensure-docker-network.sh"
exec "$DOCKER_BIN" compose "$@"
