# Aggregate Route Script Materialization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a local aggregate deployment materialize every child route referenced by its database-backed `start.sh` before executing any build or Compose command.

**Architecture:** Parse canonical `$ROOT_DIR/.../start.sh` invocations from the selected aggregate route, resolve each path to exactly one local route in the same project group, preflight all referenced scripts in memory, then publish the complete batch with database and filesystem rollback. Reuse the same batch materializer for ordinary single-route downloads so normalization and atomicity have one implementation.

**Tech Stack:** Go 1.23, GORM, SQLite test database, PostgreSQL production database, filesystem atomic rename, Docker Compose validation.

---

### Task 1: Parse aggregate child route references

**Files:**
- Create: `server/service/system/aggregate_route_materializer.go`
- Create: `server/service/system/aggregate_route_materializer_test.go`

- [ ] **Step 1: Write failing parser tests**

Add table-driven tests for quoted `sh`, quoted `bash` with arguments, unquoted canonical paths, order preservation, duplicate removal, comments, unsupported syntax, traversal, self-reference, and empty scripts. The main contract is:

```go
func TestParseAggregateChildScriptPaths(t *testing.T) {
	root := t.TempDir()
	aggregateDir := filepath.Join(root, "deploy", "compose", "full")
	content := `sh "$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"
bash "$ROOT_DIR/deploy/backend/word_agent/local_full/start.sh" --force`
	got, err := parseAggregateChildScriptPaths(root, aggregateDir, content)
	if err != nil { t.Fatal(err) }
	want := []string{
		filepath.Join(root, "deploy", "backend", "word_agent", "build_project"),
		filepath.Join(root, "deploy", "backend", "word_agent", "local_full"),
	}
	if !reflect.DeepEqual(got, want) { t.Fatalf("paths = %#v, want %#v", got, want) }
}
```

- [ ] **Step 2: Run parser tests and confirm RED**

Run `cd server && go test ./service/system -run 'TestParseAggregateChildScriptPaths' -count=1`.
Expected: compile failure because `parseAggregateChildScriptPaths` is undefined.

- [ ] **Step 3: Implement the minimal parser**

Create the following API and canonical invocation regex:

```go
var aggregateChildStartPattern = regexp.MustCompile(`^\s*(?:sh|bash)\s+(?:"\$ROOT_DIR/([^"]+/start\.sh)"|'\$ROOT_DIR/([^']+/start\.sh)'|\$ROOT_DIR/([^\s;]+/start\.sh))(?:\s+.*)?\s*$`)

func parseAggregateChildScriptPaths(rootPath, aggregateScriptPath, content string) ([]string, error)
```

The function must require an absolute root, scan line-by-line, ignore blanks/comments/unrelated commands, fail with line number when a line mentions `$ROOT_DIR/.../start.sh` in unsupported syntax, clean the extracted relative path, reject absolute/traversal/outside-root/self references, preserve order, de-duplicate normalized directories, and require at least one child.

- [ ] **Step 4: Run parser tests and confirm GREEN**

Repeat Step 2. Expected: all parser tests pass.

- [ ] **Step 5: Commit the parser**

```bash
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go
git commit -m "feat: parse aggregate route dependencies"
```

### Task 2: Resolve referenced paths to exact child routes

**Files:**
- Modify: `server/service/system/aggregate_route_materializer.go`
- Modify: `server/service/system/aggregate_route_materializer_test.go`

- [ ] **Step 1: Write failing resolver tests**

Create an in-memory SQLite database that migrates `TbProject`, `TbProjectRoute`, and `TbProjectScript`. Insert one aggregate plus Java, Python, Go, Vue, and React children. Test ordered exact resolution and separate no-match, duplicate-match, remote-only, missing/duplicate `start.sh`, and empty-`local_script_path` fallback cases.

```go
type aggregateChildRoute struct {
	Project    modelSystem.TbProject
	Route      modelSystem.TbProjectRoute
	ScriptPath string
}

children, err := resolveAggregateChildRoutes(db, aggregate, aggregateRoute, references)
```

- [ ] **Step 2: Run resolver tests and confirm RED**

Run `cd server && go test ./service/system -run 'TestResolveAggregateChildRoutes' -count=1`.
Expected: compile failure because the resolver does not exist.

- [ ] **Step 3: Implement exact route resolution**

```go
func loadSingleLocalStartScript(db *gorm.DB, projectID, routeID uint) (modelSystem.TbProjectScript, error)
func resolveAggregateChildRoutes(db *gorm.DB, aggregate modelSystem.TbProject, aggregateRoute modelSystem.TbProjectRoute, references []string) ([]aggregateChildRoute, error)
```

`loadSingleLocalStartScript` must query exact project/route, `file_name = start.sh`, `script_type <> 2`, and require exactly one active row. The resolver loads active projects in the same group with routes preloaded, skips the aggregate and other aggregate projects, considers only `server_id = 0`, compares `filepath.Clean(resolveLocalScriptPath(candidateRoute, child.LocalProjectPath))` exactly, requires one candidate, and validates its entry script. Diagnostics include aggregate IDs, reference, and conflicting candidate IDs.

- [ ] **Step 4: Run resolver tests and confirm GREEN**

Repeat Step 2. Expected: all resolver tests pass.

- [ ] **Step 5: Commit the resolver**

```bash
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go
git commit -m "feat: resolve aggregate child routes"
```

### Task 3: Refactor script download into an atomic batch materializer

**Files:**
- Create: `server/service/system/local_script_materializer.go`
- Create: `server/service/system/local_script_materializer_test.go`
- Modify: `server/service/system/sys_deploy.go:546-659`
- Modify: `server/service/system/docker_compose_network_test.go`

- [ ] **Step 1: Write failing batch preflight and rollback tests**

Test this internal API:

```go
type localScriptMaterializationRequest struct {
	Project    modelSystem.TbProject
	RouteID    uint
	ScriptPath string
}

prepared, err := loadLocalScriptsForMaterialization(db, requests)
err = publishPreparedLocalScripts(db, prepared, func(index int) error {
	if index == 1 { return errors.New("forced publish failure") }
	return nil
})
```

Prove malformed final Compose causes no writes; conflicting duplicate targets fail preflight; forced failure restores an existing file and removes a new file; optimistic DB conflict prevents publication; success applies frontend/Python/shared-network normalization with mode `0644`; and ordinary `downloadScriptsToLocalFromDB` still works.

- [ ] **Step 2: Run batch tests and confirm RED**

Run `cd server && go test ./service/system -run 'Test(LoadLocalScriptsForMaterialization|PublishPreparedLocalScripts|DownloadScripts)' -count=1`.
Expected: compile failure for the new batch APIs.

- [ ] **Step 3: Implement the reusable load phase**

```go
type preparedLocalScript struct {
	Script   modelSystem.TbProjectScript
	Content  string
	FilePath string
	TempPath string
}

func loadLocalScriptsForMaterialization(db *gorm.DB, requests []localScriptMaterializationRequest) ([]preparedLocalScript, error)
func normalizeLocalScriptForDeploy(project modelSystem.TbProject, script modelSystem.TbProjectScript) (string, error)
```

Load exact active non-remote project/route scripts, reject unsafe names, normalize entirely in memory with existing functions, require at least one local script per request, and reject duplicate targets with different content. This phase performs no filesystem or database write.

- [ ] **Step 4: Implement transactional publication and rollback**

```go
type localScriptPublishState struct {
	TargetPath string
	TempPath   string
	BackupPath string
	Existed    bool
	Published  bool
}

func publishPreparedLocalScripts(db *gorm.DB, prepared []preparedLocalScript, beforePublish func(int) error) error
func rollbackPublishedLocalScripts(states []localScriptPublishState) error
func cleanupPublishedLocalScripts(states []localScriptPublishState)
```

Stage all files before a transaction. Inside `db.Transaction`, update normalized rows with optimistic original-content conditions, optionally invoke the test failure hook, rename existing targets to same-directory backups, and rename staged files into place. On callback or commit error, restore states in reverse order. On success, remove backups/temps. Track staging-created directories and remove only those still empty during rollback.

- [ ] **Step 5: Route ordinary downloads through the batch path**

Replace `downloadScriptsToLocalFromDB` internals with project loading, one `localScriptMaterializationRequest`, `loadLocalScriptsForMaterialization`, and `publishPreparedLocalScripts`. Delete the duplicated preparation/staging code from `sys_deploy.go` only after tests pass.

- [ ] **Step 6: Run materializer and service suites**

```bash
cd server
go test ./service/system -run 'Test(LoadLocalScriptsForMaterialization|PublishPreparedLocalScripts|DownloadScripts)' -count=1
go test ./service/system -count=1
```

Expected: both commands pass.

- [ ] **Step 7: Commit the batch materializer**

```bash
git add server/service/system/local_script_materializer.go server/service/system/local_script_materializer_test.go server/service/system/sys_deploy.go server/service/system/docker_compose_network_test.go
git commit -m "feat: atomically materialize deployment scripts"
```

### Task 4: Wire generic aggregate preparation into deploy and stop flows

**Files:**
- Modify: `server/service/system/aggregate_route_materializer.go`
- Modify: `server/service/system/aggregate_route_materializer_test.go`
- Modify: `server/service/system/sys_deploy.go:107-121,662-700,1058-1068`

- [ ] **Step 1: Write failing English-style regression tests**

Build one SQLite group with an aggregate project and seven referenced routes: Python build, Python Compose, Java, Go, Vue, and two React projects. Store route-specific scripts and one unrelated incremental route. Call:

```go
err := DeployServiceApp.prepareAggregateChildDeployScripts(aggregate, aggregateRoute, logCh)
```

Assert the aggregate directory and all seven referenced directories contain their database scripts, the unrelated route is untouched, and logs contain the child dependency count and completion. Add a preflight-failure test proving neither the aggregate route nor an earlier child is written when the seventh reference is unresolved.

- [ ] **Step 2: Run aggregate regression tests and confirm RED**

Run `cd server && go test ./service/system -run 'TestPrepareAggregateChildDeployScripts' -count=1`.
Expected: compile failure because the existing method has no log channel and still uses Go/React detection.

- [ ] **Step 3: Implement generic aggregate preparation**

Use this exact control flow:

```go
func (s *DeployService) prepareAggregateChildDeployScripts(project modelSystem.TbProject, route modelSystem.TbProjectRoute, logCh chan string) error {
	if strings.TrimSpace(project.ComputerLanguage) != deployProjectTypeDockerCompose { return nil }
	sendAggregateDeployLog(logCh, "🧩 解析聚合部署依赖...")
	aggregateStart, err := loadSingleLocalStartScript(global.GVA_DB, project.ID, route.ID)
	if err != nil { return fmt.Errorf("读取聚合路线入口失败(project=%d route=%d): %w", project.ID, route.ID, err) }
	aggregatePath := resolveLocalScriptPath(route, project.LocalProjectPath)
	references, err := parseAggregateChildScriptPaths(project.LocalProjectPath, aggregatePath, aggregateStart.Content)
	if err != nil { return err }
	children, err := resolveAggregateChildRoutes(global.GVA_DB, project, route, references)
	if err != nil { return err }
	sendAggregateDeployLog(logCh, fmt.Sprintf("✅ 发现 %d 条子项目部署路线", len(children)))
	requests := make([]localScriptMaterializationRequest, 0, len(children)+1)
	requests = append(requests, localScriptMaterializationRequest{Project: project, RouteID: route.ID, ScriptPath: aggregatePath})
	for index, child := range children {
		sendAggregateDeployLog(logCh, fmt.Sprintf("📦 [%d/%d] 准备 %s / %s", index+1, len(children), child.Project.ProjectName, child.Route.RouteName))
		requests = append(requests, localScriptMaterializationRequest{Project: child.Project, RouteID: child.Route.ID, ScriptPath: child.ScriptPath})
	}
	prepared, err := loadLocalScriptsForMaterialization(global.GVA_DB, requests)
	if err != nil { return err }
	if err := publishPreparedLocalScripts(global.GVA_DB, prepared, nil); err != nil { return err }
	sendAggregateDeployLog(logCh, "✅ 聚合部署依赖脚本已全部落盘")
	return nil
}
```

The aggregate request is deliberately the first item so its complete script set is preflighted in the same batch as all dependencies. The later ordinary aggregate download in `processLocalDeploy` remains as an idempotent compatibility step. Update deploy and stop callers to pass `logCh`. Remove `downloadChildProjectRouteScripts`; retain `detectGoReactComposeParts` because project creation still uses it.

- [ ] **Step 4: Run aggregate regression and scoped suite**

```bash
cd server
go test ./service/system -run 'TestPrepareAggregateChildDeployScripts' -count=1
go test ./service/system -count=1
```

Expected: both commands pass.

- [ ] **Step 5: Commit deployment integration**

```bash
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go server/service/system/sys_deploy.go
git commit -m "feat: materialize aggregate route dependencies"
```

### Task 5: Verification, live acceptance, and conditional adoption

**Files:**
- Modify only files from Tasks 1-4 if a test exposes a defect.

- [ ] **Step 1: Run formatting and source checks**

```bash
gofmt -w server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go server/service/system/local_script_materializer.go server/service/system/local_script_materializer_test.go server/service/system/sys_deploy.go server/service/system/docker_compose_network_test.go
git diff --check
```

Expected: no formatting or whitespace errors.

- [ ] **Step 2: Run focused, race, vet, and shared-network verification**

```bash
cd server
go test ./service/system -count=1
go test -race ./service/system -run 'Test(ParseAggregate|ResolveAggregate|LoadLocal|PublishPrepared|PrepareAggregate|RunLocalDeployWithSharedNetwork)' -count=1
go vet ./service/system
```

Run the existing shared-network shell tests and validate all seven live Compose documents with `docker compose config`. Expected: every command exits zero.

- [ ] **Step 3: Record the exact live rollback snapshot**

Create a temporary snapshot directory with `mktemp -d`. Record English `deploy/` paths and hashes, English container IDs/states/network modes/ports, Docker networks, and current commit IDs. Do not alter middleware, volumes, images, or database records.

- [ ] **Step 4: Execute English aggregate project 82 route 149 through VibeDeploy**

Use the same backend deployment path as the UI. Verify all seven child route directories materialize, the aggregate command completes, expected English containers run, each persists `HostConfig.NetworkMode=vibedeploy-shared`, only the shared network is attached, no project network appears, and configured ports respond.

- [ ] **Step 5: Adopt or roll back**

If every check passes, keep the aggregate commits on `codex/shared-docker-network`, report evidence, and offer the existing branch integration choices. If any check fails, stop, revert only Tasks 1-4 commits, restore exact English files and pre-existing container states from the snapshot, remove only test-created English containers/files, and verify middleware plus `vibedeploy-shared` remain unchanged.
