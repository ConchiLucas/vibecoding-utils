#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
NETWORK_NAME="vibedeploy-shared"
NETWORK_LABEL="com.vibedeploy.managed=true"

if "$DOCKER_BIN" network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "Docker network ${NETWORK_NAME} already exists."
  exit 0
fi

if "$DOCKER_BIN" network create \
  --driver bridge \
  --label "$NETWORK_LABEL" \
  "$NETWORK_NAME" >/dev/null; then
  echo "Docker network ${NETWORK_NAME} created."
  exit 0
fi

if "$DOCKER_BIN" network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "Docker network ${NETWORK_NAME} was created concurrently."
  exit 0
fi

echo "Unable to create Docker network ${NETWORK_NAME}." >&2
exit 1
