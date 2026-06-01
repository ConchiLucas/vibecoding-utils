#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$PROJECT_DIR/target"
POM_FILE="$PROJECT_DIR/pom.xml"
POM_HASH_FILE="$TARGET_DIR/.pom.sha256"
MODE="${1:-incremental}"

log_info()  { echo "[INFO] $1"; }
log_warn()  { echo "[WARN] $1"; }
log_step()  { echo "[STEP] $1"; }
log_error() { echo "[ERROR] $1"; }

pom_hash() {
    shasum -a 256 "$POM_FILE" | awk '{print $1}'
}

run_package() {
    mkdir -p "$TARGET_DIR"
    CURRENT_POM_HASH="$(pom_hash)"
    PREVIOUS_POM_HASH=""

    if [ -f "$POM_HASH_FILE" ]; then
        PREVIOUS_POM_HASH="$(cat "$POM_HASH_FILE")"
    fi

    case "$MODE" in
        full)
            log_step "full build: mvn clean package -DskipTests"
            mvn clean package -DskipTests
            ;;
        compile)
            if [ "$CURRENT_POM_HASH" = "$PREVIOUS_POM_HASH" ] && [ -n "$PREVIOUS_POM_HASH" ]; then
                log_step "offline compile: mvn clean package -o -DskipTests"
                mvn clean package -o -DskipTests
            else
                log_warn "pom.xml changed, refreshing dependencies"
                mvn clean package -DskipTests
            fi
            log_info "compile completed"
            ;;
        incremental)
            if [ "$CURRENT_POM_HASH" = "$PREVIOUS_POM_HASH" ] && [ -n "$PREVIOUS_POM_HASH" ]; then
                log_step "pom unchanged, offline package"
                mvn clean package -o -DskipTests
            else
                log_warn "pom.xml changed or first deploy, full package"
                mvn clean package -DskipTests
            fi
            ;;
        *)
            log_error "unsupported mode: $MODE"
            exit 1
            ;;
    esac

    printf '%s' "$CURRENT_POM_HASH" > "$POM_HASH_FILE"
}

deploy_runtime_image() {
    docker build -f "$PROJECT_DIR/Dockerfile.run" -t "{{ .ImageName }}" "$PROJECT_DIR"
    cd "$PROJECT_DIR"
    docker compose up -d --no-deps --no-build --force-recreate app
}

cd "$PROJECT_DIR"
run_package

if [ "$MODE" = "compile" ]; then
    exit 0
fi

deploy_runtime_image
