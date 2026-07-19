#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d /private/tmp/vibedeploy-network-test.XXXXXX)"
trap 'rm -rf "$TEST_DIR"' EXIT

FAKE_DOCKER="$TEST_DIR/docker"
cat > "$FAKE_DOCKER" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >> "$FAKE_DOCKER_STATE_DIR/calls"
if [ "$1 $2" = "network inspect" ]; then
  [ -f "$FAKE_DOCKER_STATE_DIR/exists" ]
  exit $?
fi
if [ "$1 $2" = "network create" ]; then
  touch "$FAKE_DOCKER_STATE_DIR/exists"
  printf '%s\n' "fake-network-id"
  exit 0
fi
exit 2
FAKE
chmod +x "$FAKE_DOCKER"

run_existing_network_test() {
  local state_dir="$TEST_DIR/existing"
  mkdir -p "$state_dir"
  touch "$state_dir/exists"

  FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/ensure-docker-network.sh"

  if [ "$(wc -l < "$state_dir/calls" | tr -d ' ')" != "1" ]; then
    echo "existing network should only be inspected" >&2
    return 1
  fi
  grep -Fx "network inspect vibedeploy-shared" "$state_dir/calls" >/dev/null
}

run_missing_network_test() {
  local state_dir="$TEST_DIR/missing"
  mkdir -p "$state_dir"

  FAKE_DOCKER_STATE_DIR="$state_dir" DOCKER_BIN="$FAKE_DOCKER" \
    "$ROOT_DIR/scripts/ensure-docker-network.sh"

  grep -Fx "network inspect vibedeploy-shared" "$state_dir/calls" >/dev/null
  grep -Fx "network create --driver bridge --label com.vibedeploy.managed=true vibedeploy-shared" "$state_dir/calls" >/dev/null
  [ -f "$state_dir/exists" ]
}

run_existing_network_test
run_missing_network_test
echo "ensure-docker-network tests passed"
