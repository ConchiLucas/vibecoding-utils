package system

import (
	"path/filepath"
	"strings"
)

const composeLifecycleMigrationMarker = "# vibedeploy: migrate directory-derived compose project"

const composeLifecycleMigrationBlock = `
# vibedeploy: migrate directory-derived compose project
cleanup_vibedeploy_legacy_compose_containers() {
  vibedeploy_legacy_compose_project=$(basename "$SCRIPT_DIR")
  vibedeploy_legacy_compose_config="$SCRIPT_DIR/docker-compose.yml"
  vibedeploy_legacy_compose_ids=$(docker ps -aq \
    --filter "label=com.docker.compose.project=$vibedeploy_legacy_compose_project" \
    --filter "label=com.docker.compose.project.config_files=$vibedeploy_legacy_compose_config")
  if [ -n "$vibedeploy_legacy_compose_ids" ]; then
    docker rm -f $vibedeploy_legacy_compose_ids
  fi
}
cleanup_vibedeploy_legacy_compose_containers
`

func normalizeComposeLifecycleScript(content, fileName string) (string, bool) {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(fileName)))
	if base != "start.sh" && base != "stop.sh" {
		return content, false
	}
	if strings.Contains(content, composeLifecycleMigrationMarker) ||
		!strings.Contains(content, "docker compose") ||
		!strings.Contains(content, "-p ") ||
		!strings.Contains(content, `$SCRIPT_DIR/docker-compose.yml`) {
		return content, false
	}

	lines := strings.SplitAfter(content, "\n")
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "SCRIPT_DIR=") {
			lines[index] = strings.TrimSuffix(line, "\n") + "\n" + composeLifecycleMigrationBlock + "\n"
			return strings.Join(lines, ""), true
		}
	}
	return content, false
}
