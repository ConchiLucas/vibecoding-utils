package system

import (
	"strings"
	"testing"
)

const explicitComposeStartScript = `#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

docker compose \
  -p rob-english-word-front \
  -f "$SCRIPT_DIR/docker-compose.yml" \
  up --build -d
`

func TestNormalizeComposeLifecycleScriptInjectsGuardedLegacyCleanup(t *testing.T) {
	normalized, changed := normalizeComposeLifecycleScript(explicitComposeStartScript, "start.sh")
	if !changed {
		t.Fatal("normalizeComposeLifecycleScript() did not update eligible start.sh")
	}
	for _, expected := range []string{
		composeLifecycleMigrationMarker,
		`label=com.docker.compose.project=$vibedeploy_legacy_compose_project`,
		`label=com.docker.compose.project.config_files=$vibedeploy_legacy_compose_config`,
		`docker rm -f $vibedeploy_legacy_compose_ids`,
		`docker compose \`,
	} {
		if !strings.Contains(normalized, expected) {
			t.Fatalf("normalized script missing %q:\n%s", expected, normalized)
		}
	}
}

func TestNormalizeComposeLifecycleScriptSupportsStopAndIsIdempotent(t *testing.T) {
	stop := strings.Replace(explicitComposeStartScript, "up --build -d", "down", 1)
	first, changed := normalizeComposeLifecycleScript(stop, "stop.sh")
	if !changed {
		t.Fatal("first normalization did not change stop.sh")
	}
	second, changed := normalizeComposeLifecycleScript(first, "stop.sh")
	if changed || second != first {
		t.Fatal("second normalization was not idempotent")
	}
	if strings.Count(second, composeLifecycleMigrationMarker) != 1 {
		t.Fatal("migration block was duplicated")
	}
}

func TestNormalizeComposeLifecycleScriptLeavesIneligibleScriptsUnchanged(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		content  string
	}{
		{name: "non lifecycle file", fileName: "build.sh", content: explicitComposeStartScript},
		{name: "legacy project naming", fileName: "start.sh", content: strings.Replace(explicitComposeStartScript, "  -p rob-english-word-front \\\n", "", 1)},
		{name: "unrelated mkdir project flag", fileName: "start.sh", content: strings.Replace(strings.Replace(explicitComposeStartScript, "  -p rob-english-word-front \\\n", "", 1), "\ndocker compose", "\nmkdir -p \"$SCRIPT_DIR/logs\"\n\ndocker compose", 1)},
		{name: "comment project flag", fileName: "start.sh", content: strings.Replace(strings.Replace(explicitComposeStartScript, "  -p rob-english-word-front \\\n", "", 1), "up --build -d", "up --build -d # migrate later with -p stable-name", 1)},
		{name: "subcommand project flag", fileName: "start.sh", content: strings.Replace(strings.Replace(explicitComposeStartScript, "  -p rob-english-word-front \\\n", "", 1), "up --build -d", "up --build -d -p stable-name", 1)},
		{name: "chained compose commands", fileName: "start.sh", content: "#!/bin/sh\nSCRIPT_DIR=$(pwd)\ndocker compose -p stable -f \"$SCRIPT_DIR/docker-compose.yml\" config && docker compose up -d\n"},
		{name: "different compose subcommand", fileName: "start.sh", content: strings.Replace(explicitComposeStartScript, "up --build -d", "config up", 1)},
		{name: "different config", fileName: "start.sh", content: strings.Replace(explicitComposeStartScript, "$SCRIPT_DIR/docker-compose.yml", "$SCRIPT_DIR/compose.yaml", 1)},
		{name: "non compose script", fileName: "start.sh", content: "#!/bin/sh\necho ok\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, changed := normalizeComposeLifecycleScript(test.content, test.fileName)
			if changed || got != test.content {
				t.Fatalf("ineligible script changed:\n%s", got)
			}
		})
	}
}
