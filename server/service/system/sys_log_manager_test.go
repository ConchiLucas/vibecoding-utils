package system

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
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

	if err := db.AutoMigrate(&modelSystem.TbProject{}, &modelSystem.TbInterfaceProject{}, &modelSystem.TbLogProjectGroup{}, &modelSystem.TbLogProject{}, &modelSystem.TbLogProjectRoute{}); err != nil {
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

func TestLogRouteRestartCommandsRunStopThenStart(t *testing.T) {
	route := modelSystem.TbLogProjectRoute{
		LocalExecuteCommand: "./start.sh",
		LocalStartCommand:   "npm run dev",
		LocalStopCommand:    "./stop.sh",
	}

	commands, err := logRouteActionCommands(route, "restart", "/work/app")
	if err != nil {
		t.Fatalf("build restart commands: %v", err)
	}
	want := []string{"./stop.sh", "./start.sh", "npm run dev"}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands = %#v, want %#v", commands, want)
	}
}

func TestParseLaunchdPlistReadsProgramArguments(t *testing.T) {
	plistPath := filepath.Join(t.TempDir(), "service.plist")
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>Label</key><string>local.example.service</string>
  <key>WorkingDirectory</key><string>/work/service</string>
  <key>StandardOutPath</key><string>/work/.service-runtime/logs/service.log</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/bin/env</string>
    <string>npm</string>
    <string>run</string>
    <string>dev</string>
  </array>
</dict>
</plist>`
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	info, err := parseLaunchdPlist(plistPath)
	if err != nil {
		t.Fatalf("parse plist: %v", err)
	}
	if info.Label != "local.example.service" || info.WorkingDirectory != "/work/service" {
		t.Fatalf("basic plist fields = %#v", info)
	}
	wantArgs := []string{"/usr/bin/env", "npm", "run", "dev"}
	if !reflect.DeepEqual(info.ProgramArguments, wantArgs) {
		t.Fatalf("program arguments = %#v, want %#v", info.ProgramArguments, wantArgs)
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

func TestFileLogRouteRunningUsesServiceRuntimePidFile(t *testing.T) {
	workDir := t.TempDir()
	logPath := filepath.Join(workDir, ".service-runtime", "logs", "rob_english_word_back.log")
	pidPath := filepath.Join(workDir, ".service-runtime", "pids", "rob_english_word_back.pid")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create log dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatalf("create pid dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("started"), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	project := modelSystem.TbLogProject{LocalProjectPath: filepath.Join(workDir, "rob_english_word_back")}
	route := modelSystem.TbLogProjectRoute{
		RouteKey:    "rob_english_word_back",
		RouteName:   "rob_english_word_back 启动日志",
		BuildType:   "file_log",
		LogFilePath: logPath,
	}

	if !fileLogRouteRunning(project, route) {
		t.Fatalf("expected file log route to be running when service-runtime pid file points to live process")
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

func TestDiscoverStartupLogEntriesFromLaunchdPlist(t *testing.T) {
	workspace := t.TempDir()
	workDir := filepath.Join(workspace, "rob_english_word_back")
	logPath := filepath.Join(workspace, ".service-runtime", "logs", "rob_english_word_back.log")
	plistPath := filepath.Join(workspace, ".service-runtime", "launchd", "rob_english_word_back.plist")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("create work dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "pom.xml"), []byte("<project />"), 0o644); err != nil {
		t.Fatalf("write pom: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create log dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("started"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		t.Fatalf("create launchd dir: %v", err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>Label</key><string>local.rob_english_word_workforce.rob_english_word_back</string>
  <key>WorkingDirectory</key><string>` + workDir + `</string>
  <key>StandardOutPath</key><string>` + logPath + `</string>
</dict>
</plist>`
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	entries, err := discoverStartupLogEntries(workspace)
	if err != nil {
		t.Fatalf("discover startup logs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1: %#v", len(entries), entries)
	}
	entry := entries[0]
	if entry.ServiceName != "rob_english_word_back" {
		t.Fatalf("service name = %q, want rob_english_word_back", entry.ServiceName)
	}
	if entry.WorkDir != workDir {
		t.Fatalf("work dir = %q, want %q", entry.WorkDir, workDir)
	}
	if entry.LogFilePath != logPath {
		t.Fatalf("log path = %q, want %q", entry.LogFilePath, logPath)
	}
	if entry.Language != "java" {
		t.Fatalf("language = %q, want java", entry.Language)
	}
}

func TestSyncDiscoveredStartupLogsCreatesFileLogProject(t *testing.T) {
	db := setupLogManagerTestDB(t)
	workspace := t.TempDir()
	workDir := filepath.Join(workspace, "rob_english_word_front")
	logPath := filepath.Join(workspace, ".service-runtime", "logs", "rob_english_word_front.log")
	plistPath := filepath.Join(workspace, ".service-runtime", "launchd", "rob_english_word_front.plist")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("create work dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "package.json"), []byte(`{"scripts":{"dev":"vite"}}`), 0o644); err != nil {
		t.Fatalf("write package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "restart_all_services.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write launcher script: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create log dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("vite ready"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		t.Fatalf("create launchd dir: %v", err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>Label</key><string>local.rob_english_word_workforce.rob_english_word_front</string>
  <key>WorkingDirectory</key><string>` + workDir + `</string>
  <key>StandardOutPath</key><string>` + logPath + `</string>
</dict>
</plist>`
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}
	legacyProject := modelSystem.TbLogProject{
		ProjectConfigId:  42,
		ComputerLanguage: "react",
		ProjectName:      "rob_english_word_front",
		Description:      startupLogDescription,
		LocalProjectPath: workDir,
	}
	if err := db.Create(&legacyProject).Error; err != nil {
		t.Fatalf("create legacy project: %v", err)
	}
	legacyRoute := modelSystem.TbLogProjectRoute{
		ProjectId:   int(legacyProject.ID),
		RouteKey:    "rob_english_word_front",
		RouteName:   "rob_english_word_front 启动日志",
		BuildType:   "file_log",
		LogFilePath: logPath,
	}
	if err := db.Create(&legacyRoute).Error; err != nil {
		t.Fatalf("create legacy route: %v", err)
	}

	req := request.TbLogProjectSearch{
		ProjectConfigId: 42,
	}
	if err := (&LogManagerService{}).syncDiscoveredStartupLogsFromWorkspaces(req, []string{workspace}); err != nil {
		t.Fatalf("sync startup logs: %v", err)
	}

	var project modelSystem.TbLogProject
	if err := db.Preload("Routes").Where("project_config_id = ? AND local_project_path = ?", uint(42), workspace).First(&project).Error; err != nil {
		t.Fatalf("find synced project: %v", err)
	}
	if project.ProjectName != filepath.Base(workspace) {
		t.Fatalf("project name = %q, want %q", project.ProjectName, filepath.Base(workspace))
	}
	if project.ComputerLanguage != "service" {
		t.Fatalf("language = %q, want service", project.ComputerLanguage)
	}
	if len(project.Routes) != 2 {
		t.Fatalf("route count = %d, want launcher + file log routes", len(project.Routes))
	}
	routesByKey := map[string]modelSystem.TbLogProjectRoute{}
	for _, route := range project.Routes {
		routesByKey[route.RouteKey] = route
	}
	launcher := routesByKey[startupLogLauncherKey]
	if launcher.BuildType != "script" || launcher.LocalExecuteCommand != "./restart_all_services.sh start" || launcher.LocalStopCommand != "./restart_all_services.sh stop" {
		t.Fatalf("launcher route = %#v, want synced restart_all_services route", launcher)
	}
	fileLog := routesByKey["rob_english_word_front"]
	if fileLog.BuildType != "file_log" || fileLog.LogFilePath != logPath || fileLog.LocalProjectPath != workDir {
		t.Fatalf("file log route = %#v, want synced file_log route", fileLog)
	}

	var projectCount int64
	if err := db.Model(&modelSystem.TbLogProject{}).
		Where("project_config_id = ? AND description = ?", uint(42), startupLogDescription).
		Count(&projectCount).Error; err != nil {
		t.Fatalf("count synced projects: %v", err)
	}
	if projectCount != 1 {
		t.Fatalf("synced project count = %d, want only aggregate workspace project", projectCount)
	}
}

func TestSyncDiscoveredStartupLogsUsesMyProjectConfig(t *testing.T) {
	db := setupLogManagerTestDB(t)
	t.Setenv(startupLogProjectConfigEnv, "")

	targetProject := modelSystem.TbInterfaceProject{
		ProjectName: "我的项目",
		ProjectDesc: "我的项目",
	}
	if err := db.Create(&targetProject).Error; err != nil {
		t.Fatalf("create target project config: %v", err)
	}

	workspace := t.TempDir()
	logPath := filepath.Join(workspace, ".service-runtime", "logs", "word_agent.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create log dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("agent ready"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	wrongProject := modelSystem.TbLogProject{
		ProjectConfigId:   99,
		ProjectConfigName: "攀枝花",
		ComputerLanguage:  "service",
		ProjectName:       filepath.Base(workspace),
		Description:       startupLogDescription,
		LocalProjectPath:  workspace,
	}
	if err := db.Create(&wrongProject).Error; err != nil {
		t.Fatalf("create wrong project: %v", err)
	}
	wrongRoute := modelSystem.TbLogProjectRoute{
		ProjectId:   int(wrongProject.ID),
		RouteKey:    "word_agent",
		RouteName:   "word_agent 启动日志",
		BuildType:   "file_log",
		LogFilePath: logPath,
	}
	if err := db.Create(&wrongRoute).Error; err != nil {
		t.Fatalf("create wrong route: %v", err)
	}

	req := request.TbLogProjectSearch{
		ProjectConfigId:   99,
		ProjectConfigName: "攀枝花",
	}
	if err := (&LogManagerService{}).syncDiscoveredStartupLogsFromWorkspaces(req, []string{workspace}); err != nil {
		t.Fatalf("sync startup logs: %v", err)
	}

	var project modelSystem.TbLogProject
	if err := db.Preload("Routes").Where("local_project_path = ?", workspace).First(&project).Error; err != nil {
		t.Fatalf("find synced project: %v", err)
	}
	if project.ProjectConfigId != targetProject.ID || project.ProjectConfigName != "我的项目" {
		t.Fatalf("project config = (%d, %q), want (%d, 我的项目)", project.ProjectConfigId, project.ProjectConfigName, targetProject.ID)
	}
	if len(project.Routes) != 1 || project.Routes[0].RouteKey != "word_agent" {
		t.Fatalf("routes = %#v, want only word_agent file log route", project.Routes)
	}

	var wrongCount int64
	if err := db.Model(&modelSystem.TbLogProject{}).
		Where("project_config_id = ? AND project_config_name = ? AND description = ?", uint(99), "攀枝花", startupLogDescription).
		Count(&wrongCount).Error; err != nil {
		t.Fatalf("count wrong projects: %v", err)
	}
	if wrongCount != 0 {
		t.Fatalf("wrong project count = %d, want 0", wrongCount)
	}
}

func TestListDockerServicesSkipsPlainStartupRoutes(t *testing.T) {
	db := setupLogManagerTestDB(t)
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "docker-compose.yml"), []byte("services:\n  backend:\n    image: busybox\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	project := modelSystem.TbLogProject{
		ProjectName:      "dashboard",
		LocalProjectPath: workDir,
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

func TestListDockerServicesIncludesComposeServicesWithoutRoutes(t *testing.T) {
	db := setupLogManagerTestDB(t)
	workDir := t.TempDir()
	compose := `services:
  backend:
    image: busybox
  worker:
    image: busybox
  web:
    image: busybox
`
	if err := os.WriteFile(filepath.Join(workDir, "docker-compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	project := modelSystem.TbLogProject{
		ProjectName:      "dashboard",
		LocalProjectPath: workDir,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	routes := []modelSystem.TbLogProjectRoute{
		{
			ProjectId:           int(project.ID),
			RouteKey:            "backend",
			RouteName:           "backend",
			LocalProjectPath:    workDir,
			LocalExecuteCommand: "docker compose up -d backend",
			BuildType:           "docker_compose_deploy",
			Sort:                1,
		},
		{
			ProjectId:           int(project.ID),
			RouteKey:            "worker",
			RouteName:           "worker",
			LocalProjectPath:    workDir,
			LocalExecuteCommand: "docker compose up -d worker",
			BuildType:           "docker_compose_deploy",
			Sort:                2,
		},
	}
	if err := db.Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}

	entries, err := (&LogManagerService{}).ListDockerServices(project.ID, "docker")
	if err != nil {
		t.Fatalf("list docker services: %v", err)
	}
	names := map[string]DockerServiceSummary{}
	for _, entry := range entries {
		names[entry.ServiceName] = entry
	}
	for _, name := range []string{"backend", "worker", "web"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("missing compose service %q in entries: %#v", name, entries)
		}
	}
	if len(entries) != 3 {
		t.Fatalf("entry count = %d, want 3: %#v", len(entries), entries)
	}
	if names["web"].RouteName != "Docker Compose 自动发现" {
		t.Fatalf("auto discovered route name = %q, want Docker Compose 自动发现", names["web"].RouteName)
	}
}

func TestComposeServicesFromFilesSupportsComposeYml(t *testing.T) {
	workDir := t.TempDir()
	compose := `services:
  api:
    image: busybox
  ui:
    image: busybox
`
	if err := os.WriteFile(filepath.Join(workDir, "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	got := composeServicesFromFiles(workDir)
	want := []string{"api", "ui"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compose services = %#v, want %#v", got, want)
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
