package system

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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
		name    string
		project modelSystem.TbProject
		route   modelSystem.TbProjectRoute
		want    bool
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

func TestParseAggregateChildScriptPathsPreservesOrderAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	aggregateDir := filepath.Join(root, "deploy", "compose", "full")
	content := `#!/bin/sh
# ignored
echo "prepare"
sh "$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"
bash "$ROOT_DIR/deploy/backend/word_agent/local_full/start.sh" --force
sh $ROOT_DIR/deploy/backend/rob_english_word/local_full/start.sh
sh "$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"
`

	got, err := parseAggregateChildScriptPaths(root, aggregateDir, content)
	if err != nil {
		t.Fatalf("parseAggregateChildScriptPaths() error = %v", err)
	}
	want := []string{
		filepath.Join(root, "deploy", "backend", "word_agent", "build_project"),
		filepath.Join(root, "deploy", "backend", "word_agent", "local_full"),
		filepath.Join(root, "deploy", "backend", "rob_english_word", "local_full"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}

func TestParseAggregateChildScriptPathsRejectsUnsafeOrUnsupportedReferences(t *testing.T) {
	root := t.TempDir()
	aggregateDir := filepath.Join(root, "deploy", "compose", "full")
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "unsupported command", content: `source "$ROOT_DIR/deploy/backend/api/local_full/start.sh"`, want: "第 1 行"},
		{name: "traversal", content: `sh "$ROOT_DIR/../outside/start.sh"`, want: "越过项目目录"},
		{name: "internal traversal", content: `sh "$ROOT_DIR/deploy/backend/../outside/start.sh"`, want: "越过项目目录"},
		{name: "absolute", content: "sh \"" + filepath.Join(root, "outside", "start.sh") + "\"", want: "绝对路径"},
		{name: "self reference", content: `sh "$ROOT_DIR/deploy/compose/full/start.sh"`, want: "引用自身"},
		{name: "empty", content: "#!/bin/sh\necho ready\n", want: "未引用任何"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseAggregateChildScriptPaths(root, aggregateDir, test.content)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want diagnostic containing %q", err, test.want)
			}
		})
	}
}

func TestParseAggregateChildScriptPathsRequiresAbsoluteRoot(t *testing.T) {
	_, err := parseAggregateChildScriptPaths("relative", "relative/deploy/full", `sh "$ROOT_DIR/deploy/api/start.sh"`)
	if err == nil || !strings.Contains(err.Error(), "绝对路径") {
		t.Fatalf("error = %v, want absolute-root diagnostic", err)
	}
}

func setupAggregateRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&modelSystem.TbProject{}, &modelSystem.TbProjectRoute{}, &modelSystem.TbProjectScript{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func createAggregateChildRoute(t *testing.T, db *gorm.DB, groupID uint, language, projectName, projectPath, scriptPath string, serverID int, withStart bool) (modelSystem.TbProject, modelSystem.TbProjectRoute) {
	t.Helper()
	project := modelSystem.TbProject{GroupId: groupID, ComputerLanguage: language, ProjectName: projectName, LocalProjectPath: projectPath}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	route := modelSystem.TbProjectRoute{ProjectId: int(project.ID), RouteKey: "local_full", RouteName: "本地全量部署", ServerId: serverID, LocalScriptPath: scriptPath}
	if err := db.Create(&route).Error; err != nil {
		t.Fatal(err)
	}
	if withStart {
		script := modelSystem.TbProjectScript{ProjectId: int(project.ID), RouteId: int(route.ID), ScriptType: 1, FileName: "start.sh", Content: "#!/bin/sh\n"}
		if err := db.Create(&script).Error; err != nil {
			t.Fatal(err)
		}
	}
	return project, route
}

func TestResolveAggregateChildRoutesPreservesReferencesAcrossLanguages(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	aggregate := modelSystem.TbProject{GroupId: 31, ComputerLanguage: deployProjectTypeDockerCompose, ProjectName: "english-compose", LocalProjectPath: root}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	aggregateRoute := modelSystem.TbProjectRoute{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", LocalScriptPath: filepath.Join(root, "deploy", "compose", "full")}
	if err := db.Create(&aggregateRoute).Error; err != nil {
		t.Fatal(err)
	}

	pythonPath := filepath.Join(root, "deploy", "backend", "word_agent", "build_project")
	_, pythonRoute := createAggregateChildRoute(t, db, aggregate.GroupId, "python", "word-agent", filepath.Join(root, "word-agent"), pythonPath, 0, true)
	javaProjectPath := filepath.Join(root, "rob-english-word")
	javaProject, javaRoute := createAggregateChildRoute(t, db, aggregate.GroupId, "java", "rob-english-word", javaProjectPath, "", 0, true)

	got, err := resolveAggregateChildRoutes(db, aggregate, aggregateRoute, []string{pythonPath, javaProjectPath})
	if err != nil {
		t.Fatalf("resolveAggregateChildRoutes() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("children = %d, want 2", len(got))
	}
	if got[0].Route.ID != pythonRoute.ID || got[0].ScriptPath != pythonPath {
		t.Fatalf("first child = %#v, want python route %d", got[0], pythonRoute.ID)
	}
	if got[1].Project.ID != javaProject.ID || got[1].Route.ID != javaRoute.ID || got[1].ScriptPath != javaProjectPath {
		t.Fatalf("second child = %#v, want fallback Java route", got[1])
	}
}

func TestResolveAggregateChildRoutesRejectsMissingAmbiguousAndInvalidRoutes(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *gorm.DB, modelSystem.TbProject, string)
		want  string
	}{
		{name: "missing", setup: func(*testing.T, *gorm.DB, modelSystem.TbProject, string) {}, want: "未找到"},
		{name: "remote only", setup: func(t *testing.T, db *gorm.DB, aggregate modelSystem.TbProject, ref string) {
			createAggregateChildRoute(t, db, aggregate.GroupId, "go", "remote", filepath.Dir(ref), ref, 9, true)
		}, want: "未找到"},
		{name: "missing start", setup: func(t *testing.T, db *gorm.DB, aggregate modelSystem.TbProject, ref string) {
			createAggregateChildRoute(t, db, aggregate.GroupId, "vue", "no-start", filepath.Dir(ref), ref, 0, false)
		}, want: "start.sh"},
		{name: "ambiguous", setup: func(t *testing.T, db *gorm.DB, aggregate modelSystem.TbProject, ref string) {
			createAggregateChildRoute(t, db, aggregate.GroupId, "react", "web-one", filepath.Join(filepath.Dir(ref), "one"), ref, 0, true)
			createAggregateChildRoute(t, db, aggregate.GroupId, "react", "web-two", filepath.Join(filepath.Dir(ref), "two"), ref, 0, true)
		}, want: "匹配到 2 条"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupAggregateRouteTestDB(t)
			root := t.TempDir()
			aggregate := modelSystem.TbProject{GroupId: 88, ComputerLanguage: deployProjectTypeDockerCompose, ProjectName: "aggregate", LocalProjectPath: root}
			if err := db.Create(&aggregate).Error; err != nil {
				t.Fatal(err)
			}
			route := modelSystem.TbProjectRoute{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", LocalScriptPath: filepath.Join(root, "deploy", "aggregate")}
			if err := db.Create(&route).Error; err != nil {
				t.Fatal(err)
			}
			ref := filepath.Join(root, "deploy", "child", "local_full")
			test.setup(t, db, aggregate, ref)

			_, err := resolveAggregateChildRoutes(db, aggregate, route, []string{ref})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want diagnostic containing %q", err, test.want)
			}
		})
	}
}

func TestLoadSingleLocalStartScriptRejectsDuplicateEntries(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	project, route := createAggregateChildRoute(t, db, 3, "go", "api", t.TempDir(), t.TempDir(), 0, true)
	duplicate := modelSystem.TbProjectScript{ProjectId: int(project.ID), RouteId: int(route.ID), ScriptType: 0, FileName: "start.sh", Content: "duplicate"}
	if err := db.Create(&duplicate).Error; err != nil {
		t.Fatal(err)
	}

	_, err := loadSingleLocalStartScript(db, project.ID, route.ID)
	if err == nil || !strings.Contains(err.Error(), "2") {
		t.Fatalf("error = %v, want duplicate-count diagnostic", err)
	}
}

func TestLoadSingleLocalStartScriptRejectsEmptyEntry(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	project, route := createAggregateChildRoute(t, db, 3, "python", "empty", t.TempDir(), t.TempDir(), 0, false)
	entry := modelSystem.TbProjectScript{ProjectId: int(project.ID), RouteId: int(route.ID), ScriptType: 1, FileName: "start.sh", Content: "  \n"}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}

	_, err := loadSingleLocalStartScript(db, project.ID, route.ID)
	if err == nil || !strings.Contains(err.Error(), "内容为空") {
		t.Fatalf("error = %v, want empty-entry diagnostic", err)
	}
}

func TestPrepareAggregateChildDeployScriptsMaterializesEnglishStyleDependencies(t *testing.T) {
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
	aggregatePath := filepath.Join(root, "deploy", "compose", "full")
	aggregate := modelSystem.TbProject{GroupId: 31, ComputerLanguage: deployProjectTypeDockerCompose, ProjectName: "rob_english_word_workforce-compose", LocalProjectPath: root}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	aggregateRoute := modelSystem.TbProjectRoute{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", RouteName: "前后端全量部署", LocalScriptPath: aggregatePath}
	if err := db.Create(&aggregateRoute).Error; err != nil {
		t.Fatal(err)
	}

	type dependency struct {
		language string
		name     string
		path     string
	}
	dependencies := []dependency{
		{language: "python", name: "word-agent-build", path: filepath.Join(root, "deploy", "backend", "word_agent", "build_project")},
		{language: "python", name: "word-agent", path: filepath.Join(root, "deploy", "backend", "word_agent", "local_full")},
		{language: "java", name: "rob-english-word", path: filepath.Join(root, "deploy", "backend", "rob_english_word", "local_full")},
		{language: "go", name: "word-select-dashboard", path: filepath.Join(root, "deploy", "backend", "word_select_dashboard", "local_full")},
		{language: "vue", name: "rob-english-word-front-web", path: filepath.Join(root, "deploy", "frontend", "rob_english_word_front", "local_full")},
		{language: "react", name: "rob-english-word-cloze-web", path: filepath.Join(root, "deploy", "frontend", "rob_english_word_cloze", "local_full")},
		{language: "react", name: "word-select-dashboard-web-react", path: filepath.Join(root, "deploy", "frontend", "word_select_dashboard", "local_full")},
	}

	var aggregateLines []string
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
		aggregateLines = append(aggregateLines, `sh "$ROOT_DIR/`+filepath.ToSlash(reference)+`"`)
	}
	unrelatedPath := filepath.Join(root, "deploy", "backend", "unrelated", "local_incremental")
	unrelatedProject, unrelatedRoute := createAggregateChildRoute(t, db, aggregate.GroupId, "go", "unrelated", filepath.Join(root, "unrelated"), unrelatedPath, 0, false)
	unrelatedScript := modelSystem.TbProjectScript{ProjectId: int(unrelatedProject.ID), RouteId: int(unrelatedRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "unrelated"}
	if err := db.Create(&unrelatedScript).Error; err != nil {
		t.Fatal(err)
	}
	aggregateStart := modelSystem.TbProjectScript{ProjectId: int(aggregate.ID), RouteId: int(aggregateRoute.ID), ScriptType: 1, FileName: "start.sh", Content: strings.Join(aggregateLines, "\n") + "\n"}
	if err := db.Create(&aggregateStart).Error; err != nil {
		t.Fatal(err)
	}

	logCh := make(chan string, 32)
	if err := DeployServiceApp.prepareAggregateChildDeployScripts(aggregate, aggregateRoute, logCh); err != nil {
		t.Fatalf("prepareAggregateChildDeployScripts() error = %v", err)
	}
	close(logCh)
	var logs []string
	for line := range logCh {
		logs = append(logs, line)
	}
	joinedLogs := strings.Join(logs, "\n")
	if !strings.Contains(joinedLogs, "发现 7 条") || !strings.Contains(joinedLogs, "已全部落盘") {
		t.Fatalf("logs = %q, want dependency count and completion", joinedLogs)
	}
	if _, err := os.Stat(filepath.Join(aggregatePath, "start.sh")); err != nil {
		t.Fatalf("aggregate start.sh was not materialized: %v", err)
	}
	for _, dependency := range dependencies {
		if _, err := os.Stat(filepath.Join(dependency.path, "start.sh")); err != nil {
			t.Fatalf("dependency %s was not materialized: %v", dependency.name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(unrelatedPath, "start.sh")); !os.IsNotExist(err) {
		t.Fatalf("unrelated route was materialized: %v", err)
	}
}

func TestPrepareAggregateChildDeployScriptsPreflightsBeforeAnyWrite(t *testing.T) {
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
	aggregatePath := filepath.Join(root, "deploy", "aggregate")
	aggregate := modelSystem.TbProject{GroupId: 44, ComputerLanguage: deployProjectTypeDockerCompose, ProjectName: "aggregate", LocalProjectPath: root}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	aggregateRoute := modelSystem.TbProjectRoute{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", LocalScriptPath: aggregatePath}
	if err := db.Create(&aggregateRoute).Error; err != nil {
		t.Fatal(err)
	}
	validPath := filepath.Join(root, "deploy", "valid")
	validProject, validRoute := createAggregateChildRoute(t, db, aggregate.GroupId, "go", "valid", filepath.Join(root, "valid"), validPath, 0, false)
	validStart := modelSystem.TbProjectScript{ProjectId: int(validProject.ID), RouteId: int(validRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "valid"}
	if err := db.Create(&validStart).Error; err != nil {
		t.Fatal(err)
	}
	aggregateStart := modelSystem.TbProjectScript{ProjectId: int(aggregate.ID), RouteId: int(aggregateRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "sh \"$ROOT_DIR/deploy/valid/start.sh\"\nsh \"$ROOT_DIR/deploy/missing/start.sh\"\n"}
	if err := db.Create(&aggregateStart).Error; err != nil {
		t.Fatal(err)
	}

	err := DeployServiceApp.prepareAggregateChildDeployScripts(aggregate, aggregateRoute, nil)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %v, want missing dependency", err)
	}
	for _, target := range []string{filepath.Join(aggregatePath, "start.sh"), filepath.Join(validPath, "start.sh")} {
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("preflight wrote %s: %v", target, err)
		}
	}
}
