#!/bin/bash
# build.sh — 一键交叉编译 easy-deploy 到多平台
# 使用方法: chmod +x build.sh && ./build.sh

set -e

APP_NAME="easy-deploy"
OUTPUT_DIR="build/bin"
MAIN_PKG="."

mkdir -p "$OUTPUT_DIR"

echo "🔨 Building $APP_NAME for multiple platforms..."

build() {
  local GOOS=$1
  local GOARCH=$2
  local SUFFIX=$3
  local OUT="$OUTPUT_DIR/${APP_NAME}-${GOOS}-${GOARCH}${SUFFIX}"
  echo "  → $GOOS/$GOARCH"
  GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$OUT" $MAIN_PKG
}

# macOS
build darwin  arm64 ""
build darwin  amd64 ""

# Windows
build windows amd64 ".exe"

# Linux
build linux   amd64 ""

echo ""
echo "✅ Done! Binaries are in ./$OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR/"
