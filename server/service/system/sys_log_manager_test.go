package system

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupLogManagerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	oldDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() {
		global.GVA_DB = oldDB
	})

	if err := db.AutoMigrate(&modelSystem.TbLogProject{}, &modelSystem.TbLogProjectRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestFileLogRouteAppearsAsServiceLogEntry(t *testing.T) {
	route := modelSystem.TbLogProjectRoute{
		RouteName:   "后端日志",
		BuildType:   "file_log",
		LogFilePath: ".devserver/backend.log",
	}

	if !isFileLogRoute(route) {
		t.Fatalf("expected route to be detected as file log")
	}

	if isDirectDockerComposeLogRoute(route) {
		t.Fatalf("file log route must not be treated as docker compose")
	}
}

func TestFileLogRoutesAreExcludedFromExecutableServiceGroup(t *testing.T) {
	routes := []modelSystem.TbLogProjectRoute{
		{
			RouteName:           "启动开发服务",
			LocalExecuteCommand: "./restart-dev.sh",
			Sort:                1,
		},
		{
			RouteName:   "后端日志",
			BuildType:   "file_log",
			LogFilePath: ".devserver/backend.log",
			Sort:        2,
		},
	}

	executable := executableLogRoutes(routes)
	if len(executable) != 1 {
		t.Fatalf("executable route count = %d, want 1", len(executable))
	}
	if executable[0].RouteName != "启动开发服务" {
		t.Fatalf("executable route = %q, want startup route", executable[0].RouteName)
	}
}

func TestBuildFileLogCommandUsesProjectRelativePath(t *testing.T) {
	project := modelSystem.TbLogProject{
		LocalProjectPath: "/work/app",
	}
	route := modelSystem.TbLogProjectRoute{
		RouteName:   "前端日志",
		BuildType:   "file_log",
		LogFilePath: ".devserver/frontend.log",
	}

	command, args, workDir, err := buildFileLogCommand(project, route)
	if err != nil {
		t.Fatalf("build file log command: %v", err)
	}
	if command != "tail" {
		t.Fatalf("command = %q, want tail", command)
	}
	if workDir != "/work/app" {
		t.Fatalf("workDir = %q, want /work/app", workDir)
	}
	wantLastArg := "/work/app/.devserver/frontend.log"
	if got := args[len(args)-1]; got != wantLastArg {
		t.Fatalf("last arg = %q, want %q", got, wantLastArg)
	}
}

func TestFileLogRouteRunningUsesSiblingPidFile(t *testing.T) {
	workDir := t.TempDir()
	pidPath := filepath.Join(workDir, ".devserver", "backend.pid")
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatalf("create pid dir: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	project := modelSystem.TbLogProject{LocalProjectPath: workDir}
	route := modelSystem.TbLogProjectRoute{
		RouteName:   "后端日志",
		BuildType:   "file_log",
		LogFilePath: ".devserver/backend.log",
	}

	if !fileLogRouteRunning(project, route) {
		t.Fatalf("expected file log route to be running when sibling pid file points to live process")
	}
}

func TestFileLogRouteRunningFalseWithoutPidFile(t *testing.T) {
	workDir := t.TempDir()
	project := modelSystem.TbLogProject{LocalProjectPath: workDir}
	route := modelSystem.TbLogProjectRoute{
		RouteName:   "后端日志",
		BuildType:   "file_log",
		LogFilePath: ".devserver/backend.log",
	}

	if fileLogRouteRunning(project, route) {
		t.Fatalf("expected file log route to be stopped without pid file")
	}
}

func TestListDockerServicesSkipsPlainStartupRoutes(t *testing.T) {
	db := setupLogManagerTestDB(t)
	project := modelSystem.TbLogProject{
		ProjectName:      "dashboard",
		LocalProjectPath: "/work/app",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	routes := []modelSystem.TbLogProjectRoute{
		{
			ProjectId:           int(project.ID),
			RouteName:           "启动开发服务",
			LocalExecuteCommand: "./restart-dev.sh",
			Sort:                1,
		},
		{
			ProjectId:   int(project.ID),
			RouteName:   "后端日志",
			BuildType:   "file_log",
			LogFilePath: ".devserver/backend.log",
			Sort:        2,
		},
	}
	if err := db.Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}

	entries, err := (&LogManagerService{}).ListDockerServices(project.ID, "service")
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	if entries[0].RouteName != "后端日志" || entries[0].Source != "file-log" {
		t.Fatalf("entry = %#v, want backend file log only", entries[0])
	}
}

func TestDockerScopeExcludesScriptAndFileLogRoutes(t *testing.T) {
	db := setupLogManagerTestDB(t)
	project := modelSystem.TbLogProject{
		ProjectName:      "dashboard",
		LocalProjectPath: "/work/app",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	routes := []modelSystem.TbLogProjectRoute{
		{
			ProjectId:           int(project.ID),
			RouteName:           "启动开发服务",
			LocalExecuteCommand: "./restart-dev.sh",
			BuildType:           "script",
		},
		{
			ProjectId:   int(project.ID),
			RouteName:   "后端日志",
			BuildType:   "file_log",
			LogFilePath: ".devserver/backend.log",
		},
	}
	if err := db.Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}

	entries, err := (&LogManagerService{}).ListDockerServices(project.ID, "docker")
	if err != nil {
		t.Fatalf("list docker services: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("docker entry count = %d, want 0: %#v", len(entries), entries)
	}
}

func TestGetLogProjectByIdAnnotatesRouteTypes(t *testing.T) {
	db := setupLogManagerTestDB(t)
	project := modelSystem.TbLogProject{
		ProjectName:      "dashboard",
		LocalProjectPath: "/work/app",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	routes := []modelSystem.TbLogProjectRoute{
		{
			ProjectId:           int(project.ID),
			RouteName:           "启动开发服务",
			LocalExecuteCommand: "./restart-dev.sh",
			BuildType:           "script",
		},
		{
			ProjectId:   int(project.ID),
			RouteName:   "后端日志",
			BuildType:   "file_log",
			LogFilePath: ".devserver/backend.log",
		},
		{
			ProjectId:           int(project.ID),
			RouteName:           "Docker 服务",
			LocalExecuteCommand: "docker compose up -d",
			BuildType:           "docker_compose_deploy",
		},
	}
	if err := db.Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}

	got, err := (&LogManagerService{}).GetLogProjectById(project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	types := map[string]string{}
	for _, route := range got.Routes {
		types[route.RouteName] = route.RouteType
	}
	if types["启动开发服务"] != "script" {
		t.Fatalf("startup route type = %q, want script", types["启动开发服务"])
	}
	if types["后端日志"] != "file_log" {
		t.Fatalf("file log route type = %q, want file_log", types["后端日志"])
	}
	if types["Docker 服务"] != "docker_compose" {
		t.Fatalf("docker route type = %q, want docker_compose", types["Docker 服务"])
	}
}
