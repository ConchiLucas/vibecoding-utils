package system

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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
