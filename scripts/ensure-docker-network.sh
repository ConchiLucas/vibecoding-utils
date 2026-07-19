#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
NETWORK_NAME="vibedeploy-shared"
NETWORK_LABEL="com.vibedeploy.managed=true"

inspect_driver() {
  "$DOCKER_BIN" network inspect --format '{{.Driver}}' "$NETWORK_NAME" 2>/dev/null
}

validate_driver() {
  local driver="$1"
  if [ "$driver" != "bridge" ]; then
    echo "Docker network ${NETWORK_NAME} uses driver ${driver:-unknown}; expected bridge." >&2
    return 1
  fi
}

if EXISTING_DRIVER="$(inspect_driver)"; then
  validate_driver "$EXISTING_DRIVER"
  echo "Docker network ${NETWORK_NAME} already exists."
  exit 0
fi

if "$DOCKER_BIN" network create \
  --driver bridge \
  --label "$NETWORK_LABEL" \
  "$NETWORK_NAME" >/dev/null; then
  CREATED_DRIVER="$(inspect_driver)"
  validate_driver "$CREATED_DRIVER"
  echo "Docker network ${NETWORK_NAME} created."
  exit 0
fi

if RACE_DRIVER="$(inspect_driver)"; then
  validate_driver "$RACE_DRIVER"
  echo "Docker network ${NETWORK_NAME} was created concurrently."
  exit 0
fi

echo "Unable to create Docker network ${NETWORK_NAME}." >&2
exit 1
