#!/bin/sh
set -eu

IMAGE_TAR="${1:-{{ .ImageName }}.tar}"
IMAGE_NAME="${2:-{{ .ImageName }}}"
CONTAINER_NAME="${3:-{{ .ContainerName }}}"
APP_PORT="{{ .AppPort }}"

if [ ! -f "$IMAGE_TAR" ]; then
  echo "[ERROR] image tar not found: $IMAGE_TAR"
  exit 1
fi

echo "[STEP] loading image: $IMAGE_TAR"
docker load -i "$IMAGE_TAR"

echo "[STEP] recreating container: $CONTAINER_NAME"
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

ENV_ARGS=""
if [ -f ".env" ]; then
  ENV_ARGS="--env-file .env"
fi

docker run -d \
  $ENV_ARGS \
  -p "${APP_PORT}:${APP_PORT}" \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -v "$(pwd)/logs:/app/logs" \
  "$IMAGE_NAME"

echo "[INFO] remote incremental deploy completed: $CONTAINER_NAME"
