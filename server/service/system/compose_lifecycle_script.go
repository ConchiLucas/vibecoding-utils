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
		!hasExplicitComposeLifecycleCommand(content) {
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

func hasExplicitComposeLifecycleCommand(content string) bool {
	logicalContent := strings.ReplaceAll(content, "\\\r\n", " ")
	logicalContent = strings.ReplaceAll(logicalContent, "\\\n", " ")
	for _, line := range strings.Split(logicalContent, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || fields[0] != "docker" || fields[1] != "compose" {
			continue
		}

		hasProjectName := false
		hasExpectedConfig := false
		for index := 2; index < len(fields); index++ {
			field := fields[index]
			switch {
			case (field == "-p" || field == "--project-name") && index+1 < len(fields):
				hasProjectName = strings.TrimSpace(fields[index+1]) != ""
				index++
			case strings.HasPrefix(field, "--project-name="):
				hasProjectName = strings.TrimSpace(strings.TrimPrefix(field, "--project-name=")) != ""
			case (field == "-f" || field == "--file") && index+1 < len(fields):
				hasExpectedConfig = strings.Trim(fields[index+1], `"'`) == `$SCRIPT_DIR/docker-compose.yml`
				index++
			case strings.HasPrefix(field, "--file="):
				config := strings.Trim(strings.TrimPrefix(field, "--file="), `"'`)
				hasExpectedConfig = config == `$SCRIPT_DIR/docker-compose.yml`
			}
		}
		if hasProjectName && hasExpectedConfig {
			return true
		}
	}
	return false
}
