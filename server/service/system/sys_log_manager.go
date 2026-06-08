package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gorm.io/gorm"
)

type LogManagerService struct{}

const (
	logRouteTypeScript        = "script"
	logRouteTypeFileLog       = "file_log"
	logRouteTypeDockerCompose = "docker_compose"
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
}

func (s *LogManagerService) GetLogProjectPage(req request.TbLogProjectSearch) (list []system.TbLogProject, total int64, err error) {
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
		return global.GVA_DB.Model(&system.TbLogProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
			"group_id":            project.GroupId,
			"project_config_id":   project.ProjectConfigId,
			"project_config_name": project.ProjectConfigName,
			"computer_language":   project.ComputerLanguage,
			"project_name":        project.ProjectName,
			"description":         project.Description,
			"local_project_path":  project.LocalProjectPath,
			"user_id":             project.UserId,
		}).Error
	}
	return global.GVA_DB.Create(&project).Error
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

	routes := scopedLogRoutes(localLogRoutes(project.Routes), scope)
	result := make([]DockerServiceSummary, 0)
	seen := map[string]struct{}{}
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
			serviceName := strings.TrimSpace(route.RouteName)
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
			})
			continue
		}
		if !isLogDockerComposeRoute(route, workDir) {
			continue
		}
		services := composeServices(workDir)
		if serviceName, ok := routeSpecificComposeService(route, services); ok {
			services = []string{serviceName}
		}
		if len(services) == 0 {
			services = []string{""}
		}
		for _, serviceName := range services {
			key := fmt.Sprintf("%d:%s", route.ID, serviceName)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			source := "route"
			if serviceName != "" {
				source = "docker-compose"
			}
			result = append(result, DockerServiceSummary{
				ProjectID:   project.ID,
				RouteID:     route.ID,
				RouteName:   route.RouteName,
				ServiceName: serviceName,
				WorkDir:     workDir,
				Source:      source,
				RouteType:   logRouteTypeDockerCompose,
			})
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

func executeLogRoute(ctx context.Context, project system.TbLogProject, route system.TbLogProjectRoute, action string, logCh chan string) error {
	if isFileLogRoute(route) {
		return fmt.Errorf("文件日志路线不支持启动或关闭")
	}
	workDir := resolveLogRouteWorkDir(project, route)
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("服务目录为空")
	}

	commands := make([]string, 0, 2)
	switch action {
	case "stop":
		stopCommand := strings.TrimSpace(route.LocalStopCommand)
		if stopCommand == "" && isLogDockerComposeRoute(route, workDir) {
			stopCommand = "docker compose down"
		}
		if stopCommand == "" {
			return fmt.Errorf("关闭命令未配置")
		}
		commands = append(commands, stopCommand)
	default:
		if command := strings.TrimSpace(route.LocalExecuteCommand); command != "" {
			commands = append(commands, command)
		}
		if command := strings.TrimSpace(route.LocalStartCommand); command != "" {
			commands = append(commands, command)
		}
		if len(commands) == 0 {
			return fmt.Errorf("启动命令未配置")
		}
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
	return !isScriptLaunchCommand(routeLaunchCommand(route)) && (directComposeCommand || composeMarked)
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
	composeFilePath := filepath.Join(localProjectPath, "docker-compose.yml")
	composeYamlPath := filepath.Join(localProjectPath, "docker-compose.yaml")
	_, composeFileErr := os.Stat(composeFilePath)
	_, composeYamlErr := os.Stat(composeYamlPath)
	return strings.Contains(executeCommand, "docker compose") ||
		strings.Contains(executeCommand, "docker-compose") ||
		route.DockerComposeDeploy ||
		route.BuildType == "docker_compose_deploy" ||
		composeFileErr == nil ||
		composeYamlErr == nil
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
		return nil
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
	return services
}

func normalizeLogAction(action string) (string, error) {
	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	if normalizedAction == "" {
		normalizedAction = "start"
	}
	if normalizedAction != "start" && normalizedAction != "stop" {
		return "", fmt.Errorf("不支持的服务动作: %s", action)
	}
	return normalizedAction, nil
}

func groupActionLabel(action string) string {
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
