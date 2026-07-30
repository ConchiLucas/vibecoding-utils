# Compose Project Collision Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make database-backed local Compose deployment scripts safely remove containers created by the same route under the legacy directory-derived Compose project before running the new explicitly named Compose project.

**Architecture:** Add a focused, idempotent shell-script normalizer and call it from the existing local script materialization pipeline. The normalizer injects a guarded Docker-label cleanup block only into eligible `start.sh` and `stop.sh` files; the existing transactional publisher persists normalized content back to `tb_project_script`.

**Tech Stack:** Go, GORM/SQLite tests, POSIX shell, Docker Compose labels.

---

### Task 1: Define lifecycle-script normalization behavior

**Files:**
- Create: `server/service/system/compose_lifecycle_script_test.go`
- Create: `server/service/system/compose_lifecycle_script.go`

- [ ] **Step 1: Write failing unit tests**

Create tests for start scripts, stop scripts, idempotence, and ineligible scripts:

```go
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
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server
go test ./service/system -run 'TestNormalizeComposeLifecycleScript' -count=1
```

Expected: compilation fails because `normalizeComposeLifecycleScript` and `composeLifecycleMigrationMarker` do not exist.

- [ ] **Step 3: Implement the minimal normalizer**

Create `server/service/system/compose_lifecycle_script.go` with:

```go
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
```

- [ ] **Step 4: Run focused tests and verify GREEN**

Run:

```bash
cd server
go test ./service/system -run 'TestNormalizeComposeLifecycleScript' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/service/system/compose_lifecycle_script.go server/service/system/compose_lifecycle_script_test.go
git commit -m "fix: normalize compose lifecycle scripts"
```

### Task 2: Persist normalized lifecycle scripts through materialization

**Files:**
- Modify: `server/service/system/local_script_materializer.go`
- Modify: `server/service/system/local_script_materializer_test.go`

- [ ] **Step 1: Write a failing database persistence test**

Add a test that creates an eligible `start.sh`, materializes it, and verifies both the file and database contain exactly one migration marker:

```go
func TestPublishPreparedLocalScriptsPersistsComposeLifecycleMigration(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	project, route := createAggregateChildRoute(t, db, 6, "vue", "rob-english-word-front-web", root, filepath.Join(root, "deploy", "local_full"), 0, false)
	script := modelSystem.TbProjectScript{
		ProjectId: int(project.ID),
		RouteId: int(route.ID),
		ScriptType: 1,
		FileName: "start.sh",
		Content: explicitComposeStartScript,
	}
	if err := db.Create(&script).Error; err != nil {
		t.Fatal(err)
	}
	prepared, err := loadLocalScriptsForMaterialization(db, []localScriptMaterializationRequest{
		createMaterializationRequest(t, project, route, route.LocalScriptPath),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := publishPreparedLocalScripts(db, prepared, nil); err != nil {
		t.Fatal(err)
	}
	fileContent, err := os.ReadFile(filepath.Join(route.LocalScriptPath, "start.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var reloaded modelSystem.TbProjectScript
	if err := db.First(&reloaded, script.ID).Error; err != nil {
		t.Fatal(err)
	}
	for source, content := range map[string]string{"file": string(fileContent), "database": reloaded.Content} {
		if strings.Count(content, composeLifecycleMigrationMarker) != 1 {
			t.Fatalf("%s migration marker count = %d, want 1:\n%s", source, strings.Count(content, composeLifecycleMigrationMarker), content)
		}
	}
}
```

- [ ] **Step 2: Run the persistence test and verify RED**

Run:

```bash
cd server
go test ./service/system -run 'TestPublishPreparedLocalScriptsPersistsComposeLifecycleMigration' -count=1
```

Expected: FAIL because `normalizeLocalScriptForDeploy` has not invoked the lifecycle normalizer.

- [ ] **Step 3: Integrate normalization**

In `normalizeLocalScriptForDeploy`, add:

```go
	if normalized, changed := normalizeComposeLifecycleScript(content, script.FileName); changed {
		content = normalized
	}
```

Place it before Compose YAML normalization so shell and YAML paths remain independent.

- [ ] **Step 4: Run focused and package tests**

Run:

```bash
cd server
go test ./service/system -run 'TestNormalizeComposeLifecycleScript|TestPublishPreparedLocalScriptsPersistsComposeLifecycleMigration' -count=1
go test ./service/system -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/service/system/local_script_materializer.go server/service/system/local_script_materializer_test.go
git commit -m "fix: persist compose collision recovery"
```

### Task 3: Apply and verify the live English-word deployment

**Files:**
- Verify: live `tb_project_script` rows materialized under `/Users/conchi/workforce/rob_english_word_workforce/deploy`

- [ ] **Step 1: Run complete server verification**

Run:

```bash
cd server
go test ./... -count=1
```

Expected: all packages PASS.

- [ ] **Step 2: Integrate the verified branch into the active workspace**

Use the finishing-development workflow to present and execute the selected integration option. Do not start the live migration from an unintegrated worktree.

- [ ] **Step 3: Restart VibeDeploy**

From the active repository root:

```bash
./scripts/restart-dev.sh restart
```

Expected: backend and frontend health checks pass. Startup linkage materializes normalized scripts and starts the enabled English-word group.

- [ ] **Step 4: Verify database and materialized scripts**

Confirm the English-word local `start.sh` and `stop.sh` files contain exactly one migration marker. Confirm backend logs show the scripts were loaded from the database and no `container name is already in use` error.

- [ ] **Step 5: Verify containers and ports**

Confirm the six expected services listen on ports `6011`, `6012`, `6014`, `6015`, `6016`, and `6017`. Open the three frontend URLs and verify each renders application content rather than `ERR_CONNECTION_REFUSED`.

- [ ] **Step 6: Verify repeat deployment**

Trigger the English-word aggregate full deployment again and confirm it completes without a container-name conflict. Recheck all six ports.
