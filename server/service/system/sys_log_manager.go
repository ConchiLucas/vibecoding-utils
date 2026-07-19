package system

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type LogManagerService struct{}

const (
	logRouteTypeScript        = "script"
	logRouteTypeFileLog       = "file_log"
	logRouteTypeDockerCompose = "docker_compose"

	startupLogWorkspaceEnv             = "VIBEDEPLOY_STARTUP_LOG_WORKSPACES"
	startupLogProjectConfigEnv         = "VIBEDEPLOY_STARTUP_LOG_PROJECT_CONFIG"
	startupLogDefaultProjectConfigName = "我的项目"
	startupLogDescription              = "自动发现的本地启动日志"
	startupLogLauncherKey              = "__startup_all__"
)

type DockerServiceSummary struct {
	ProjectID   uint   `json:"projectId"`
	RouteID     uint   `json:"routeId"`
	RouteName   string `json:"routeName"`
	ServiceName string `json:"serviceName"`
	WorkDir     string `json:"workDir"`
	Source      string `json:"source"`
	LogFilePath string `json:"logFilePath"`
	RouteType   string `json:"routeType"`
	Running     bool   `json:"running"`
}

type startupLogEntry struct {
	ServiceName string
	WorkDir     string
	LogFilePath string
	Language    string
}

type launchdPlistInfo struct {
	Label             string
	WorkingDirectory  string
	StandardOutPath   string
	StandardErrorPath string
	ProgramArguments  []string
}

type composeRouteGroup struct {
	WorkDir string
	Route   system.TbLogProjectRoute
}

type composeServiceListState struct {
	WorkDir   string
	Route     system.TbLogProjectRoute
	Services  []string
	Emitted   map[string]struct{}
	RouteOnly bool
}

func (s *LogManagerService) GetLogProjectPage(req request.TbLogProjectSearch) (list []system.TbLogProject, total int64, err error) {
	if err = s.syncDiscoveredStartupLogs(req); err != nil {
		return
	}

	db := global.GVA_DB.Model(&system.TbLogProject{})
	if req.ProjectName != "" {
		db = db.Where("project_name LIKE ?", "%"+req.ProjectName+"%")
	}
	if req.ProjectConfigId != 0 {
		db = db.Where("project_config_id = ?", req.ProjectConfigId)
	} else if req.ProjectConfigName != "" {
		db = db.Where("project_config_name = ?", req.ProjectConfigName)
	}
	if req.ComputerLanguage != "" {
		db = db.Where("computer_language = ?", req.ComputerLanguage)
	}
	if req.GroupId != 0 {
		db = db.Where("group_id = ?", req.GroupId)
	}
	if req.UserId != 0 {
		db = db.Where("user_id = ?", req.UserId)
	}

	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Scopes(req.Paginate()).
		Order("id desc").
		Preload("Routes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort asc, id asc")
		}).
		Find(&list).Error
	if err == nil {
		for i := range list {
			annotateLogProjectRoutes(&list[i])
		}
	}
	return
}

func (s *LogManagerService) GetLogProjectGroups() ([]system.TbLogProjectGroup, error) {
	var list []system.TbLogProjectGroup
	err := global.GVA_DB.Order("sort asc, id asc").Find(&list).Error
	return list, err
}

func (s *LogManagerService) SaveOrUpdateLogProject(project system.TbLogProject) error {
	if strings.TrimSpace(project.ProjectName) == "" {
		return fmt.Errorf("项目名称不能为空")
	}
	if project.ID != 0 {
		return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
			var existing system.TbLogProject
			if err := tx.Where("id = ?", project.ID).First(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&system.TbLogProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
				"group_id":            project.GroupId,
				"project_config_id":   project.ProjectConfigId,
				"project_config_name": project.ProjectConfigName,
				"computer_language":   project.ComputerLanguage,
				"project_name":        project.ProjectName,
				"description":         project.Description,
				"local_project_path":  strings.TrimSpace(project.LocalProjectPath),
				"user_id":             project.UserId,
			}).Error; err != nil {
				return err
			}
			return migrateLogProjectRoutePaths(tx, project.ID, existing.LocalProjectPath, project.LocalProjectPath)
		})
	}
	project.LocalProjectPath = strings.TrimSpace(project.LocalProjectPath)
	return global.GVA_DB.Create(&project).Error
}

func migrateLogProjectRoutePaths(tx *gorm.DB, projectID uint, oldRoot string, newRoot string) error {
	oldRoot = strings.TrimSpace(oldRoot)
	newRoot = strings.TrimSpace(newRoot)
	if oldRoot == "" || newRoot == "" || filepath.Clean(oldRoot) == filepath.Clean(newRoot) {
		return nil
	}

	var routes []system.TbLogProjectRoute
	if err := tx.Where("project_id = ?", projectID).Find(&routes).Error; err != nil {
		return err
	}
	for _, route := range routes {
		updates := map[string]interface{}{}
		if migrated, ok := rebaseLogProjectPath(route.LocalProjectPath, oldRoot, newRoot); ok {
			updates["local_project_path"] = migrated
		}
		if migrated, ok := rebaseLogProjectPath(route.LogFilePath, oldRoot, newRoot); ok {
			updates["log_file_path"] = migrated
		}
		if len(updates) == 0 {
			continue
		}
		if err := tx.Model(&system.TbLogProjectRoute{}).Where("id = ?", route.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func rebaseLogProjectPath(path string, oldRoot string, newRoot string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return path, false
	}
	oldRoot = filepath.Clean(oldRoot)
	path = filepath.Clean(path)
	if !isPathInside(oldRoot, path) {
		return path, false
	}
	rel, err := filepath.Rel(oldRoot, path)
	if err != nil {
		return path, false
	}
	return filepath.Join(filepath.Clean(newRoot), rel), true
}

func (s *LogManagerService) DeleteLogProject(ids []int) error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if err := tx.Where("project_id = ?", id).Unscoped().Delete(&system.TbLogProjectRoute{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", id).Unscoped().Delete(&system.TbLogProject{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *LogManagerService) SaveOrUpdateLogRoute(route system.TbLogProjectRoute) error {
	if route.ProjectId == 0 {
		return fmt.Errorf("项目ID不能为空")
	}
	if strings.TrimSpace(route.RouteName) == "" {
		return fmt.Errorf("路线名称不能为空")
	}
	if route.ID != 0 {
		return global.GVA_DB.Model(&system.TbLogProjectRoute{}).Where("id = ?", route.ID).Updates(map[string]interface{}{
			"project_id":            route.ProjectId,
			"route_key":             route.RouteKey,
			"route_name":            route.RouteName,
			"local_project_path":    route.LocalProjectPath,
			"local_execute_command": route.LocalExecuteCommand,
			"local_start_command":   route.LocalStartCommand,
			"local_stop_command":    route.LocalStopCommand,
			"log_file_path":         route.LogFilePath,
			"build_type":            route.BuildType,
			"docker_compose_deploy": route.DockerComposeDeploy,
			"color":                 route.Color,
			"icon":                  route.Icon,
			"sort":                  route.Sort,
		}).Error
	}
	return global.GVA_DB.Create(&route).Error
}

func (s *LogManagerService) DeleteLogRoute(id int) error {
	return global.GVA_DB.Where("id = ?", id).Unscoped().Delete(&system.TbLogProjectRoute{}).Error
}

func (s *LogManagerService) syncDiscoveredStartupLogs(req request.TbLogProjectSearch) error {
	return s.syncDiscoveredStartupLogsFromWorkspaces(req, startupLogWorkspaces())
}

func (s *LogManagerService) syncDiscoveredStartupLogsFromWorkspaces(req request.TbLogProjectSearch, workspaces []string) error {
	if global.GVA_DB == nil {
		return nil
	}

	for _, workspace := range workspaces {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			continue
		}
		if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
			continue
		}

		entries, err := discoverStartupLogEntries(workspace)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			continue
		}

		groupID, err := ensureStartupLogProjectGroup(filepath.Base(workspace))
		if err != nil {
			return err
		}

		projectConfigID, projectConfigName, err := resolveStartupLogProjectConfig(req, filepath.Base(workspace))
		if err != nil {
			return err
		}

		if err := upsertStartupLogWorkspaceProject(groupID, projectConfigID, projectConfigName, workspace, entries); err != nil {
			return err
		}
	}
	return nil
}

func (s *LogManagerService) GetLogProjectById(projectId uint) (system.TbLogProject, error) {
	var project system.TbLogProject
	err := global.GVA_DB.
		Preload("Routes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort asc, id asc")
		}).
		Where("id = ?", projectId).
		First(&project).Error
	if err == nil {
		annotateLogProjectRoutes(&project)
	}
	return project, err
}

func (s *LogManagerService) StreamProjectServiceGroup(projectId uint, action string, scope string, logCh chan string) error {
	project, err := s.GetLogProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取日志项目信息失败: %w", err)
	}

	normalizedAction, err := normalizeLogAction(action)
	if err != nil {
		return err
	}

	routes := executableLogRoutes(scopedLogRoutes(localLogRoutes(project.Routes), scope))
	if len(routes) == 0 {
		return fmt.Errorf("当前日志项目没有可执行的%s路线", logRouteScopeLabel(scope))
	}

	if normalizedAction == "restart" && strings.EqualFold(strings.TrimSpace(scope), "docker") {
		return streamDockerComposeServiceGroupRestart(project, routes, logCh)
	}

	sendLog(logCh, fmt.Sprintf("📋 日志项目: %s", project.ProjectName))
	sendLog(logCh, fmt.Sprintf("📦 本次%s %d 个%s", groupActionLabel(normalizedAction), len(routes), logRouteScopeLabel(scope)))

	var failed []string
	for index, route := range routes {
		sendLog(logCh, "")
		sendLog(logCh, fmt.Sprintf("▶️ [%d/%d] %s", index+1, len(routes), route.RouteName))
		if err := executeLogRoute(context.Background(), project, route, normalizedAction, logCh); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", route.RouteName, err))
			sendLog(logCh, fmt.Sprintf("❌ %s失败: %v", route.RouteName, err))
			continue
		}
		sendLog(logCh, fmt.Sprintf("✅ %s完成", route.RouteName))
	}

	if len(failed) > 0 {
		return fmt.Errorf("%d 个服务执行失败: %s", len(failed), strings.Join(failed, "；"))
	}
	sendLog(logCh, fmt.Sprintf("✅ 服务组%s完成", groupActionLabel(normalizedAction)))
	return nil
}

func (s *LogManagerService) StreamProjectRoute(projectId uint, targetEnv string, action string, logCh chan string) error {
	project, err := s.GetLogProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取日志项目信息失败: %w", err)
	}
	route, err := findLogRoute(project.Routes, targetEnv)
	if err != nil {
		return err
	}
	normalizedAction, err := normalizeLogAction(action)
	if err != nil {
		return err
	}
	sendLog(logCh, fmt.Sprintf("📋 日志项目: %s", project.ProjectName))
	sendLog(logCh, fmt.Sprintf("📋 服务路线: %s", route.RouteName))
	return executeLogRoute(context.Background(), project, route, normalizedAction, logCh)
}

func (s *LogManagerService) StreamDockerComposeServiceRestart(projectId uint, targetEnv string, serviceName string, logCh chan string) error {
	project, err := s.GetLogProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取日志项目信息失败: %w", err)
	}
	route, err := findLogRoute(project.Routes, targetEnv)
	if err != nil {
		return err
	}
	workDir := resolveLogRouteWorkDir(project, route)
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("项目本地路径为空，无法定位 Docker Compose 服务")
	}
	if !isLogDockerComposeRoute(route, workDir) {
		return fmt.Errorf("当前路线不是 Docker Compose 服务路线")
	}

	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return fmt.Errorf("Docker Compose 服务名为空")
	}

	sendLog(logCh, fmt.Sprintf("📋 日志项目: %s", project.ProjectName))
	sendLog(logCh, fmt.Sprintf("📋 服务路线: %s", route.RouteName))
	sendLog(logCh, fmt.Sprintf("🐳 Compose 服务: %s", serviceName))
	sendLog(logCh, fmt.Sprintf("🏠 工作目录: %s", workDir))
	sendLog(logCh, fmt.Sprintf("🔨 执行命令: docker compose restart %s", serviceName))
	if err := runCommandArgsWithLog(context.Background(), workDir, "docker", []string{"compose", "restart", serviceName}, logCh); err != nil {
		return fmt.Errorf("Docker Compose 服务重启失败: %w", err)
	}
	sendLog(logCh, fmt.Sprintf("✅ Docker Compose 服务 %s 重启完成", serviceName))
	return nil
}

func (s *LogManagerService) StreamDockerLogs(ctx context.Context, projectId uint, targetEnv string, serviceName string, logCh chan string) error {
	project, err := s.GetLogProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取日志项目信息失败: %w", err)
	}
	route, err := findLogRoute(project.Routes, targetEnv)
	if err != nil {
		return err
	}

	localProjectPath := resolveLogRouteWorkDir(project, route)
	if isFileLogRoute(route) {
		command, args, workDir, err := buildFileLogCommand(project, route)
		if err != nil {
			return err
		}
		sendLog(logCh, fmt.Sprintf("📋 日志项目: %s", project.ProjectName))
		sendLog(logCh, fmt.Sprintf("📋 服务路线: %s", route.RouteName))
		sendLog(logCh, fmt.Sprintf("📄 日志文件: %s", args[len(args)-1]))
		sendLog(logCh, fmt.Sprintf("📄 日志命令: %s %s", command, strings.Join(args, " ")))
		return streamCommandLines(ctx, workDir, command, args, logCh)
	}
	if localProjectPath == "" {
		return fmt.Errorf("项目本地路径为空，无法定位 Docker 日志")
	}

	sendLog(logCh, fmt.Sprintf("📋 日志项目: %s", project.ProjectName))
	sendLog(logCh, fmt.Sprintf("📋 服务路线: %s", route.RouteName))

	command, args, workDir := buildLogDockerLogCommand(project, route, localProjectPath, serviceName)
	sendLog(logCh, fmt.Sprintf("🐳 日志命令: %s %s", command, strings.Join(args, " ")))
	return streamCommandLines(ctx, workDir, command, args, logCh)
}

func (s *LogManagerService) ListDockerServices(projectId uint, scope string) ([]DockerServiceSummary, error) {
	project, err := s.GetLogProjectById(projectId)
	if err != nil {
		return nil, fmt.Errorf("获取日志项目信息失败: %w", err)
	}

	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	routes := scopedLogRoutes(localLogRoutes(project.Routes), scope)
	result := make([]DockerServiceSummary, 0)
	seen := map[string]struct{}{}
	composeStates := map[string]*composeServiceListState{}
	composeStateOrder := make([]*composeServiceListState, 0)

	appendComposeSummary := func(route system.TbLogProjectRoute, workDir string, serviceName string, source string, routeName string, state *composeServiceListState) {
		serviceName = strings.TrimSpace(serviceName)
		key := fmt.Sprintf("compose:%s:%s", filepath.Clean(workDir), serviceName)
		if serviceName == "" {
			key = fmt.Sprintf("compose:%s:route:%d", filepath.Clean(workDir), route.ID)
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if state != nil {
			state.Emitted[serviceName] = struct{}{}
			if serviceName == "" {
				state.RouteOnly = true
			}
		}
		if strings.TrimSpace(routeName) == "" {
			routeName = route.RouteName
		}
		result = append(result, DockerServiceSummary{
			ProjectID:   project.ID,
			RouteID:     route.ID,
			RouteName:   routeName,
			ServiceName: serviceName,
			WorkDir:     workDir,
			Source:      source,
			RouteType:   logRouteTypeDockerCompose,
			Running:     dockerComposeServiceRunning(workDir, serviceName),
		})
	}

	for _, route := range routes {
		workDir := resolveLogRouteWorkDir(project, route)
		if isFileLogRoute(route) {
			logFilePath := resolveLogFilePath(project, route)
			if strings.TrimSpace(logFilePath) == "" {
				continue
			}
			key := fmt.Sprintf("%d:file:%s", route.ID, logFilePath)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			serviceName := strings.TrimSpace(route.RouteKey)
			if serviceName == startupLogLauncherKey {
				continue
			}
			if serviceName == "" {
				serviceName = strings.TrimSpace(route.RouteName)
			}
			if serviceName == "" {
				serviceName = filepath.Base(logFilePath)
			}
			result = append(result, DockerServiceSummary{
				ProjectID:   project.ID,
				RouteID:     route.ID,
				RouteName:   route.RouteName,
				ServiceName: serviceName,
				WorkDir:     workDir,
				Source:      "file-log",
				LogFilePath: logFilePath,
				RouteType:   logRouteTypeFileLog,
				Running:     fileLogRouteRunning(project, route),
			})
			continue
		}
		if normalizedScope == "service" {
			continue
		}
		if !isLogDockerComposeRoute(route, workDir) {
			continue
		}
		stateKey := filepath.Clean(workDir)
		state := composeStates[stateKey]
		if state == nil {
			state = &composeServiceListState{
				WorkDir:  workDir,
				Route:    route,
				Services: composeServices(workDir),
				Emitted:  map[string]struct{}{},
			}
			composeStates[stateKey] = state
			composeStateOrder = append(composeStateOrder, state)
		}
		services := state.Services
		if serviceName, ok := routeSpecificComposeService(route, services); ok {
			appendComposeSummary(route, workDir, serviceName, "docker-compose", route.RouteName, state)
			continue
		}
		if len(services) == 0 {
			appendComposeSummary(route, workDir, "", "route", route.RouteName, state)
			continue
		}
		for _, serviceName := range services {
			appendComposeSummary(route, workDir, serviceName, "docker-compose", route.RouteName, state)
		}
	}

	for _, state := range composeStateOrder {
		if state.RouteOnly || len(state.Services) == 0 {
			continue
		}
		for _, serviceName := range state.Services {
			if _, ok := state.Emitted[serviceName]; ok {
				continue
			}
			appendComposeSummary(state.Route, state.WorkDir, serviceName, "docker-compose", "Docker Compose 自动发现", state)
		}
	}
	return result, nil
}

func annotateLogProjectRoutes(project *system.TbLogProject) {
	if project == nil {
		return
	}
	for i := range project.Routes {
		project.Routes[i].RouteType = logRouteType(project.Routes[i])
	}
}

func logRouteType(route system.TbLogProjectRoute) string {
	if isFileLogRoute(route) {
		return logRouteTypeFileLog
	}
	if isDirectDockerComposeLogRoute(route) {
		return logRouteTypeDockerCompose
	}
	return logRouteTypeScript
}

func routeSpecificComposeService(route system.TbLogProjectRoute, services []string) (string, bool) {
	routeKey := strings.TrimSpace(route.RouteKey)
	if routeKey == "" || len(services) == 0 {
		return "", false
	}
	for _, serviceName := range services {
		if serviceName == routeKey {
			return routeKey, true
		}
	}
	return "", false
}

func streamDockerComposeServiceGroupRestart(project system.TbLogProject, routes []system.TbLogProjectRoute, logCh chan string) error {
	groups := dockerComposeRouteGroups(project, routes)
	if len(groups) == 0 {
		return fmt.Errorf("当前日志项目没有可执行的 Docker Compose 服务路线")
	}

	sendLog(logCh, fmt.Sprintf("📋 日志项目: %s", project.ProjectName))
	sendLog(logCh, fmt.Sprintf("📦 本次重启 %d 个 Docker Compose 编排", len(groups)))

	var failed []string
	for index, group := range groups {
		services := composeServices(group.WorkDir)
		sendLog(logCh, "")
		sendLog(logCh, fmt.Sprintf("▶️ [%d/%d] Docker Compose: %s", index+1, len(groups), group.WorkDir))
		if len(services) > 0 {
			sendLog(logCh, fmt.Sprintf("🐳 Compose 服务: %s", strings.Join(services, ", ")))
		}
		sendLog(logCh, fmt.Sprintf("🏠 工作目录: %s", group.WorkDir))
		sendLog(logCh, "🔨 执行命令: docker compose restart")
		if err := runCommandArgsWithLog(context.Background(), group.WorkDir, "docker", []string{"compose", "restart"}, logCh); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", group.WorkDir, err))
			sendLog(logCh, fmt.Sprintf("❌ Docker Compose 重启失败: %v", err))
			continue
		}
		sendLog(logCh, "✅ Docker Compose 编排重启完成")
	}

	if len(failed) > 0 {
		return fmt.Errorf("%d 个 Docker Compose 编排重启失败: %s", len(failed), strings.Join(failed, "；"))
	}
	sendLog(logCh, "✅ Docker Compose 服务组重启完成")
	return nil
}

func dockerComposeRouteGroups(project system.TbLogProject, routes []system.TbLogProjectRoute) []composeRouteGroup {
	groups := make([]composeRouteGroup, 0)
	seen := map[string]struct{}{}
	for _, route := range routes {
		workDir := resolveLogRouteWorkDir(project, route)
		if strings.TrimSpace(workDir) == "" || !isLogDockerComposeRoute(route, workDir) {
			continue
		}
		key := filepath.Clean(workDir)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		groups = append(groups, composeRouteGroup{
			WorkDir: workDir,
			Route:   route,
		})
	}
	return groups
}

func executeLogRoute(ctx context.Context, project system.TbLogProject, route system.TbLogProjectRoute, action string, logCh chan string) error {
	if isFileLogRoute(route) {
		if action == "restart" {
			return restartFileLogRoute(ctx, project, route, logCh)
		}
		return fmt.Errorf("文件日志路线不支持%s", groupActionLabel(action))
	}
	workDir := resolveLogRouteWorkDir(project, route)
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("服务目录为空")
	}

	commands, err := logRouteActionCommands(route, action, workDir)
	if err != nil {
		return err
	}

	for _, command := range commands {
		sendLog(logCh, fmt.Sprintf("🏠 工作目录: %s", workDir))
		sendLog(logCh, fmt.Sprintf("🔨 执行命令: %s", command))
		if err := runShellCommandWithLog(ctx, workDir, command, logCh); err != nil {
			return err
		}
	}
	return nil
}

func logRouteActionCommands(route system.TbLogProjectRoute, action string, workDir string) ([]string, error) {
	switch action {
	case "stop":
		return logRouteStopCommands(route, workDir)
	case "restart":
		stopCommands, err := logRouteStopCommands(route, workDir)
		if err != nil {
			return nil, err
		}
		startCommands, err := logRouteStartCommands(route)
		if err != nil {
			return nil, err
		}
		return append(stopCommands, startCommands...), nil
	default:
		return logRouteStartCommands(route)
	}
}

func logRouteStopCommands(route system.TbLogProjectRoute, workDir string) ([]string, error) {
	stopCommand := strings.TrimSpace(route.LocalStopCommand)
	if stopCommand == "" && isLogDockerComposeRoute(route, workDir) {
		stopCommand = "docker compose down"
	}
	if stopCommand == "" {
		return nil, fmt.Errorf("关闭命令未配置")
	}
	return []string{stopCommand}, nil
}

func logRouteStartCommands(route system.TbLogProjectRoute) ([]string, error) {
	commands := make([]string, 0, 2)
	if command := strings.TrimSpace(route.LocalExecuteCommand); command != "" {
		commands = append(commands, command)
	}
	if command := strings.TrimSpace(route.LocalStartCommand); command != "" {
		commands = append(commands, command)
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("启动命令未配置")
	}
	return commands, nil
}

func restartFileLogRoute(ctx context.Context, project system.TbLogProject, route system.TbLogProjectRoute, logCh chan string) error {
	logFilePath := resolveLogFilePath(project, route)
	if strings.TrimSpace(logFilePath) == "" {
		return fmt.Errorf("日志文件路径未配置")
	}
	serviceName := strings.TrimSpace(route.RouteKey)
	if serviceName == "" {
		serviceName = strings.TrimSuffix(filepath.Base(logFilePath), filepath.Ext(logFilePath))
	}
	if serviceName == "" {
		return fmt.Errorf("服务名为空，无法重启单个服务")
	}

	runtimeDir := serviceRuntimeDir(logFilePath)
	if runtimeDir == "" {
		return fmt.Errorf("日志文件不在 .service-runtime/logs 下，无法定位单服务运行时")
	}
	plistPath := filepath.Join(runtimeDir, "launchd", serviceName+".plist")
	info, err := parseLaunchdPlist(plistPath)
	if err != nil {
		return fmt.Errorf("读取服务启动配置失败: %w", err)
	}
	if strings.TrimSpace(info.WorkingDirectory) == "" {
		info.WorkingDirectory = resolveLogRouteWorkDir(project, route)
	}
	if strings.TrimSpace(info.StandardOutPath) == "" {
		info.StandardOutPath = logFilePath
	}
	if len(info.ProgramArguments) == 0 {
		return fmt.Errorf("服务启动参数未配置")
	}

	pidPath := serviceRuntimePidPath(logFilePath, serviceName)
	sendLog(logCh, fmt.Sprintf("📋 服务名: %s", serviceName))
	sendLog(logCh, fmt.Sprintf("📄 日志文件: %s", logFilePath))
	sendLog(logCh, fmt.Sprintf("🏠 工作目录: %s", info.WorkingDirectory))

	stopRuntimeService(ctx, serviceName, info, plistPath, pidPath, logCh)
	if err := startRuntimeService(ctx, serviceName, info, plistPath, logFilePath, pidPath, logCh); err != nil {
		return err
	}
	sendLog(logCh, fmt.Sprintf("✅ %s 重启完成", serviceName))
	return nil
}

func stopRuntimeService(ctx context.Context, serviceName string, info launchdPlistInfo, plistPath string, pidPath string, logCh chan string) {
	if canUseLaunchctl() && strings.TrimSpace(info.Label) != "" {
		domain := launchctlDomain()
		sendLog(logCh, fmt.Sprintf("⏹️ 关闭 launchd 服务: %s", serviceName))
		if err := runCommandArgsWithLog(ctx, "", "launchctl", []string{"bootout", domain + "/" + info.Label}, logCh); err != nil {
			_ = runCommandArgsWithLog(ctx, "", "launchctl", []string{"bootout", domain, plistPath}, logCh)
			_ = runCommandArgsWithLog(ctx, "", "launchctl", []string{"remove", info.Label}, logCh)
		}
	}
	stopPidFile(pidPath, logCh)
}

func startRuntimeService(ctx context.Context, serviceName string, info launchdPlistInfo, plistPath string, logFilePath string, pidPath string, logCh chan string) error {
	if canUseLaunchctl() && strings.TrimSpace(info.Label) != "" {
		domain := launchctlDomain()
		sendLog(logCh, fmt.Sprintf("🚀 启动 launchd 服务: %s", serviceName))
		if err := runCommandArgsWithLog(ctx, "", "launchctl", []string{"bootstrap", domain, plistPath}, logCh); err != nil {
			return fmt.Errorf("启动 launchd 服务失败: %w", err)
		}
		_ = runCommandArgsWithLog(ctx, "", "launchctl", []string{"kickstart", "-k", domain + "/" + info.Label}, logCh)
		time.Sleep(time.Second)
		if pid := launchctlServicePID(ctx, info.Label); pid > 0 && strings.TrimSpace(pidPath) != "" {
			_ = os.MkdirAll(filepath.Dir(pidPath), 0o755)
			_ = os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o644)
			sendLog(logCh, fmt.Sprintf("✅ %s 已启动，pid=%d", serviceName, pid))
		}
		return nil
	}

	sendLog(logCh, fmt.Sprintf("🚀 直接启动服务: %s", serviceName))
	return startDetachedRuntimeProcess(info.WorkingDirectory, info.ProgramArguments, logFilePath, pidPath, logCh)
}

func canUseLaunchctl() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if strings.TrimSpace(os.Getenv("USE_LAUNCHCTL")) == "0" {
		return false
	}
	_, err := exec.LookPath("launchctl")
	return err == nil
}

func launchctlDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func launchctlServicePID(ctx context.Context, label string) int {
	if strings.TrimSpace(label) == "" {
		return 0
	}
	cmd := exec.CommandContext(ctx, "launchctl", "print", launchctlDomain()+"/"+label)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "= ", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != "pid" {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err == nil && pid > 0 {
			return pid
		}
	}
	return 0
}

func stopPidFile(pidPath string, logCh chan string) {
	if strings.TrimSpace(pidPath) == "" {
		return
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		_ = os.Remove(pidPath)
		return
	}
	if syscall.Kill(pid, 0) == nil {
		sendLog(logCh, fmt.Sprintf("⏹️ 停止 pid: %d", pid))
		_ = syscall.Kill(pid, syscall.SIGTERM)
		time.Sleep(2 * time.Second)
		if syscall.Kill(pid, 0) == nil {
			sendLog(logCh, fmt.Sprintf("⏹️ 强制停止 pid: %d", pid))
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	_ = os.Remove(pidPath)
}

func startDetachedRuntimeProcess(workDir string, args []string, logFilePath string, pidPath string, logCh chan string) error {
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("服务工作目录为空")
	}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("服务启动参数为空")
	}
	if strings.TrimSpace(logFilePath) == "" {
		return fmt.Errorf("服务日志路径为空")
	}
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0o755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Env = enrichedCommandEnv()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}
	if strings.TrimSpace(pidPath) != "" {
		_ = os.MkdirAll(filepath.Dir(pidPath), 0o755)
		_ = os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	}
	sendLog(logCh, fmt.Sprintf("✅ 服务已启动，pid=%d", cmd.Process.Pid))
	return nil
}

func runCommandArgsWithLog(ctx context.Context, workDir string, command string, args []string, logCh chan string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Env = enrichedCommandEnv()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建 stdout 管道失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建 stderr 管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanDone := make(chan struct{}, 2)
	scan := func(scanner *bufio.Scanner) {
		defer func() { scanDone <- struct{}{} }()
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case logCh <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}
	go scan(bufio.NewScanner(stdout))
	go scan(bufio.NewScanner(stderr))

	err = cmd.Wait()
	<-scanDone
	<-scanDone
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func runShellCommandWithLog(ctx context.Context, workDir string, command string, logCh chan string) error {
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = workDir
	cmd.Env = enrichedCommandEnv()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建 stdout 管道失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建 stderr 管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	scanDone := make(chan struct{}, 2)
	scan := func(scanner *bufio.Scanner) {
		defer func() { scanDone <- struct{}{} }()
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case logCh <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}
	go scan(bufio.NewScanner(stdout))
	go scan(bufio.NewScanner(stderr))

	err = cmd.Wait()
	<-scanDone
	<-scanDone
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("命令执行失败: %w", err)
	}
	return nil
}

func localLogRoutes(routes []system.TbLogProjectRoute) []system.TbLogProjectRoute {
	result := make([]system.TbLogProjectRoute, 0, len(routes))
	for _, route := range routes {
		if strings.TrimSpace(route.RouteName) == "" {
			continue
		}
		result = append(result, route)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Sort != result[j].Sort {
			return result[i].Sort < result[j].Sort
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func scopedLogRoutes(routes []system.TbLogProjectRoute, scope string) []system.TbLogProjectRoute {
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	if normalizedScope == "" {
		return routes
	}

	result := make([]system.TbLogProjectRoute, 0, len(routes))
	for _, route := range routes {
		routeType := logRouteType(route)
		switch normalizedScope {
		case "docker":
			if routeType == logRouteTypeDockerCompose {
				result = append(result, route)
			}
		case "service":
			if routeType == logRouteTypeScript || routeType == logRouteTypeFileLog {
				result = append(result, route)
			}
		default:
			result = append(result, route)
		}
	}
	return result
}

func executableLogRoutes(routes []system.TbLogProjectRoute) []system.TbLogProjectRoute {
	result := make([]system.TbLogProjectRoute, 0, len(routes))
	for _, route := range routes {
		if isFileLogRoute(route) {
			continue
		}
		result = append(result, route)
	}
	return result
}

func logRouteScopeLabel(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "docker":
		return "Docker Compose 服务"
	case "service":
		return "脚本服务"
	default:
		return "服务"
	}
}

func routeLaunchCommand(route system.TbLogProjectRoute) string {
	if command := strings.TrimSpace(route.LocalExecuteCommand); command != "" {
		return command
	}
	return strings.TrimSpace(route.LocalStartCommand)
}

func isScriptLaunchCommand(command string) bool {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "./") ||
		strings.Contains(normalized, ".sh ") ||
		strings.HasSuffix(normalized, ".sh") {
		return true
	}
	for _, prefix := range []string{
		"sh ", "bash ", "zsh ", "fish ", "source ",
		"make ", "npm ", "pnpm ", "yarn ", "node ",
		"python ", "python3 ", "java ", "go ", "mvn ", "gradle ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func isDirectDockerComposeLogRoute(route system.TbLogProjectRoute) bool {
	if isFileLogRoute(route) {
		return false
	}
	commandText := strings.ToLower(route.LocalExecuteCommand + " " + route.LocalStartCommand)
	directComposeCommand := strings.Contains(commandText, "docker compose") || strings.Contains(commandText, "docker-compose")
	composeMarked := route.DockerComposeDeploy || route.BuildType == "docker_compose_deploy"
	if composeMarked {
		return true
	}
	return !isScriptLaunchCommand(routeLaunchCommand(route)) && directComposeCommand
}

func findLogRoute(routes []system.TbLogProjectRoute, targetEnv string) (system.TbLogProjectRoute, error) {
	targetEnv = strings.TrimSpace(targetEnv)
	for _, route := range routes {
		if route.RouteKey == targetEnv || strconv.FormatUint(uint64(route.ID), 10) == targetEnv {
			return route, nil
		}
	}
	return system.TbLogProjectRoute{}, fmt.Errorf("未找到对应服务路线: %s", targetEnv)
}

func resolveLogRouteWorkDir(project system.TbLogProject, route system.TbLogProjectRoute) string {
	if path := strings.TrimSpace(route.LocalProjectPath); path != "" {
		return path
	}
	return strings.TrimSpace(project.LocalProjectPath)
}

func isFileLogRoute(route system.TbLogProjectRoute) bool {
	return strings.EqualFold(strings.TrimSpace(route.BuildType), "file_log") ||
		strings.TrimSpace(route.LogFilePath) != ""
}

func resolveLogFilePath(project system.TbLogProject, route system.TbLogProjectRoute) string {
	logFilePath := strings.TrimSpace(route.LogFilePath)
	if logFilePath == "" {
		return ""
	}
	if filepath.IsAbs(logFilePath) {
		return filepath.Clean(logFilePath)
	}
	workDir := resolveLogRouteWorkDir(project, route)
	if strings.TrimSpace(workDir) == "" {
		return filepath.Clean(logFilePath)
	}
	return filepath.Join(workDir, logFilePath)
}

func fileLogRouteRunning(project system.TbLogProject, route system.TbLogProjectRoute) bool {
	logFilePath := resolveLogFilePath(project, route)
	if strings.TrimSpace(logFilePath) == "" {
		return false
	}
	pidCandidates := []string{
		strings.TrimSuffix(logFilePath, filepath.Ext(logFilePath)) + ".pid",
	}
	routeKey := strings.TrimSpace(route.RouteKey)
	if routeKey != "" {
		pidCandidates = append(pidCandidates, filepath.Join(filepath.Dir(logFilePath), routeKey+".pid"))
	}
	if pidPath := serviceRuntimePidPath(logFilePath, routeKey); pidPath != "" {
		pidCandidates = append(pidCandidates, pidPath)
	}
	for _, pidPath := range pidCandidates {
		if pidFileRunning(pidPath) {
			return true
		}
	}
	return fileHasOpenProcess(logFilePath)
}

func pidFileRunning(pidPath string) bool {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func dockerComposeServiceRunning(workDir string, serviceName string) bool {
	if strings.TrimSpace(workDir) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	args := []string{"compose", "ps", "--status", "running", "--services"}
	if strings.TrimSpace(serviceName) != "" {
		args = append(args, strings.TrimSpace(serviceName))
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = workDir
	cmd.Env = enrichedCommandEnv()
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}
	return false
}

func fileHasOpenProcess(filePath string) bool {
	if strings.TrimSpace(filePath) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsof", "-t", filePath)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		pid := strings.TrimSpace(line)
		if pid == "" {
			continue
		}
		if _, err := strconv.Atoi(pid); err == nil {
			return true
		}
	}
	return false
}

func buildFileLogCommand(project system.TbLogProject, route system.TbLogProjectRoute) (string, []string, string, error) {
	logFilePath := resolveLogFilePath(project, route)
	if strings.TrimSpace(logFilePath) == "" {
		return "", nil, "", fmt.Errorf("日志文件路径未配置")
	}
	workDir := resolveLogRouteWorkDir(project, route)
	if strings.TrimSpace(workDir) == "" {
		workDir = filepath.Dir(logFilePath)
	}
	return "tail", []string{"-n", dockerLogTailLines, "-f", logFilePath}, workDir, nil
}

func buildLogDockerLogCommand(project system.TbLogProject, route system.TbLogProjectRoute, localProjectPath string, serviceName string) (string, []string, string) {
	if isLogDockerComposeRoute(route, localProjectPath) {
		args := []string{"compose", "logs", "--tail", dockerLogTailLines, "-f"}
		if strings.TrimSpace(serviceName) != "" {
			args = append(args, strings.TrimSpace(serviceName))
		}
		return "docker", args, localProjectPath
	}
	containerName := strings.TrimSpace(project.ProjectName)
	args := []string{"logs", "--tail", dockerLogTailLines, "-f", containerName}
	return "docker", args, localProjectPath
}

func isLogDockerComposeRoute(route system.TbLogProjectRoute, localProjectPath string) bool {
	executeCommand := strings.ToLower(route.LocalExecuteCommand + " " + route.LocalStopCommand + " " + route.BuildType)
	hasComposeFile := false
	for _, composeFilePath := range composeFilePaths(localProjectPath) {
		if _, err := os.Stat(composeFilePath); err == nil {
			hasComposeFile = true
			break
		}
	}
	return strings.Contains(executeCommand, "docker compose") ||
		strings.Contains(executeCommand, "docker-compose") ||
		route.DockerComposeDeploy ||
		route.BuildType == "docker_compose_deploy" ||
		hasComposeFile
}

func composeServices(workDir string) []string {
	if strings.TrimSpace(workDir) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "config", "--services")
	cmd.Dir = workDir
	cmd.Env = enrichedCommandEnv()
	output, err := cmd.Output()
	if err != nil {
		return composeServicesFromFiles(workDir)
	}

	lines := strings.Split(string(output), "\n")
	services := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		services = append(services, name)
	}
	sort.Strings(services)
	if len(services) == 0 {
		return composeServicesFromFiles(workDir)
	}
	return services
}

func composeServicesFromFiles(workDir string) []string {
	for _, composeFilePath := range composeFilePaths(workDir) {
		data, err := os.ReadFile(composeFilePath)
		if err != nil {
			continue
		}
		var compose struct {
			Services map[string]interface{} `yaml:"services"`
		}
		if err := yaml.Unmarshal(data, &compose); err != nil {
			continue
		}
		services := make([]string, 0, len(compose.Services))
		for serviceName := range compose.Services {
			serviceName = strings.TrimSpace(serviceName)
			if serviceName != "" {
				services = append(services, serviceName)
			}
		}
		sort.Strings(services)
		if len(services) > 0 {
			return services
		}
	}
	return nil
}

func composeFilePaths(workDir string) []string {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return nil
	}
	return []string{
		filepath.Join(workDir, "docker-compose.yml"),
		filepath.Join(workDir, "docker-compose.yaml"),
		filepath.Join(workDir, "compose.yml"),
		filepath.Join(workDir, "compose.yaml"),
	}
}

func normalizeLogAction(action string) (string, error) {
	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	if normalizedAction == "" {
		normalizedAction = "start"
	}
	if normalizedAction != "start" && normalizedAction != "stop" && normalizedAction != "restart" {
		return "", fmt.Errorf("不支持的服务动作: %s", action)
	}
	return normalizedAction, nil
}

func groupActionLabel(action string) string {
	if action == "restart" {
		return "重启"
	}
	if action == "stop" {
		return "关闭"
	}
	return "启动"
}

func sendLog(logCh chan string, msg string) {
	if logCh != nil {
		select {
		case logCh <- msg:
		default:
		}
	}
}

func startupLogWorkspaces() []string {
	if value := strings.TrimSpace(os.Getenv(startupLogWorkspaceEnv)); value != "" {
		parts := filepath.SplitList(value)
		workspaces := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				workspaces = append(workspaces, trimmed)
			}
		}
		return workspaces
	}
	return []string{"/Users/conchi/workforce/rob_english_word_workforce"}
}

func resolveStartupLogProjectConfig(req request.TbLogProjectSearch, fallbackName string) (uint, string, error) {
	projectConfigName := strings.TrimSpace(os.Getenv(startupLogProjectConfigEnv))
	if projectConfigName == "" {
		projectConfigName = startupLogDefaultProjectConfigName
	}
	if projectConfigName != "" && global.GVA_DB != nil {
		var project system.TbInterfaceProject
		err := global.GVA_DB.Where("project_name = ?", projectConfigName).First(&project).Error
		if err == nil {
			return project.ID, project.ProjectName, nil
		}
		if err != gorm.ErrRecordNotFound {
			return 0, "", err
		}
	}

	projectConfigID := req.ProjectConfigId
	projectConfigName = strings.TrimSpace(req.ProjectConfigName)
	if projectConfigID == 0 && projectConfigName == "" {
		projectConfigName = strings.TrimSpace(fallbackName)
	}
	return projectConfigID, projectConfigName, nil
}

func discoverStartupLogEntries(workspace string) ([]startupLogEntry, error) {
	workspace = filepath.Clean(workspace)
	entries := make([]startupLogEntry, 0)
	seenLogs := map[string]struct{}{}

	plistPaths, err := filepath.Glob(filepath.Join(workspace, ".service-runtime", "launchd", "*.plist"))
	if err != nil {
		return nil, err
	}
	sort.Strings(plistPaths)
	for _, plistPath := range plistPaths {
		info, err := parseLaunchdPlist(plistPath)
		if err != nil {
			return nil, err
		}
		logPath := strings.TrimSpace(info.StandardOutPath)
		if logPath == "" {
			logPath = strings.TrimSpace(info.StandardErrorPath)
		}
		if logPath == "" {
			continue
		}
		logPath = filepath.Clean(logPath)
		if _, ok := seenLogs[logPath]; ok {
			continue
		}
		seenLogs[logPath] = struct{}{}
		serviceName := startupLogServiceName(info.Label, logPath)
		workDir := strings.TrimSpace(info.WorkingDirectory)
		entries = append(entries, startupLogEntry{
			ServiceName: serviceName,
			WorkDir:     workDir,
			LogFilePath: logPath,
			Language:    detectStartupLogLanguage(workDir),
		})
	}

	logPaths, err := filepath.Glob(filepath.Join(workspace, ".service-runtime", "logs", "*.log"))
	if err != nil {
		return nil, err
	}
	sort.Strings(logPaths)
	for _, logPath := range logPaths {
		logPath = filepath.Clean(logPath)
		if _, ok := seenLogs[logPath]; ok {
			continue
		}
		serviceName := startupLogServiceName("", logPath)
		workDir := inferStartupLogWorkDir(workspace, serviceName)
		entries = append(entries, startupLogEntry{
			ServiceName: serviceName,
			WorkDir:     workDir,
			LogFilePath: logPath,
			Language:    detectStartupLogLanguage(workDir),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ServiceName < entries[j].ServiceName
	})
	return entries, nil
}

func parseLaunchdPlist(plistPath string) (launchdPlistInfo, error) {
	file, err := os.Open(plistPath)
	if err != nil {
		return launchdPlistInfo{}, err
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	var info launchdPlistInfo
	currentKey := ""
	inProgramArguments := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return launchdPlistInfo{}, err
		}

		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "key":
				var key string
				if err := decoder.DecodeElement(&key, &element); err != nil {
					return launchdPlistInfo{}, err
				}
				currentKey = key
			case "array":
				if currentKey == "ProgramArguments" {
					inProgramArguments = true
				}
			case "string":
				var value string
				if err := decoder.DecodeElement(&value, &element); err != nil {
					return launchdPlistInfo{}, err
				}
				if inProgramArguments {
					info.ProgramArguments = append(info.ProgramArguments, value)
					continue
				}
				switch currentKey {
				case "Label":
					info.Label = value
				case "WorkingDirectory":
					info.WorkingDirectory = value
				case "StandardOutPath":
					info.StandardOutPath = value
				case "StandardErrorPath":
					info.StandardErrorPath = value
				}
			}
		case xml.EndElement:
			if element.Name.Local == "array" && inProgramArguments {
				inProgramArguments = false
				currentKey = ""
			}
		}
	}
	return info, nil
}

func startupLogServiceName(label string, logPath string) string {
	if base := strings.TrimSuffix(filepath.Base(logPath), filepath.Ext(logPath)); base != "" {
		return base
	}
	parts := strings.Split(strings.TrimSpace(label), ".")
	if len(parts) > 0 {
		if tail := strings.TrimSpace(parts[len(parts)-1]); tail != "" {
			return tail
		}
	}
	return "startup-log"
}

func inferStartupLogWorkDir(workspace string, serviceName string) string {
	direct := filepath.Join(workspace, serviceName)
	if info, err := os.Stat(direct); err == nil && info.IsDir() {
		return direct
	}

	markers := []string{"package.json", "go.mod", "pom.xml", "pyproject.toml", "requirements.txt", "Makefile"}
	bestDir := ""
	bestScore := -1
	_ = filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == ".git" || name == "node_modules" || name == "target" || name == "dist" || name == ".service-runtime" {
			return filepath.SkipDir
		}

		hasMarker := false
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
				hasMarker = true
				break
			}
		}
		if !hasMarker {
			return nil
		}

		score := startupLogWorkDirScore(path, serviceName)
		if score > bestScore {
			bestScore = score
			bestDir = path
		}
		return nil
	})
	if bestDir != "" {
		return bestDir
	}
	return workspace
}

func startupLogWorkDirScore(path string, serviceName string) int {
	normalizedPath := strings.ToLower(strings.ReplaceAll(path, "-", "_"))
	normalizedService := strings.ToLower(strings.ReplaceAll(serviceName, "-", "_"))
	score := 0
	for _, part := range strings.Split(normalizedService, "_") {
		if part == "" {
			continue
		}
		if strings.Contains(normalizedPath, part) {
			score++
		}
	}
	if strings.Contains(normalizedPath, normalizedService) {
		score += 10
	}
	return score
}

func detectStartupLogLanguage(workDir string) string {
	if strings.TrimSpace(workDir) == "" {
		return "service"
	}
	if startupLogFileExists(filepath.Join(workDir, "pom.xml")) {
		return "java"
	}
	if startupLogFileExists(filepath.Join(workDir, "go.mod")) {
		return "go"
	}
	if startupLogFileExists(filepath.Join(workDir, "pyproject.toml")) || startupLogFileExists(filepath.Join(workDir, "requirements.txt")) {
		return "python"
	}
	if startupLogFileExists(filepath.Join(workDir, "package.json")) {
		return "react"
	}
	return "service"
}

func startupLogFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func ensureStartupLogProjectGroup(groupName string) (uint, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = "startup logs"
	}
	var group system.TbLogProjectGroup
	err := global.GVA_DB.Where("group_name = ?", groupName).First(&group).Error
	if err == nil {
		return group.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	group = system.TbLogProjectGroup{
		GroupName: groupName,
		Sort:      100,
	}
	if err := global.GVA_DB.Create(&group).Error; err != nil {
		return 0, err
	}
	return group.ID, nil
}

func upsertStartupLogWorkspaceProject(groupID uint, projectConfigID uint, projectConfigName string, workspace string, entries []startupLogEntry) error {
	workspace = filepath.Clean(workspace)
	projectName := filepath.Base(workspace)
	if strings.TrimSpace(projectName) == "" || projectName == "." || projectName == string(filepath.Separator) {
		projectName = "startup logs"
	}

	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		project, err := ensureStartupLogWorkspaceProject(tx, groupID, projectConfigID, projectConfigName, projectName, workspace)
		if err != nil {
			return err
		}

		if err := upsertStartupLogLauncherRoute(tx, project.ID, workspace); err != nil {
			return err
		}

		currentRouteKeys := make(map[string]struct{}, len(entries))
		for index, entry := range entries {
			routeKey := strings.TrimSpace(entry.ServiceName)
			if routeKey == "" {
				routeKey = strings.TrimSuffix(filepath.Base(entry.LogFilePath), filepath.Ext(entry.LogFilePath))
			}
			if routeKey == "" {
				routeKey = fmt.Sprintf("startup-log-%d", index+1)
			}
			currentRouteKeys[routeKey] = struct{}{}
			if err := upsertStartupLogFileRoute(tx, project.ID, workspace, index+1, routeKey, entry); err != nil {
				return err
			}
		}

		if err := deleteStaleStartupLogRoutes(tx, project.ID, workspace, currentRouteKeys); err != nil {
			return err
		}
		return deleteLegacyStartupLogProjects(tx, project.ID, workspace, entries)
	})
}

func ensureStartupLogWorkspaceProject(tx *gorm.DB, groupID uint, projectConfigID uint, projectConfigName string, projectName string, workspace string) (system.TbLogProject, error) {
	var project system.TbLogProject
	err := tx.
		Where("project_config_id = ? AND project_config_name = ? AND local_project_path = ?", projectConfigID, projectConfigName, workspace).
		First(&project).Error
	if err == nil {
		updates := map[string]interface{}{
			"group_id":            groupID,
			"project_config_id":   projectConfigID,
			"project_config_name": projectConfigName,
			"computer_language":   "service",
			"project_name":        projectName,
			"description":         startupLogDescription,
			"local_project_path":  workspace,
		}
		if err := tx.Model(&system.TbLogProject{}).Where("id = ?", project.ID).Updates(updates).Error; err != nil {
			return system.TbLogProject{}, err
		}
		project.GroupId = groupID
		project.ProjectConfigId = projectConfigID
		project.ProjectConfigName = projectConfigName
		project.ComputerLanguage = "service"
		project.ProjectName = projectName
		project.Description = startupLogDescription
		project.LocalProjectPath = workspace
		return project, nil
	}
	if err != gorm.ErrRecordNotFound {
		return system.TbLogProject{}, err
	}

	project = system.TbLogProject{
		GroupId:           groupID,
		ProjectConfigId:   projectConfigID,
		ProjectConfigName: projectConfigName,
		ComputerLanguage:  "service",
		ProjectName:       projectName,
		Description:       startupLogDescription,
		LocalProjectPath:  workspace,
	}
	if err := tx.Create(&project).Error; err != nil {
		return system.TbLogProject{}, err
	}
	return project, nil
}

func upsertStartupLogLauncherRoute(tx *gorm.DB, projectID uint, workspace string) error {
	scriptPath := filepath.Join(workspace, "restart_all_services.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		if err := tx.Where("project_id = ? AND route_key = ?", projectID, startupLogLauncherKey).
			Unscoped().
			Delete(&system.TbLogProjectRoute{}).Error; err != nil {
			return err
		}
		return nil
	}

	route := system.TbLogProjectRoute{}
	err := tx.Where("project_id = ? AND route_key = ?", projectID, startupLogLauncherKey).First(&route).Error
	values := map[string]interface{}{
		"route_name":            "全部服务启动器",
		"local_project_path":    workspace,
		"local_execute_command": "./restart_all_services.sh restart",
		"local_start_command":   "",
		"local_stop_command":    "./restart_all_services.sh stop",
		"log_file_path":         "",
		"build_type":            logRouteTypeScript,
		"docker_compose_deploy": false,
		"color":                 "emerald",
		"icon":                  "play",
		"sort":                  0,
	}
	if err == nil {
		return tx.Model(&system.TbLogProjectRoute{}).Where("id = ?", route.ID).Updates(values).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	route = system.TbLogProjectRoute{
		ProjectId:           int(projectID),
		RouteKey:            startupLogLauncherKey,
		RouteName:           "全部服务启动器",
		LocalProjectPath:    workspace,
		LocalExecuteCommand: "./restart_all_services.sh restart",
		LocalStopCommand:    "./restart_all_services.sh stop",
		BuildType:           logRouteTypeScript,
		Color:               "emerald",
		Icon:                "play",
		Sort:                0,
	}
	return tx.Create(&route).Error
}

func upsertStartupLogFileRoute(tx *gorm.DB, projectID uint, workspace string, sortOrder int, routeKey string, entry startupLogEntry) error {
	workDir := strings.TrimSpace(entry.WorkDir)
	if workDir == "" {
		workDir = workspace
	}
	routeName := fmt.Sprintf("%s 启动日志", routeKey)

	var route system.TbLogProjectRoute
	err := tx.
		Where("project_id = ? AND route_key = ?", projectID, routeKey).
		First(&route).Error
	values := map[string]interface{}{
		"route_name":            routeName,
		"local_project_path":    workDir,
		"local_execute_command": "",
		"local_start_command":   "",
		"local_stop_command":    "",
		"log_file_path":         entry.LogFilePath,
		"build_type":            logRouteTypeFileLog,
		"docker_compose_deploy": false,
		"color":                 "blue",
		"icon":                  "scroll-text",
		"sort":                  sortOrder,
	}
	if err == nil {
		return tx.Model(&system.TbLogProjectRoute{}).Where("id = ?", route.ID).Updates(values).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	route = system.TbLogProjectRoute{
		ProjectId:        int(projectID),
		RouteKey:         routeKey,
		RouteName:        routeName,
		LocalProjectPath: workDir,
		LogFilePath:      entry.LogFilePath,
		BuildType:        logRouteTypeFileLog,
		Color:            "blue",
		Icon:             "scroll-text",
		Sort:             sortOrder,
	}
	return tx.Create(&route).Error
}

func deleteStaleStartupLogRoutes(tx *gorm.DB, projectID uint, workspace string, currentRouteKeys map[string]struct{}) error {
	var routes []system.TbLogProjectRoute
	if err := tx.
		Where("project_id = ? AND build_type = ?", projectID, logRouteTypeFileLog).
		Find(&routes).Error; err != nil {
		return err
	}
	for _, route := range routes {
		if _, ok := currentRouteKeys[route.RouteKey]; ok {
			continue
		}
		if !isStartupLogPath(workspace, route.LogFilePath) {
			continue
		}
		if err := tx.Where("id = ?", route.ID).Unscoped().Delete(&system.TbLogProjectRoute{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteLegacyStartupLogProjects(tx *gorm.DB, projectID uint, workspace string, entries []startupLogEntry) error {
	workDirSet := make(map[string]struct{}, len(entries))
	serviceNameSet := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if workDir := strings.TrimSpace(entry.WorkDir); workDir != "" {
			workDirSet[filepath.Clean(workDir)] = struct{}{}
		}
		if serviceName := strings.TrimSpace(entry.ServiceName); serviceName != "" {
			serviceNameSet[serviceName] = struct{}{}
		}
	}

	var projects []system.TbLogProject
	if err := tx.
		Where("description = ? AND id <> ?", startupLogDescription, projectID).
		Find(&projects).Error; err != nil {
		return err
	}

	for _, project := range projects {
		projectPath := filepath.Clean(strings.TrimSpace(project.LocalProjectPath))
		exactWorkspace := projectPath == filepath.Clean(workspace)
		_, exactWorkDir := workDirSet[projectPath]
		_, matchingServiceName := serviceNameSet[strings.TrimSpace(project.ProjectName)]
		if !exactWorkspace && !exactWorkDir && !(matchingServiceName && isPathInside(workspace, projectPath)) {
			continue
		}
		if err := tx.Where("project_id = ?", project.ID).Unscoped().Delete(&system.TbLogProjectRoute{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", project.ID).Unscoped().Delete(&system.TbLogProject{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func isStartupLogPath(workspace string, logFilePath string) bool {
	logFilePath = strings.TrimSpace(logFilePath)
	if logFilePath == "" {
		return false
	}
	if !filepath.IsAbs(logFilePath) {
		logFilePath = filepath.Join(workspace, logFilePath)
	}
	return isPathInside(filepath.Join(workspace, ".service-runtime", "logs"), logFilePath)
}

func isPathInside(parent string, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func serviceRuntimePidPath(logFilePath string, routeKey string) string {
	runtimeDir := serviceRuntimeDir(logFilePath)
	if runtimeDir == "" {
		return ""
	}
	pidName := strings.TrimSpace(routeKey)
	if pidName == "" {
		pidName = strings.TrimSuffix(filepath.Base(logFilePath), filepath.Ext(logFilePath))
	}
	if pidName == "" {
		return ""
	}
	return filepath.Join(runtimeDir, "pids", pidName+".pid")
}

func serviceRuntimeDir(logFilePath string) string {
	logFilePath = strings.TrimSpace(logFilePath)
	if logFilePath == "" {
		return ""
	}
	logDir := filepath.Dir(logFilePath)
	if filepath.Base(logDir) != "logs" {
		return ""
	}
	runtimeDir := filepath.Dir(logDir)
	if filepath.Base(runtimeDir) != ".service-runtime" {
		return ""
	}
	return runtimeDir
}

func enrichedCommandEnv() []string {
	env := os.Environ()
	currentPath := os.Getenv("PATH")
	extraPaths := []string{
		"/usr/local/bin",
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/sbin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	}
	pathSet := map[string]bool{}
	for _, item := range strings.Split(currentPath, string(os.PathListSeparator)) {
		if item != "" {
			pathSet[item] = true
		}
	}
	nextPaths := []string{currentPath}
	for _, item := range extraPaths {
		if !pathSet[item] {
			nextPaths = append(nextPaths, item)
		}
	}
	env = append(env, "PATH="+strings.Join(nextPaths, string(os.PathListSeparator)))
	return env
}
