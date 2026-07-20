# Generic Aggregate Route Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every supported local Docker Compose aggregate route materialize its referenced child deployment scripts on a fresh computer without misclassifying ordinary Compose routes.

**Architecture:** Add two narrow, package-local predicates in the existing aggregate materializer: one canonicalizes the explicit supported Compose type aliases, and one combines that classification with local `frontend_backend_*` route metadata. Use those predicates at the preflight entry and child-candidate exclusion boundaries while leaving the existing parser, resolver, atomic publisher, and rollback path unchanged.

**Tech Stack:** Go 1.24, GORM, `glebarez/sqlite` service tests, Docker/Compose, PostgreSQL-backed VibeDeploy route scripts.

---

## File Structure

- Modify `server/service/system/aggregate_route_materializer.go`: own Compose-type and aggregate-route classification, use it to gate preflight, and exclude orchestration projects from child candidates.
- Modify `server/service/system/aggregate_route_materializer_test.go`: table-test metadata boundaries, reproduce task-center alias materialization on an empty filesystem, and guard alias child exclusion.
- No database migrations, UI changes, project IDs, project names, or repository paths are added to production code.

### Task 1: Define strict metadata classification

**Files:**
- Modify: `server/service/system/aggregate_route_materializer_test.go:17`
- Modify: `server/service/system/aggregate_route_materializer.go:17`

- [x] **Step 1: Write failing table tests for supported Compose aliases and aggregate route boundaries**

Insert these tests before the parser tests:

```go
func TestIsDockerComposeProjectType(t *testing.T) {
	tests := []struct {
		name     string
		language string
		want     bool
	}{
		{name: "editor value", language: "docker-compose", want: true},
		{name: "spaced value", language: "docker compose", want: true},
		{name: "canonical value", language: "前后端 docker-compose", want: true},
		{name: "canonical spaced value", language: "前后端 docker compose", want: true},
		{name: "case and whitespace", language: "  DOCKER-COMPOSE  ", want: true},
		{name: "collapsed whitespace", language: "前后端   docker compose", want: true},
		{name: "malformed substring", language: "not-docker-compose", want: false},
		{name: "ordinary language", language: "go", want: false},
		{name: "empty", language: " ", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isDockerComposeProjectType(test.language); got != test.want {
				t.Fatalf("isDockerComposeProjectType(%q) = %v, want %v", test.language, got, test.want)
			}
		})
	}
}

func TestIsAggregateDeployRouteRequiresComposeLocalFrontendBackendMetadata(t *testing.T) {
	tests := []struct {
		name     string
		project  modelSystem.TbProject
		route    modelSystem.TbProjectRoute
		want     bool
	}{
		{name: "editor alias full", project: modelSystem.TbProject{ComputerLanguage: "docker-compose"}, route: modelSystem.TbProjectRoute{RouteKey: "frontend_backend_full", ServerId: 0}, want: true},
		{name: "canonical incremental", project: modelSystem.TbProject{ComputerLanguage: "前后端 docker-compose"}, route: modelSystem.TbProjectRoute{RouteKey: " FRONTEND_BACKEND_INCREMENTAL ", ServerId: 0}, want: true},
		{name: "remote aggregate", project: modelSystem.TbProject{ComputerLanguage: "docker-compose"}, route: modelSystem.TbProjectRoute{RouteKey: "frontend_backend_full", ServerId: 7}, want: false},
		{name: "ordinary compose route", project: modelSystem.TbProject{ComputerLanguage: "docker-compose"}, route: modelSystem.TbProjectRoute{RouteKey: "local_full", ServerId: 0}, want: false},
		{name: "name is not metadata", project: modelSystem.TbProject{ComputerLanguage: "go", ProjectName: "misleading-compose"}, route: modelSystem.TbProjectRoute{RouteKey: "frontend_backend_full", ServerId: 0}, want: false},
		{name: "empty route key", project: modelSystem.TbProject{ComputerLanguage: "docker-compose"}, route: modelSystem.TbProjectRoute{RouteKey: "", ServerId: 0}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAggregateDeployRoute(test.project, test.route); got != test.want {
				t.Fatalf("isAggregateDeployRoute(%q, %q, %d) = %v, want %v", test.project.ComputerLanguage, test.route.RouteKey, test.route.ServerId, got, test.want)
			}
		})
	}
}
```

- [x] **Step 2: Run the new tests and verify RED**

Run from `server/`:

```sh
go test ./service/system -run 'TestIsDockerComposeProjectType|TestIsAggregateDeployRoute' -count=1
```

Expected: build fails because `isDockerComposeProjectType` and `isAggregateDeployRoute` are undefined.

- [x] **Step 3: Implement the minimal strict predicates**

Insert after `aggregateChildRoute` in `aggregate_route_materializer.go`:

```go
func isDockerComposeProjectType(language string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(language), " "))
	switch normalized {
	case "docker-compose", "docker compose", "前后端 docker-compose", "前后端 docker compose":
		return true
	default:
		return false
	}
}

func isAggregateDeployRoute(project modelSystem.TbProject, route modelSystem.TbProjectRoute) bool {
	routeKey := strings.ToLower(strings.TrimSpace(route.RouteKey))
	return isDockerComposeProjectType(project.ComputerLanguage) &&
		route.ServerId == 0 &&
		strings.HasPrefix(routeKey, "frontend_backend_")
}
```

- [x] **Step 4: Run classification tests and verify GREEN**

Run:

```sh
go test ./service/system -run 'TestIsDockerComposeProjectType|TestIsAggregateDeployRoute' -count=1
```

Expected: `ok github.com/flipped-aurora/easy-deploy/server/service/system`.

- [x] **Step 5: Commit the classification contract**

```sh
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go
git commit -m "test: define aggregate route metadata contract"
```

### Task 2: Materialize task-center-shaped alias dependencies

**Files:**
- Modify: `server/service/system/aggregate_route_materializer_test.go:210`
- Modify: `server/service/system/aggregate_route_materializer.go:147`

- [x] **Step 1: Write a failing fresh-filesystem materialization test for the editor alias**

Add this test next to the existing English-style materialization test:

```go
func TestPrepareAggregateChildDeployScriptsMaterializesDockerComposeAliasDependencies(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	oldDB := global.GVA_DB
	oldLog := global.GVA_LOG
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_DB = oldDB
		global.GVA_LOG = oldLog
	})

	root := t.TempDir()
	aggregatePath := filepath.Join(root, "deploy", "compose", "frontend_backend_full")
	aggregate := modelSystem.TbProject{GroupId: 42, ComputerLanguage: "docker-compose", ProjectName: "orchestration", LocalProjectPath: root}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	aggregateRoute := modelSystem.TbProjectRoute{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", RouteName: "全部项目全量部署", ServerId: 0, LocalScriptPath: aggregatePath}
	if err := db.Create(&aggregateRoute).Error; err != nil {
		t.Fatal(err)
	}

	dependencies := []struct {
		language string
		name     string
		path     string
	}{
		{language: "python", name: "worker", path: filepath.Join(root, "deploy", "backend", "python_worker", "local_full")},
		{language: "java", name: "server", path: filepath.Join(root, "deploy", "backend", "java_server", "local_full")},
		{language: "react", name: "web", path: filepath.Join(root, "deploy", "frontend", "web_react", "local_full")},
	}

	lines := make([]string, 0, len(dependencies))
	for _, dependency := range dependencies {
		project, route := createAggregateChildRoute(t, db, aggregate.GroupId, dependency.language, dependency.name, filepath.Join(root, dependency.name), dependency.path, 0, false)
		script := modelSystem.TbProjectScript{ProjectId: int(project.ID), RouteId: int(route.ID), ScriptType: 1, FileName: "start.sh", Content: "#!/bin/sh\necho " + dependency.name + "\n"}
		if err := db.Create(&script).Error; err != nil {
			t.Fatal(err)
		}
		reference, err := filepath.Rel(root, filepath.Join(dependency.path, "start.sh"))
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, `sh "$ROOT_DIR/`+filepath.ToSlash(reference)+`"`)
	}
	aggregateStart := modelSystem.TbProjectScript{ProjectId: int(aggregate.ID), RouteId: int(aggregateRoute.ID), ScriptType: 1, FileName: "start.sh", Content: strings.Join(lines, "\n") + "\n"}
	if err := db.Create(&aggregateStart).Error; err != nil {
		t.Fatal(err)
	}

	logCh := make(chan string, 16)
	if err := DeployServiceApp.prepareAggregateChildDeployScripts(aggregate, aggregateRoute, logCh); err != nil {
		t.Fatalf("prepareAggregateChildDeployScripts() error = %v", err)
	}
	close(logCh)
	var logs []string
	for line := range logCh {
		logs = append(logs, line)
	}
	if joined := strings.Join(logs, "\n"); !strings.Contains(joined, "发现 3 条") || !strings.Contains(joined, "已全部落盘") {
		t.Fatalf("logs = %q, want alias preflight and completion", joined)
	}
	for _, target := range append([]string{aggregatePath}, dependencies[0].path, dependencies[1].path, dependencies[2].path) {
		if _, err := os.Stat(filepath.Join(target, "start.sh")); err != nil {
			t.Fatalf("start.sh was not materialized in %s: %v", target, err)
		}
	}
}
```

- [x] **Step 2: Run the task-center-shaped test and verify RED**

Run:

```sh
go test ./service/system -run TestPrepareAggregateChildDeployScriptsMaterializesDockerComposeAliasDependencies -count=1
```

Expected: FAIL because the current exact-language guard returns without logs or files for `docker-compose`.

- [x] **Step 3: Replace the exact-language preflight gate**

Change the beginning of `prepareAggregateChildDeployScripts` to:

```go
func (s *DeployService) prepareAggregateChildDeployScripts(project modelSystem.TbProject, route modelSystem.TbProjectRoute, logCh chan string) error {
	if !isAggregateDeployRoute(project, route) {
		return nil
	}
```

Keep the remaining parser, resolver, loader, publisher, and logs unchanged.

- [x] **Step 4: Run alias and canonical materialization tests and verify GREEN**

Run:

```sh
go test ./service/system -run 'TestPrepareAggregateChildDeployScriptsMaterializes(DockerComposeAlias|EnglishStyle)Dependencies' -count=1
```

Expected: both tests pass; the alias test reports three dependencies and the existing canonical test reports seven.

- [x] **Step 5: Commit generic preflight activation**

```sh
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go
git commit -m "fix: activate aggregate preflight for compose aliases"
```

### Task 3: Exclude every supported orchestration alias from child candidates

**Files:**
- Modify: `server/service/system/aggregate_route_materializer_test.go:108`
- Modify: `server/service/system/aggregate_route_materializer.go:115`

- [x] **Step 1: Write a failing ambiguity regression test**

Add after `TestResolveAggregateChildRoutesPreservesReferencesAcrossLanguages`:

```go
func TestResolveAggregateChildRoutesExcludesDockerComposeAliases(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	aggregate := modelSystem.TbProject{GroupId: 77, ComputerLanguage: deployProjectTypeDockerCompose, ProjectName: "aggregate", LocalProjectPath: root}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	aggregateRoute := modelSystem.TbProjectRoute{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", ServerId: 0, LocalScriptPath: filepath.Join(root, "deploy", "aggregate")}
	if err := db.Create(&aggregateRoute).Error; err != nil {
		t.Fatal(err)
	}

	reference := filepath.Join(root, "deploy", "backend", "worker", "local_full")
	_, expectedRoute := createAggregateChildRoute(t, db, aggregate.GroupId, "python", "worker", filepath.Join(root, "worker"), reference, 0, true)
	createAggregateChildRoute(t, db, aggregate.GroupId, "docker-compose", "nested-orchestration", filepath.Join(root, "nested"), reference, 0, true)

	children, err := resolveAggregateChildRoutes(db, aggregate, aggregateRoute, []string{reference})
	if err != nil {
		t.Fatalf("resolveAggregateChildRoutes() error = %v", err)
	}
	if len(children) != 1 || children[0].Route.ID != expectedRoute.ID {
		t.Fatalf("children = %#v, want only route %d", children, expectedRoute.ID)
	}
}
```

- [x] **Step 2: Run the exclusion test and verify RED**

Run:

```sh
go test ./service/system -run TestResolveAggregateChildRoutesExcludesDockerComposeAliases -count=1
```

Expected: FAIL with `聚合子路线匹配到 2 条`; the current resolver excludes only the exact canonical value.

- [x] **Step 3: Use the shared Compose predicate when filtering candidates**

Replace the candidate guard in `resolveAggregateChildRoutes` with:

```go
if project.ID == aggregate.ID || isDockerComposeProjectType(project.ComputerLanguage) {
	continue
}
```

- [x] **Step 4: Run all aggregate materializer tests and verify GREEN**

Run:

```sh
go test ./service/system -run Aggregate -count=1
```

Expected: all parser, resolver, materialization, safety, preflight, and rollback tests pass.

- [x] **Step 5: Commit candidate filtering**

```sh
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go
git commit -m "fix: exclude compose aliases from aggregate children"
```

### Task 4: Run automated and real deployment acceptance

**Files:**
- Verify: `server/service/system/aggregate_route_materializer.go`
- Verify: `server/service/system/aggregate_route_materializer_test.go`
- Verify: `/Users/conchi/workforce/python_workforce/ai-task-center/deploy/` generated runtime files only; do not commit them.

- [x] **Step 1: Format and inspect the complete change**

Run:

```sh
gofmt -w service/system/aggregate_route_materializer.go service/system/aggregate_route_materializer_test.go
git diff --check
git diff --stat
```

Expected: no formatting or whitespace diagnostics; only the two scoped Go files differ from the design commit.

- [x] **Step 2: Run scoped tests, race detector, and vet**

Run from `server/`:

```sh
go test ./service/system -count=1
go test -race ./service/system -count=1
go vet ./service/system
```

Expected: all three commands exit 0.

- [x] **Step 3: Reconfirm the repository-wide baseline separately**

Run:

```sh
go test ./... -count=1
```

Expected baseline: `service/system` passes while the unchanged repository-wide build still reports missing `frontend/dist/*` and missing `AwsS3`/`CloudflareR2` fields. If any new package failure appears, stop and fix or roll back the scoped change.

- [x] **Step 4: Commit formatting or verification-only adjustments if present**

If `gofmt` changed tracked files after Task 3:

```sh
git add server/service/system/aggregate_route_materializer.go server/service/system/aggregate_route_materializer_test.go
git commit -m "test: complete aggregate alias regression coverage"
```

If the worktree is already clean, do not create an empty commit.

- [ ] **Step 5: Integrate the verified branch and restart VibeDeploy**

Use the finishing-development-branch workflow to review and fast-forward the verified branch into `main`. Restart the local VibeDeploy backend from the integrated main worktree and confirm:

```sh
curl -fsS http://127.0.0.1:23638/health
```

Expected: HTTP success from the restarted backend. Preserve a recoverable reference to the pre-integration commit until live acceptance finishes.

- [ ] **Step 6: Prove fresh task-center child scripts are generated before execution**

Before deploying, move these generated directories to a temporary backup if they exist; do not delete them irrecoverably:

```text
/Users/conchi/workforce/python_workforce/ai-task-center/deploy/backend/python_worker/local_full
/Users/conchi/workforce/python_workforce/ai-task-center/deploy/backend/java_server/local_full
/Users/conchi/workforce/python_workforce/ai-task-center/deploy/frontend/web_react/local_full
```

Authenticate without printing the token, then call the UI's deployment stream:

```sh
VIBEDEPLOY_TOKEN="$(curl -fsS -X POST http://127.0.0.1:23638/base/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"123456"}' | jq -r '.data.token')"
curl -N --fail-with-body "http://127.0.0.1:23638/project/deployStream/128?env=205&token=${VIBEDEPLOY_TOKEN}"
unset VIBEDEPLOY_TOKEN
```

Expected stream order:

```text
🧩 解析聚合部署依赖...
✅ 发现 3 条子项目部署路线
✅ 聚合部署依赖脚本已全部落盘
```

The stream must end with the deployment completion event, not status 127. On failure, restore the backup directories and return main/backend to the pre-integration reference.

- [ ] **Step 7: Verify task-center runtime and shared-network invariants**

Run:

```sh
test -f /Users/conchi/workforce/python_workforce/ai-task-center/deploy/backend/python_worker/local_full/start.sh
test -f /Users/conchi/workforce/python_workforce/ai-task-center/deploy/backend/java_server/local_full/start.sh
test -f /Users/conchi/workforce/python_workforce/ai-task-center/deploy/frontend/web_react/local_full/start.sh
curl -fsS http://127.0.0.1:10052/api/health
curl -fsS http://127.0.0.1:10051/api/ai/execution-targets
curl -fsS http://127.0.0.1:6051/
docker inspect ai-task-center-server ai-task-center-web --format '{{.Name}} restart={{.HostConfig.RestartPolicy.Name}} networks={{range $name, $_ := .NetworkSettings.Networks}}{{$name}} {{end}}'
```

Expected: all three scripts exist; Python Worker, Java server, and React frontend health checks succeed; both containers report `restart=no` and only `vibedeploy-shared`.

- [ ] **Step 8: Verify canonical and non-aggregate regressions**

Run the canonical English-style service test again and the ordinary-route classification test:

```sh
go test ./service/system -run 'TestPrepareAggregateChildDeployScriptsMaterializesEnglishStyleDependencies|TestIsAggregateDeployRouteRequiresComposeLocalFrontendBackendMetadata' -count=1
```

Expected: both pass, proving the canonical form still materializes and ordinary `local_full` Compose routes still bypass aggregate parsing.
