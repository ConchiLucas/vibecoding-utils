#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/vibedeploy-network-test.XXXXXX")"
trap 'rm -rf "$TEST_DIR"' EXIT

FAKE_DOCKER="$TEST_DIR/docker"
cat > "$FAKE_DOCKER" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >> "$FAKE_DOCKER_STATE_DIR/calls"
if [ "$1 $2" = "network inspect" ]; then
  if [ ! -f "$FAKE_DOCKER_STATE_DIR/exists" ]; then
    exit 1
  fi
  cat "$FAKE_DOCKER_STATE_DIR/driver"
  exit 0
fi
if [ "$1 $2" = "network create" ]; then
  if [ -f "$FAKE_DOCKER_STATE_DIR/create_fails" ]; then
    if [ -f "$FAKE_DOCKER_STATE_DIR/race_driver" ]; then
      cp "$FAKE_DOCKER_STATE_DIR/race_driver" "$FAKE_DOCKER_STATE_DIR/driver"
      touch "$FAKE_DOCKER_STATE_DIR/exists"
    fi
    exit 1
  fi
  touch "$FAKE_DOCKER_STATE_DIR/exists"
  printf '%s\n' "bridge" > "$FAKE_DOCKER_STATE_DIR/driver"
  printf '%s\n' "fake-network-id"
  exit 0
fi
if [ "$1" = "compose" ]; then
  exit 0
fi
exit 2
FAKE
chmod +x "$FAKE_DOCKER"

run_existing_network_test() {
  local state_dir="$TEST_DIR/existing"
  mkdir -p "$state_dir"
  touch "$state_dir/exists"
  printf '%s\n' "bridge" > "$state_dir/driver"

  FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/ensure-docker-network.sh"

  if [ "$(wc -l < "$state_dir/calls" | tr -d ' ')" != "1" ]; then
    echo "existing network should only be inspected" >&2
    return 1
  fi
  grep -Fx 'network inspect --format {{.Driver}} vibedeploy-shared' "$state_dir/calls" >/dev/null
}

run_missing_network_test() {
  local state_dir="$TEST_DIR/missing"
  mkdir -p "$state_dir"

  FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/ensure-docker-network.sh"

  grep -Fx 'network inspect --format {{.Driver}} vibedeploy-shared' "$state_dir/calls" >/dev/null
  grep -Fx "network create --driver bridge --label com.vibedeploy.managed=true vibedeploy-shared" "$state_dir/calls" >/dev/null
  [ -f "$state_dir/exists" ]
}

run_wrong_driver_test() {
  local state_dir="$TEST_DIR/wrong-driver"
  mkdir -p "$state_dir"
  touch "$state_dir/exists"
  printf '%s\n' "overlay" > "$state_dir/driver"

  if FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/ensure-docker-network.sh"; then
    echo "wrong existing network driver should fail" >&2
    return 1
  fi
  if grep -F "network create" "$state_dir/calls" >/dev/null; then
    echo "wrong existing network should not be recreated automatically" >&2
    return 1
  fi
}

run_wrong_driver_race_test() {
  local state_dir="$TEST_DIR/wrong-driver-race"
  mkdir -p "$state_dir"
  touch "$state_dir/create_fails"
  printf '%s\n' "macvlan" > "$state_dir/race_driver"

  if FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/ensure-docker-network.sh"; then
    echo "wrong race-created network driver should fail" >&2
    return 1
  fi
}

run_compose_wrapper_test() {
  local state_dir="$TEST_DIR/compose-wrapper"
  mkdir -p "$state_dir"

  FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/docker-compose-up.sh" -f docker-compose.yml up -d

  grep -Fx "compose -f docker-compose.yml up -d" "$state_dir/calls" >/dev/null
}

run_existing_network_test
run_missing_network_test
run_wrong_driver_test
run_wrong_driver_race_test
run_compose_wrapper_test
echo "ensure-docker-network tests passed"
