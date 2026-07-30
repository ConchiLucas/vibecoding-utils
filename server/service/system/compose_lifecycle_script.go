package system

import (
	"path/filepath"
	"strings"
	"unicode"
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
		commands, valid := parseShellCommandSegments(line)
		if !valid {
			continue
		}
		for _, fields := range commands {
			if isExplicitComposeLifecycleCommand(fields) {
				return true
			}
		}
	}
	return false
}

func isExplicitComposeLifecycleCommand(fields []string) bool {
	if len(fields) < 2 || fields[0] != "docker" || fields[1] != "compose" {
		return false
	}

	hasProjectName := false
	hasExpectedConfig := false
	for index := 2; index < len(fields); index++ {
		field := fields[index]
		switch {
		case field == "up" || field == "down":
			return hasProjectName && hasExpectedConfig
		case (field == "-p" || field == "--project-name") && index+1 < len(fields):
			hasProjectName = strings.TrimSpace(fields[index+1]) != ""
			index++
		case strings.HasPrefix(field, "--project-name="):
			hasProjectName = strings.TrimSpace(strings.TrimPrefix(field, "--project-name=")) != ""
		case (field == "-f" || field == "--file") && index+1 < len(fields):
			hasExpectedConfig = fields[index+1] == `$SCRIPT_DIR/docker-compose.yml`
			index++
		case strings.HasPrefix(field, "--file="):
			config := strings.TrimPrefix(field, "--file=")
			hasExpectedConfig = config == `$SCRIPT_DIR/docker-compose.yml`
		}
	}
	return false
}

func parseShellCommandSegments(line string) ([][]string, bool) {
	commands := make([][]string, 0)
	fields := make([]string, 0)
	var current strings.Builder
	var quote rune
	escaped := false
	tokenStarted := false

	flushToken := func() {
		if !tokenStarted {
			return
		}
		fields = append(fields, current.String())
		current.Reset()
		tokenStarted = false
	}
	flushCommand := func() {
		flushToken()
		if len(fields) == 0 {
			return
		}
		commands = append(commands, fields)
		fields = make([]string, 0)
	}

	characters := []rune(line)
	for index := 0; index < len(characters); index++ {
		character := characters[index]
		if escaped {
			current.WriteRune(character)
			tokenStarted = true
			escaped = false
			continue
		}
		if quote != 0 {
			switch {
			case character == quote:
				quote = 0
			case character == '\\' && quote == '"':
				escaped = true
			default:
				current.WriteRune(character)
			}
			continue
		}

		switch {
		case character == '\\':
			escaped = true
			tokenStarted = true
		case character == '\'' || character == '"':
			quote = character
			tokenStarted = true
		case character == '#' && !tokenStarted:
			flushCommand()
			return commands, true
		case character == ';' || character == '|' || character == '&':
			flushCommand()
			if index+1 < len(characters) && characters[index+1] == character && (character == '|' || character == '&') {
				index++
			}
		case unicode.IsSpace(character):
			flushToken()
		default:
			current.WriteRune(character)
			tokenStarted = true
		}
	}
	if escaped || quote != 0 {
		return nil, false
	}
	flushCommand()
	return commands, true
}
