package system

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type DeployToolService struct{}

var DeployToolServiceApp = new(DeployToolService)

const deployImageVersion = "1.0.0"

// ─── Tool 1: scan_project ──────────────────────────────────────────────

// ScanProjectResult 扫描结果
type ScanProjectResult struct {
	ProjectName string `json:"projectName"`
	Language    string `json:"language"`
	EntryFile   string `json:"entryFile"`
	HasEnvFile  bool   `json:"hasEnvFile"`
}

// ScanProject 扫描本地项目目录，自动检测语言类型和项目名
func (s *DeployToolService) ScanProject(localPath string) (*ScanProjectResult, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("路径不存在: %s", localPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", localPath)
	}

	result := &ScanProjectResult{
		ProjectName: filepath.Base(localPath),
	}

	// 检测语言类型
	checks := []struct {
		file string
		lang string
	}{
		{"go.mod", "go"},
		{"pom.xml", "java"},
		{"build.gradle", "java"},
		{"pyproject.toml", "python"},
		{"requirements.txt", "python"},
	}

	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(localPath, c.file)); err == nil {
			result.Language = c.lang
			break
		}
	}

	// 如果还没检测到，检查 package.json (vue/react)
	if result.Language == "" {
		pkgPath := filepath.Join(localPath, "package.json")
		if data, err := os.ReadFile(pkgPath); err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
					if _, hasVue := deps["vue"]; hasVue {
						result.Language = "vue"
					} else if _, hasReact := deps["react"]; hasReact {
						result.Language = "react"
					}
				}
				if name, ok := pkg["name"].(string); ok && name != "" {
					result.ProjectName = name
				}
			}
		}
	}

	// 从 go.mod 提取 module name 作为参考
	if result.Language == "go" {
		if data, err := os.ReadFile(filepath.Join(localPath, "go.mod")); err == nil {
			lines := strings.Split(string(data), "\n")
			if len(lines) > 0 && strings.HasPrefix(lines[0], "module ") {
				mod := strings.TrimPrefix(lines[0], "module ")
				mod = strings.TrimSpace(mod)
				parts := strings.Split(mod, "/")
				if len(parts) > 0 {
					result.ProjectName = parts[len(parts)-1]
				}
			}
		}
	}

	// 从 pom.xml 提取 artifactId
	if result.Language == "java" {
		if data, err := os.ReadFile(filepath.Join(localPath, "pom.xml")); err == nil {
			if artifactId := extractRootMavenArtifactID(string(data)); artifactId != "" {
				result.ProjectName = artifactId
			}
		}
	}

	// 检测入口文件
	entryFiles := map[string][]string{
		"go":     {"main.go"},
		"java":   {"pom.xml"},
		"python": {"main.py", "app.py", "manage.py"},
		"vue":    {"src/main.ts", "src/main.js"},
		"react":  {"src/main.tsx", "src/index.tsx", "src/main.ts"},
	}
	if files, ok := entryFiles[result.Language]; ok {
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(localPath, f)); err == nil {
				result.EntryFile = f
				break
			}
		}
	}

	// 检测是否有 .env 文件
	if _, err := os.Stat(filepath.Join(localPath, ".env")); err == nil {
		result.HasEnvFile = true
	}

	return result, nil
}

// ─── Tool 2: create_deploy_project ─────────────────────────────────────

// QuickInitRequest 一键初始化请求
type QuickInitRequest struct {
	ProjectName        string `json:"projectName"`
	ComputerLanguage   string `json:"computerLanguage"`
	LocalProjectPath   string `json:"localProjectPath"`
	GroupId            uint   `json:"groupId"`
	ContainerName      string `json:"containerName"`
	AppPort            int    `json:"appPort"`
	FrontendDeployPort int    `json:"frontendDeployPort"`
	AllowDuplicatePath bool   `json:"allowDuplicatePath"`
	IncludeRemote      bool   `json:"includeRemote"`
	PreserveDeployName bool   `json:"-"`
}

type AutoCreateDeployProjectRequest struct {
	LocalPath          string `json:"localPath"`
	ProjectName        string `json:"projectName"`
	GroupId            uint   `json:"groupId"`
	ContainerName      string `json:"containerName"`
	AppPort            int    `json:"appPort"`
	FrontendDeployPort int    `json:"frontendDeployPort"`
	AllowDuplicatePath bool   `json:"allowDuplicatePath"`
	IncludeRemote      bool   `json:"includeRemote"`
	PreserveDeployName bool   `json:"-"`
}

type GenerateDeployInfoRequest struct {
	Input              string `json:"input"`
	LocalPath          string `json:"local_path"`
	GroupName          string `json:"group_name"`
	GroupAction        string `json:"group_action"`
	ProjectName        string `json:"project_name"`
	AppPort            int    `json:"app_port"`
	FrontendDeployPort int    `json:"frontend_deploy_port"`
	UseExistingGroup   bool   `json:"use_existing_group"`
	AllowDuplicatePath bool   `json:"allow_duplicate_path"`
	IncludeRemote      bool   `json:"include_remote"`
}

type PreviewReactDeployProjectRequest struct {
	LocalProjectPath   string `json:"localProjectPath"`
	ProjectName        string `json:"projectName"`
	GroupId            uint   `json:"groupId"`
	ContainerName      string `json:"containerName"`
	FrontendDeployPort int    `json:"frontendDeployPort"`
	BackendDeployPort  int    `json:"backendDeployPort"`
	AllowDuplicatePath bool   `json:"allowDuplicatePath"`
	PreserveDeployName bool   `json:"-"`
}

type PreviewFrontendDeployProjectRequest = PreviewReactDeployProjectRequest

type PreviewDeployRoute struct {
	RouteKey            string `json:"routeKey"`
	RouteName           string `json:"routeName"`
	LocalExecuteCommand string `json:"localExecuteCommand"`
	LocalStopCommand    string `json:"localStopCommand"`
	BuildType           string `json:"buildType"`
}

type PreviewDeployScript struct {
	FileName string `json:"fileName"`
	Bytes    int    `json:"bytes"`
}

type PreviewReactDeployProjectResult struct {
	ProjectName        string                `json:"projectName"`
	ComputerLanguage   string                `json:"computerLanguage"`
	LocalProjectPath   string                `json:"localProjectPath"`
	GroupId            uint                  `json:"groupId"`
	ContainerName      string                `json:"containerName"`
	AccessUrl          string                `json:"accessUrl"`
	FrontendDeployPort int                   `json:"frontendDeployPort"`
	BackendDeployPort  int                   `json:"backendDeployPort"`
	PackageManager     string                `json:"packageManager"`
	InstallCommand     string                `json:"installCommand"`
	BuildCommand       string                `json:"buildCommand"`
	Routes             []PreviewDeployRoute  `json:"routes"`
	Scripts            []PreviewDeployScript `json:"scripts"`
	ScriptPreview      map[string]string     `json:"scriptPreview"`
	Warnings           []string              `json:"warnings"`
}

type PreviewFrontendDeployProjectResult = PreviewReactDeployProjectResult

// QuickInitResult 初始化结果
type QuickInitResult struct {
	ProjectId        uint   `json:"projectId"`
	ProjectName      string `json:"projectName"`
	ComputerLanguage string `json:"computerLanguage"`
	AccessUrl        string `json:"accessUrl"`
	LocalProjectPath string `json:"localProjectPath"`
	RoutesCreated    int    `json:"routesCreated"`
	ScriptsCreated   int    `json:"scriptsCreated"`
}

type CreateProjectGroupResult struct {
	GroupId   uint   `json:"groupId"`
	GroupName string `json:"groupName"`
}

type GenerateDeployInfoResult struct {
	ProjectId          uint   `json:"projectId"`
	ProjectName        string `json:"projectName"`
	ComputerLanguage   string `json:"computerLanguage"`
	AccessUrl          string `json:"accessUrl"`
	LocalProjectPath   string `json:"localProjectPath"`
	GroupId            uint   `json:"groupId"`
	GroupName          string `json:"groupName"`
	AllowDuplicatePath bool   `json:"allowDuplicatePath"`
	RoutesCreated      int    `json:"routesCreated"`
	ScriptsCreated     int    `json:"scriptsCreated"`
}

type deployInfoIntent struct {
	LocalPath          string
	GroupName          string
	GroupAction        string
	ProjectName        string
	DeployPort         int
	AppPort            int
	FrontendDeployPort int
	AllowDuplicatePath bool
	UseExistingGroup   bool
	HasGroupName       bool
	HasProjectName     bool
	IncludeRemote      bool
}

const (
	deployInfoGroupActionCreate = "create"
	deployInfoGroupActionReuse  = "reuse"
	deployInfoGroupActionAuto   = "auto"
)

func (s *DeployToolService) AutoCreateDeployProject(req AutoCreateDeployProjectRequest) (*QuickInitResult, error) {
	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		return nil, fmt.Errorf("本地项目目录不能为空")
	}

	detected, err := DetectDeployProjectType(localPath)
	if err != nil {
		return nil, err
	}

	switch language := normalizeDeployLanguage(detected.ProjectType); language {
	case "react", "vue":
		return s.ConfirmFrontendDeployProject(PreviewFrontendDeployProjectRequest{
			LocalProjectPath:   localPath,
			ProjectName:        req.ProjectName,
			GroupId:            req.GroupId,
			ContainerName:      req.ContainerName,
			FrontendDeployPort: req.FrontendDeployPort,
			BackendDeployPort:  req.AppPort,
			AllowDuplicatePath: req.AllowDuplicatePath,
			PreserveDeployName: req.PreserveDeployName,
		}, language)
	case "docker-compose":
		return s.CreateGoReactComposeDeployProject(req, detected)
	case "java", "go", "python":
		projectName := strings.TrimSpace(req.ProjectName)
		if projectName == "" {
			projectName = detected.ProjectName
		}
		if projectName == "" {
			projectName = filepath.Base(localPath)
		}

		appPort := req.AppPort
		if appPort == 0 {
			next, err := ProjectServiceApp.GetNextDeployPort("backend")
			if err != nil {
				return nil, fmt.Errorf("获取后端建议端口失败: %w", err)
			}
			appPort = next.NextPort
		}

		return s.QuickInitProject(QuickInitRequest{
			ProjectName:        projectName,
			ComputerLanguage:   language,
			LocalProjectPath:   localPath,
			GroupId:            req.GroupId,
			ContainerName:      req.ContainerName,
			AppPort:            appPort,
			AllowDuplicatePath: req.AllowDuplicatePath,
			IncludeRemote:      req.IncludeRemote,
			PreserveDeployName: req.PreserveDeployName,
		})
	default:
		return nil, fmt.Errorf("当前自动目录生成支持 Vue/React/Java/Go/Python 项目，检测结果为: %s", detected.ProjectType)
	}
}

type goReactComposeParts struct {
	BackendPath string
	Frontends   []goReactComposeFrontendPart
	BackendRel  string
}

type goReactComposeFrontendPart struct {
	Path string
	Rel  string
}

func (s *DeployToolService) CreateGoReactComposeDeployProject(req AutoCreateDeployProjectRequest, detected *DeployProjectTypeResult) (*QuickInitResult, error) {
	if detected == nil {
		return nil, fmt.Errorf("前后端 docker-compose 检测结果不能为空")
	}
	if strings.TrimSpace(detected.PrimaryLanguage) != "Go + React" {
		return nil, fmt.Errorf("当前前后端 docker-compose 仅支持 Go + React，检测结果为: %s", detected.PrimaryLanguage)
	}

	parts, err := detectGoReactComposeParts(req.LocalPath)
	if err != nil {
		return nil, err
	}

	backendDeployName := composeChildDeployName(parts.BackendPath)
	backend, err := s.AutoCreateDeployProject(AutoCreateDeployProjectRequest{
		LocalPath:          parts.BackendPath,
		ProjectName:        backendDeployName,
		GroupId:            req.GroupId,
		ContainerName:      backendDeployName,
		AllowDuplicatePath: true,
		IncludeRemote:      req.IncludeRemote,
		PreserveDeployName: true,
	})
	if err != nil {
		return nil, fmt.Errorf("生成 Go 后端部署信息失败: %w", err)
	}

	frontendBasePort := req.FrontendDeployPort
	if frontendBasePort == 0 {
		next, err := ProjectServiceApp.GetNextDeployPort("frontend")
		if err != nil {
			return nil, fmt.Errorf("获取前端建议端口失败: %w", err)
		}
		frontendBasePort = next.NextPort
	}

	var frontends []*QuickInitResult
	for index, frontendPart := range parts.Frontends {
		frontendDeployName := composeChildDeployName(frontendPart.Path)
		frontend, err := s.AutoCreateDeployProject(AutoCreateDeployProjectRequest{
			LocalPath:          frontendPart.Path,
			ProjectName:        frontendDeployName,
			GroupId:            req.GroupId,
			ContainerName:      frontendDeployName,
			FrontendDeployPort: frontendBasePort + index,
			AllowDuplicatePath: true,
			IncludeRemote:      req.IncludeRemote,
			PreserveDeployName: true,
		})
		if err != nil {
			return nil, fmt.Errorf("生成 React 前端部署信息失败(%s): %w", frontendPart.Path, err)
		}
		frontends = append(frontends, frontend)
	}
	primaryFrontend := frontends[0]

	projectName := strings.TrimSpace(req.ProjectName)
	if projectName == "" {
		projectName = strings.TrimSpace(detected.ProjectName)
	}
	if projectName == "" {
		projectName = filepath.Base(req.LocalPath)
	}
	if !strings.Contains(strings.ToLower(projectName), "compose") {
		projectName += "-compose"
	}

	result := &QuickInitResult{
		ProjectName:      projectName,
		ComputerLanguage: deployProjectTypeDockerCompose,
		AccessUrl:        primaryFrontend.AccessUrl,
		LocalProjectPath: req.LocalPath,
	}

	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		project := system.TbProject{
			ProjectName:      projectName,
			ComputerLanguage: deployProjectTypeDockerCompose,
			AccessUrl:        primaryFrontend.AccessUrl,
			LocalProjectPath: req.LocalPath,
			GroupId:          req.GroupId,
			UserId:           1,
		}
		if err := tx.Create(&project).Error; err != nil {
			return fmt.Errorf("创建前后端聚合项目失败: %w", err)
		}
		result.ProjectId = project.ID

		frontendProjectIDs := quickInitProjectIDs(frontends)
		if err := s.createGoReactAggregateRoute(tx, project.ID, req.LocalPath, parts, backend.ProjectId, frontendProjectIDs, "frontend_backend_full", "前后端全量部署", "sh deploy-compose-full.sh", "deploy-compose-full.sh", "local_full", "local_full"); err != nil {
			return err
		}
		result.RoutesCreated++

		if err := s.createGoReactAggregateRoute(tx, project.ID, req.LocalPath, parts, backend.ProjectId, frontendProjectIDs, "frontend_backend_incremental", "前后端增量部署", "sh deploy-compose-incremental.sh", "deploy-compose-incremental.sh", "local_incremental", "local_full"); err != nil {
			return err
		}
		result.RoutesCreated++

		var scriptCount int64
		if err := tx.Model(&system.TbProjectScript{}).Where("project_id = ?", project.ID).Count(&scriptCount).Error; err != nil {
			return err
		}
		result.ScriptsCreated = int(scriptCount)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func quickInitProjectIDs(results []*QuickInitResult) []uint {
	ids := make([]uint, 0, len(results))
	for _, result := range results {
		if result != nil {
			ids = append(ids, result.ProjectId)
		}
	}
	return ids
}

func (s *DeployToolService) createGoReactAggregateRoute(tx *gorm.DB, aggregateProjectID uint, rootPath string, parts goReactComposeParts, backendProjectID uint, frontendProjectIDs []uint, routeKey string, routeName string, executeCommand string, scriptName string, backendRouteKey string, frontendRouteKey string) error {
	route := system.TbProjectRoute{
		ProjectId:           int(aggregateProjectID),
		RouteKey:            routeKey,
		RouteName:           routeName,
		ServerId:            0,
		LocalProjectPath:    rootPath,
		LocalExecuteCommand: executeCommand,
		LocalStopCommand:    "docker compose down",
		BuildType:           "docker_compose_deploy",
		Color:               deployRouteColor(routeKey, routeName, "docker_compose_deploy", 0),
	}
	if err := tx.Create(&route).Error; err != nil {
		return fmt.Errorf("创建前后端聚合路线失败: %w", err)
	}

	backendRoute, err := findProjectRoute(tx, backendProjectID, backendRouteKey)
	if err != nil {
		return fmt.Errorf("读取后端路线失败: %w", err)
	}
	frontendCommands := make([]string, 0, len(frontendProjectIDs))
	for _, frontendProjectID := range frontendProjectIDs {
		frontendRoute, err := findProjectRoute(tx, frontendProjectID, frontendRouteKey)
		if err != nil {
			return fmt.Errorf("读取前端路线失败: %w", err)
		}
		frontendCommands = append(frontendCommands, frontendRoute.LocalExecuteCommand)
	}

	content := renderGoReactAggregateScript(parts, backendRoute.LocalExecuteCommand, frontendCommands)
	if err := s.createScriptWithType(tx, aggregateProjectID, route.ID, scriptName, content, 1); err != nil {
		return fmt.Errorf("创建前后端聚合脚本失败: %w", err)
	}
	return nil
}

func detectGoReactComposeParts(root string) (goReactComposeParts, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return goReactComposeParts{}, fmt.Errorf("本地项目目录不能为空")
	}

	backendPath := detectPreferredSubprojectPath(root, "go")
	frontendPaths := detectPreferredSubprojectPaths(root, "react")
	if backendPath == "" || len(frontendPaths) == 0 {
		return goReactComposeParts{}, fmt.Errorf("未能在目录中同时识别 Go 后端和 React 前端")
	}

	backendRel, err := filepath.Rel(root, backendPath)
	if err != nil {
		return goReactComposeParts{}, err
	}
	frontends := make([]goReactComposeFrontendPart, 0, len(frontendPaths))
	for _, frontendPath := range frontendPaths {
		frontendRel, err := filepath.Rel(root, frontendPath)
		if err != nil {
			return goReactComposeParts{}, err
		}
		frontends = append(frontends, goReactComposeFrontendPart{
			Path: frontendPath,
			Rel:  filepath.ToSlash(frontendRel),
		})
	}
	return goReactComposeParts{
		BackendPath: backendPath,
		Frontends:   frontends,
		BackendRel:  filepath.ToSlash(backendRel),
	}, nil
}

func detectPreferredSubprojectPath(root string, kind string) string {
	paths := detectPreferredSubprojectPaths(root, kind)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func detectPreferredSubprojectPaths(root string, kind string) []string {
	var candidates []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && shouldSkipDeployProjectTypeDir(entry.Name()) {
				return filepath.SkipDir
			}
			if rel, relErr := filepath.Rel(root, path); relErr == nil && rel != "." && len(strings.Split(filepath.ToSlash(rel), "/")) > deployProjectTypeScanMaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		switch kind {
		case "go":
			if entry.Name() == "go.mod" {
				candidates = append(candidates, filepath.Dir(path))
			}
		case "react":
			if entry.Name() == "package.json" && packageJSONHasDependency(path, "react") {
				candidates = append(candidates, filepath.Dir(path))
			}
		}
		return nil
	})

	sort.SliceStable(candidates, func(i, j int) bool {
		leftScore := subprojectPathScore(root, candidates[i], kind)
		rightScore := subprojectPathScore(root, candidates[j], kind)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		return candidates[i] < candidates[j]
	})
	return candidates
}

func subprojectPathScore(root string, candidate string, kind string) int {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		rel = candidate
	}
	relSlash := filepath.ToSlash(rel)
	name := strings.ToLower(filepath.Base(candidate))
	score := 0
	if relSlash != "." {
		score += 10
	}
	switch kind {
	case "go":
		if name == "server" || strings.Contains(name, "backend") || strings.Contains(name, "api") {
			score += 40
		}
		if fileExists(filepath.Join(candidate, "main.go")) {
			score += 10
		}
	case "react":
		if name == "web-react" || name == "web" || strings.Contains(name, "frontend") || strings.Contains(name, "ui") {
			score += 40
		}
		if fileExists(filepath.Join(candidate, "src", "main.tsx")) || fileExists(filepath.Join(candidate, "src", "index.tsx")) {
			score += 10
		}
	}
	return score
}

func packageJSONHasDependency(path string, dependency string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]interface{} `json:"dependencies"`
		DevDependencies map[string]interface{} `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	return hasPackageDependency(pkg.Dependencies, pkg.DevDependencies, dependency)
}

func findProjectRoute(tx *gorm.DB, projectID uint, routeKey string) (system.TbProjectRoute, error) {
	var route system.TbProjectRoute
	if err := tx.Where("project_id = ? AND route_key = ?", projectID, routeKey).First(&route).Error; err != nil {
		return route, err
	}
	return route, nil
}

func copyRouteScriptsWithPrefix(tx *gorm.DB, sourceProjectID uint, sourceRouteID uint, targetProjectID uint, targetRouteID uint, prefix string) error {
	var scripts []system.TbProjectScript
	if err := tx.Where("project_id = ? AND route_id = ?", sourceProjectID, sourceRouteID).Find(&scripts).Error; err != nil {
		return err
	}
	for _, script := range scripts {
		fileName := filepath.ToSlash(filepath.Join(prefix, script.FileName))
		copy := system.TbProjectScript{
			ProjectId:    int(targetProjectID),
			RouteId:      int(targetRouteID),
			ScriptType:   script.ScriptType,
			Content:      script.Content,
			FileName:     fileName,
			FileNickName: fileName,
		}
		if err := tx.Create(&copy).Error; err != nil {
			return err
		}
	}
	return nil
}

func renderGoReactAggregateScript(parts goReactComposeParts, backendCommand string, frontendCommands []string) string {
	backendCommand = strings.TrimSpace(backendCommand)
	if backendCommand == "" {
		backendCommand = "docker compose up --build -d"
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`#!/usr/bin/env sh
set -e

ROOT_DIR=$(pwd)

cd "%s"
%s

`, parts.BackendRel, backendCommand))
	for index, frontend := range parts.Frontends {
		frontendCommand := ""
		if index < len(frontendCommands) {
			frontendCommand = strings.TrimSpace(frontendCommands[index])
		}
		if frontendCommand == "" {
			frontendCommand = "docker compose up --build -d"
		}
		builder.WriteString(fmt.Sprintf(`cd "$ROOT_DIR"
cd "%s"
%s

`, frontend.Rel, frontendCommand))
	}
	return builder.String()
}

func (s *DeployToolService) GenerateDeployInfo(req GenerateDeployInfoRequest) (*GenerateDeployInfoResult, error) {
	intent, err := parseDeployInfoIntent(req.Input)
	if err != nil && strings.TrimSpace(req.LocalPath) == "" {
		return nil, err
	}
	intent = mergeStructuredDeployInfoIntent(intent, req)

	detected, err := DetectDeployProjectType(intent.LocalPath)
	if err != nil {
		return nil, err
	}

	projectName := strings.TrimSpace(intent.ProjectName)
	if projectName == "" {
		projectName = strings.TrimSpace(detected.ProjectName)
	}
	if projectName == "" {
		projectName = filepath.Base(intent.LocalPath)
	}

	var existing system.TbProject
	hasExisting := global.GVA_DB.Where("local_project_path = ?", intent.LocalPath).Order("id desc").First(&existing).Error == nil
	allowDuplicatePath := intent.AllowDuplicatePath || hasExisting

	groupName := strings.TrimSpace(intent.GroupName)
	if groupName == "" {
		groupName = s.defaultDeployGroupName(projectName, hasExisting, existing.GroupId)
	}
	if groupName == "" {
		groupName = projectName
	}

	groupAction := normalizeDeployInfoGroupAction(intent.GroupAction)
	if groupAction == "" {
		if intent.UseExistingGroup {
			groupAction = deployInfoGroupActionReuse
		} else if intent.HasGroupName {
			groupAction = deployInfoGroupActionCreate
		}
	}
	if intent.HasGroupName && groupAction == deployInfoGroupActionAuto {
		return nil, fmt.Errorf("项目组意图不明确：无法判断是新建项目组还是复用已有项目组，请在提示词中明确说明“新建项目组”或“使用已有项目组”")
	}

	var group *CreateProjectGroupResult
	createdGroup := false
	if intent.HasGroupName && groupAction == deployInfoGroupActionReuse {
		var existingGroup system.TbProjectGroup
		if err := global.GVA_DB.Where("group_name = ?", groupName).First(&existingGroup).Error; err != nil {
			return nil, fmt.Errorf("项目组「%s」不存在，请检查组名，或改用新组名生成", groupName)
		}
		group = &CreateProjectGroupResult{GroupId: existingGroup.ID, GroupName: existingGroup.GroupName}
	} else if intent.HasGroupName && groupAction == deployInfoGroupActionAuto {
		var existingGroup system.TbProjectGroup
		if err := global.GVA_DB.Where("group_name = ?", groupName).First(&existingGroup).Error; err == nil {
			group = &CreateProjectGroupResult{GroupId: existingGroup.ID, GroupName: existingGroup.GroupName}
		}
	} else if intent.HasGroupName {
		var existingGroup system.TbProjectGroup
		if err := global.GVA_DB.Where("group_name = ?", groupName).First(&existingGroup).Error; err == nil {
			return nil, fmt.Errorf("项目组「%s」已存在，请换一个组名，或明确说明使用已有项目组", groupName)
		}
	} else {
		groupName = nextAvailableProjectGroupName(groupName)
	}

	if group == nil {
		newGroup, err := s.CreateProjectGroup(groupName)
		if err != nil {
			return nil, err
		}
		group = newGroup
		createdGroup = true
	}

	created, err := s.AutoCreateDeployProject(AutoCreateDeployProjectRequest{
		LocalPath:          intent.LocalPath,
		ProjectName:        projectName,
		GroupId:            group.GroupId,
		AppPort:            deployInfoAppPortForLanguage(intent, detected.ProjectType),
		FrontendDeployPort: deployInfoFrontendPortForLanguage(intent, detected.ProjectType),
		AllowDuplicatePath: allowDuplicatePath,
		IncludeRemote:      intent.IncludeRemote,
	})
	if err != nil {
		if createdGroup {
			_ = global.GVA_DB.Where("id = ?", group.GroupId).Unscoped().Delete(&system.TbProjectGroup{}).Error
		}
		return nil, err
	}

	return &GenerateDeployInfoResult{
		ProjectId:          created.ProjectId,
		ProjectName:        created.ProjectName,
		ComputerLanguage:   created.ComputerLanguage,
		AccessUrl:          created.AccessUrl,
		LocalProjectPath:   created.LocalProjectPath,
		GroupId:            group.GroupId,
		GroupName:          group.GroupName,
		AllowDuplicatePath: allowDuplicatePath,
		RoutesCreated:      created.RoutesCreated,
		ScriptsCreated:     created.ScriptsCreated,
	}, nil
}

func mergeStructuredDeployInfoIntent(intent deployInfoIntent, req GenerateDeployInfoRequest) deployInfoIntent {
	if localPath := strings.TrimSpace(req.LocalPath); localPath != "" {
		parsedPath := strings.TrimSpace(intent.LocalPath)
		if parsedPath == "" || deployInfoInputContainsPath(req.Input, localPath) || !deployInfoInputContainsPath(req.Input, parsedPath) {
			intent.LocalPath = localPath
		}
	}
	if groupName := strings.TrimSpace(req.GroupName); groupName != "" {
		intent.GroupName = groupName
		intent.HasGroupName = true
	}
	if groupAction := normalizeDeployInfoGroupAction(req.GroupAction); groupAction != "" {
		if groupAction == deployInfoGroupActionCreate || groupAction == deployInfoGroupActionReuse {
			intent.GroupAction = groupAction
			intent.UseExistingGroup = groupAction == deployInfoGroupActionReuse
		} else if intent.GroupAction == "" {
			intent.GroupAction = groupAction
		}
	}
	if projectName := strings.TrimSpace(req.ProjectName); projectName != "" {
		intent.ProjectName = projectName
		intent.HasProjectName = true
	}
	if req.AppPort > 0 {
		intent.AppPort = req.AppPort
	}
	if req.FrontendDeployPort > 0 {
		intent.FrontendDeployPort = req.FrontendDeployPort
	}
	if req.UseExistingGroup && intent.GroupAction == "" && !hasExplicitNewGroupNamePhrase(req.Input) && (!hasExplicitGroupNamePhrase(req.Input) || hasUseExistingGroupPhrase(req.Input)) {
		intent.UseExistingGroup = true
		intent.GroupAction = deployInfoGroupActionReuse
	}
	if req.AllowDuplicatePath {
		intent.AllowDuplicatePath = true
	}
	if req.IncludeRemote {
		intent.IncludeRemote = true
	}
	return intent
}

func deployInfoInputContainsPath(input string, localPath string) bool {
	input = strings.TrimSpace(input)
	localPath = strings.TrimSpace(localPath)
	if input == "" || localPath == "" {
		return false
	}
	cleanedPath := cleanPathToken(localPath)
	return strings.Contains(input, localPath) || (cleanedPath != "" && strings.Contains(input, cleanedPath))
}

func normalizeDeployInfoGroupAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case deployInfoGroupActionCreate, "new", "create_new":
		return deployInfoGroupActionCreate
	case deployInfoGroupActionReuse, "existing", "use_existing":
		return deployInfoGroupActionReuse
	case deployInfoGroupActionAuto:
		return deployInfoGroupActionAuto
	default:
		return ""
	}
}

func deployInfoAppPortForLanguage(intent deployInfoIntent, projectType string) int {
	if intent.AppPort > 0 {
		return intent.AppPort
	}
	language := normalizeDeployLanguage(projectType)
	if language != "react" && language != "vue" {
		return intent.DeployPort
	}
	return 0
}

func deployInfoFrontendPortForLanguage(intent deployInfoIntent, projectType string) int {
	if intent.FrontendDeployPort > 0 {
		return intent.FrontendDeployPort
	}
	language := normalizeDeployLanguage(projectType)
	if language == "react" || language == "vue" {
		return intent.DeployPort
	}
	return 0
}

func (s *DeployToolService) PreviewReactDeployProject(req PreviewReactDeployProjectRequest) (*PreviewReactDeployProjectResult, error) {
	return s.PreviewFrontendDeployProject(req, "react")
}

func (s *DeployToolService) PreviewFrontendDeployProject(req PreviewFrontendDeployProjectRequest, language string) (*PreviewFrontendDeployProjectResult, error) {
	localPath := strings.TrimSpace(req.LocalProjectPath)
	if localPath == "" {
		return nil, fmt.Errorf("本地项目目录不能为空")
	}

	language = normalizeDeployLanguage(language)
	if language != "react" && language != "vue" {
		return nil, fmt.Errorf("当前仅支持 Vue/React 项目自动生成，收到语言: %s", language)
	}

	detected, err := DetectDeployProjectType(localPath)
	if err != nil {
		return nil, err
	}
	if normalizeDeployLanguage(detected.ProjectType) != language {
		return nil, fmt.Errorf("当前仅支持 %s 项目自动生成，检测结果为: %s", frontendLanguageDisplayName(language), detected.ProjectType)
	}

	projectName := strings.TrimSpace(req.ProjectName)
	if projectName == "" {
		projectName = detected.ProjectName
	}
	if projectName == "" {
		projectName = filepath.Base(localPath)
	}

	frontendPort := req.FrontendDeployPort
	if frontendPort == 0 {
		next, err := ProjectServiceApp.GetNextDeployPort("frontend")
		if err != nil {
			return nil, fmt.Errorf("获取前端建议端口失败: %w", err)
		}
		frontendPort = next.NextPort
	}
	frontendPort, err = nextAvailableDeployPort("frontend", frontendPort)
	if err != nil {
		return nil, fmt.Errorf("获取可用前端端口失败: %w", err)
	}

	backendPort := req.BackendDeployPort
	if backendPort == 0 {
		if groupBackendPort, ok := inferGroupBackendHTTPPort(req.GroupId); ok {
			backendPort = groupBackendPort
		} else {
			next, err := ProjectServiceApp.GetNextDeployPort("backend")
			if err != nil {
				return nil, fmt.Errorf("获取后端建议端口失败: %w", err)
			}
			backendPort = next.NextPort
		}
	}

	deployName := deploymentBaseName(localPath, projectName, language, req.PreserveDeployName)
	containerName := strings.TrimSpace(req.ContainerName)
	if containerName == "" {
		containerName = deployName
	}

	baseCtx := templateContext{
		ProjectName:   deployName,
		ContainerName: containerName,
		AppPort:       backendPort,
		ImageName:     deployImageName(deployName),
		BaseImageName: deployBaseImageName(deployName),
	}
	renderCtx := buildDeployTemplateContext(baseCtx, frontendPort, language)
	renderCtx.HasWebSocket = detectFrontendUsesWebSocket(localPath)
	renderCtx.APIProxyStripPrefix = detectFrontendAPIProxyStripPrefix(localPath)
	renderCtx.ObjectStorageProxyPrefixes = objectStorageProxyPrefixesForGroupDB(global.GVA_DB, req.GroupId)
	packageManager, installCommand, buildCommand, copyLockFileCommand := detectFrontendPackageCommands(localPath)
	renderCtx.PackageManager = packageManager
	renderCtx.InstallCommand = installCommand
	renderCtx.BuildCommand = buildCommand
	renderCtx.CopyLockFileCommand = copyLockFileCommand

	rendered, err := loadDeployTemplate(language, "local-full", renderCtx)
	if err != nil {
		return nil, err
	}

	result := &PreviewFrontendDeployProjectResult{
		ProjectName:        deployName,
		ComputerLanguage:   language,
		LocalProjectPath:   localPath,
		GroupId:            req.GroupId,
		ContainerName:      containerName,
		AccessUrl:          fmt.Sprintf("http://localhost:%d", frontendPort),
		FrontendDeployPort: frontendPort,
		BackendDeployPort:  backendPort,
		PackageManager:     packageManager,
		InstallCommand:     installCommand,
		BuildCommand:       buildCommand,
		Routes: []PreviewDeployRoute{{
			RouteKey:            rendered.Route.RouteKey,
			RouteName:           rendered.Route.RouteName,
			LocalExecuteCommand: rendered.Route.LocalExecuteCommand,
			LocalStopCommand:    rendered.Route.LocalStopCommand,
			BuildType:           rendered.Route.BuildType,
		}},
		ScriptPreview: make(map[string]string, len(rendered.Scripts)),
		Warnings:      detected.Warnings,
	}
	for _, script := range rendered.Scripts {
		result.Scripts = append(result.Scripts, PreviewDeployScript{FileName: script.FileName, Bytes: len(script.Content)})
		result.ScriptPreview[script.FileName] = script.Content
	}
	return result, nil
}

func inferGroupBackendHTTPPort(groupID uint) (int, bool) {
	if groupID == 0 || global.GVA_DB == nil {
		return 0, false
	}

	var projects []system.TbProject
	if err := global.GVA_DB.
		Where("group_id = ? AND computer_language NOT IN ?", groupID, []string{"react", "vue"}).
		Order("id desc").
		Find(&projects).Error; err != nil {
		return 0, false
	}
	for _, project := range projects {
		port, ok := extractAccessURLPort(project.AccessUrl)
		if ok && portMatchesDeployType(port, "backend") {
			return port, true
		}
	}
	return 0, false
}

func (s *DeployToolService) ConfirmReactDeployProject(req PreviewReactDeployProjectRequest) (*QuickInitResult, error) {
	return s.ConfirmFrontendDeployProject(req, "react")
}

func (s *DeployToolService) ConfirmFrontendDeployProject(req PreviewFrontendDeployProjectRequest, language string) (*QuickInitResult, error) {
	preview, err := s.PreviewFrontendDeployProject(req, language)
	if err != nil {
		return nil, err
	}
	return s.QuickInitProject(QuickInitRequest{
		ProjectName:        preview.ProjectName,
		ComputerLanguage:   preview.ComputerLanguage,
		LocalProjectPath:   preview.LocalProjectPath,
		GroupId:            preview.GroupId,
		ContainerName:      preview.ContainerName,
		AppPort:            preview.BackendDeployPort,
		FrontendDeployPort: preview.FrontendDeployPort,
		AllowDuplicatePath: req.AllowDuplicatePath,
		PreserveDeployName: req.PreserveDeployName,
	})
}

// QuickInitProject 一键创建项目 + 路由 + 脚本
func (s *DeployToolService) QuickInitProject(req QuickInitRequest) (*QuickInitResult, error) {
	computerLanguage := normalizeDeployLanguage(req.ComputerLanguage)

	// 检查是否已存在。用户明确要求“再生成一个组名/再来一份”时允许同一路径重复创建部署信息。
	if !req.AllowDuplicatePath {
		var existing system.TbProject
		if err := global.GVA_DB.Where("local_project_path = ?", req.LocalProjectPath).First(&existing).Error; err == nil {
			return nil, fmt.Errorf("项目已存在: %s (ID=%d)", existing.ProjectName, existing.ID)
		}
	}

	deployName := deploymentBaseName(req.LocalProjectPath, req.ProjectName, computerLanguage, req.PreserveDeployName)
	result := &QuickInitResult{ProjectName: deployName}

	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		appPort := req.AppPort
		if appPort == 0 {
			appPort = 8080
		}
		accessPort := appPort
		if computerLanguage == "react" || computerLanguage == "vue" {
			accessPort = normalizeFrontendDeployPort(req.FrontendDeployPort, appPort)
			availableAccessPort, err := nextAvailableDeployPort("frontend", accessPort)
			if err != nil {
				return fmt.Errorf("获取可用前端端口失败: %w", err)
			}
			accessPort = availableAccessPort
			req.FrontendDeployPort = availableAccessPort
		}

		// 1. 创建项目
		project := system.TbProject{
			ProjectName:      deployName,
			ComputerLanguage: computerLanguage,
			Description:      deployProjectDescription(computerLanguage),
			AccessUrl:        fmt.Sprintf("http://localhost:%d", accessPort),
			LocalProjectPath: req.LocalProjectPath,
			GroupId:          req.GroupId,
			UserId:           1,
		}
		if err := tx.Create(&project).Error; err != nil {
			return fmt.Errorf("创建项目失败: %w", err)
		}
		result.ProjectId = project.ID
		result.ProjectName = project.ProjectName
		result.ComputerLanguage = project.ComputerLanguage
		result.AccessUrl = project.AccessUrl
		result.LocalProjectPath = project.LocalProjectPath

		// 2. 根据语言生成路由和脚本
		containerName := req.ContainerName
		if containerName == "" {
			containerName = deployName
		}

		tplCtx := templateContext{
			ProjectName:   deployName,
			ContainerName: containerName,
			AppPort:       appPort,
			ImageName:     deployImageName(deployName),
			BaseImageName: deployBaseImageName(deployName),
		}

		switch computerLanguage {
		case "go":
			renderCtx := buildDeployTemplateContext(tplCtx, req.FrontendDeployPort, "go")
			renderCtx.BackendDeployPort = appPort
			renderCtx.AppPort = detectGoApplicationPort(req.LocalProjectPath, appPort)
			renderCtx.GoConfigCopyCommand = detectGoConfigCopyCommand(req.LocalProjectPath)
			return s.createTemplateDeployConfigs(tx, project.ID, req.LocalProjectPath, result, renderCtx, "go", []string{"local-full", "local-incremental"})
		case "java":
			renderCtx := buildDeployTemplateContext(tplCtx, req.FrontendDeployPort, "java")
			javaConfig := detectJavaDeployConfig(req.LocalProjectPath)
			renderCtx.BackendDeployPort = appPort
			if javaConfig.AppPort > 0 {
				renderCtx.AppPort = javaConfig.AppPort
			} else {
				renderCtx.AppPort = detectJavaApplicationPort(req.LocalProjectPath, 8080)
			}
			applyJavaDeployConfig(&renderCtx, javaConfig)
			renderCtx.HasWebSocket = detectJavaHasWebSocket(req.LocalProjectPath, javaConfig)
			if renderCtx.HasWebSocket {
				renderCtx.WebSocketDeployPort = appPort + 1
			}
			return s.createTemplateDeployConfigs(tx, project.ID, req.LocalProjectPath, result, renderCtx, "java", []string{"local-full", "local-incremental"})
		case "python":
			renderCtx := buildDeployTemplateContext(tplCtx, req.FrontendDeployPort, "python")
			renderCtx.BackendDeployPort = appPort
			renderCtx.AppPort = appPort
			renderCtx.PythonDependencyCopyCommand, renderCtx.PythonDependencyInstallCommand, renderCtx.PythonStartCommand = detectPythonDeployCommands(req.LocalProjectPath)
			deployTypes := []string{"local-full", "local-incremental", "local-compose-full"}
			if req.IncludeRemote {
				deployTypes = append(deployTypes, "remote-incremental")
			}
			return s.createTemplateDeployConfigs(tx, project.ID, req.LocalProjectPath, result, renderCtx, "python", deployTypes)
		case "react":
			return s.createReactDeployConfig(tx, project.ID, req.LocalProjectPath, req.GroupId, tplCtx, req.FrontendDeployPort, result)
		case "vue":
			return s.createVueDeployConfig(tx, project.ID, req.LocalProjectPath, req.GroupId, tplCtx, req.FrontendDeployPort, result)
		default:
			return fmt.Errorf("%s 暂未接入部署模板", req.ComputerLanguage)
		}
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *DeployToolService) CreateProjectGroup(groupName string) (*CreateProjectGroupResult, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return nil, fmt.Errorf("项目组名不能为空")
	}

	group, err := ProjectGroupServiceApp.SaveOrUpdateGroup(system.TbProjectGroup{
		GroupName: groupName,
		UserId:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("创建项目组失败: %w", err)
	}
	return &CreateProjectGroupResult{GroupId: group.ID, GroupName: group.GroupName}, nil
}

func parseDeployInfoIntent(input string) (deployInfoIntent, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return deployInfoIntent{}, fmt.Errorf("请输入项目目录")
	}

	path := extractLocalPath(input)
	if path == "" {
		return deployInfoIntent{}, fmt.Errorf("没有识别到有效项目目录，请输入本地项目绝对路径")
	}

	intent := deployInfoIntent{
		LocalPath:          path,
		DeployPort:         extractExplicitDeployPort(input),
		AllowDuplicatePath: containsAny(input, []string{"再生成", "再来一份", "重新生成", "已经有", "已存在", "compare", "对比"}),
		IncludeRemote:      containsAny(input, []string{"远程", "远端", "服务器部署", "部署到服务器", "上传服务器", "上传到服务器", "ssh", "sftp"}),
	}

	if groupName := extractNewGroupName(input); groupName != "" {
		intent.GroupName = groupName
		intent.GroupAction = deployInfoGroupActionCreate
		intent.HasGroupName = true
	} else if groupName := extractLabeledValue(input, []string{"组名", "项目组", "组"}); groupName != "" {
		intent.GroupName = groupName
		intent.HasGroupName = true
		if hasUseExistingShortGroupPhrase(input) || hasUseExistingGroupPhrase(input) {
			intent.UseExistingGroup = true
			intent.GroupAction = deployInfoGroupActionReuse
		} else {
			intent.GroupAction = deployInfoGroupActionCreate
		}
	} else if groupName := extractAdjacentGroupName(input); groupName != "" {
		intent.GroupName = groupName
		intent.GroupAction = deployInfoGroupActionCreate
		intent.HasGroupName = true
	} else if groupName := extractPlacementGroupName(input, path); groupName != "" {
		intent.GroupName = groupName
		intent.HasGroupName = true
		if hasGenericNewGroupPhrase(input) {
			intent.GroupAction = deployInfoGroupActionCreate
		} else {
			intent.UseExistingGroup = true
			intent.GroupAction = deployInfoGroupActionReuse
		}
	} else if groupName := extractInPhraseGroupName(input, path); groupName != "" {
		intent.GroupName = groupName
		intent.HasGroupName = true
		intent.UseExistingGroup = true
		intent.GroupAction = deployInfoGroupActionReuse
	}
	if projectName := extractLabeledValue(input, []string{"项目名", "服务名"}); projectName != "" {
		intent.ProjectName = projectName
		intent.HasProjectName = true
	}

	return intent, nil
}

func extractLocalPath(input string) string {
	re := regexp.MustCompile(`/(?:[^\s，,。；;` + "`" + `"'的为]+)`)
	match := re.FindStringSubmatch(input)
	if len(match) < 1 {
		return ""
	}
	return cleanPathToken(match[0])
}

func cleanPathToken(path string) string {
	path = strings.Trim(strings.TrimRight(strings.TrimSpace(path), "，,。；;"), "`\"'")
	for _, marker := range []string{"这个目录", "该目录", "此目录", "以上目录", "以上这个目录", "目录生成", "目录创建"} {
		if idx := strings.Index(path, marker); idx >= 0 {
			path = path[:idx]
		}
	}
	return strings.Trim(strings.TrimRight(strings.TrimSpace(path), "，,。；;"), "`\"'")
}

func extractExplicitDeployPort(input string) int {
	patterns := []string{
		`(?:^|[\s，,。；;])(?:端口号|端口|port)\s*(?:用|使用|为|是|:|：|=)?\s*([0-9]{2,5})`,
		`(?:^|[\s，,。；;])(?:用|使用|为|是)\s*([0-9]{2,5})\s*(?:这个)?(?:端口号|端口|port)`,
		`(?:^|[\s，,。；;])([0-9]{2,5})\s*(?:这个)?(?:端口号|端口|port)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if match := re.FindStringSubmatch(input); len(match) >= 2 {
			return parseValidPort(match[1])
		}
	}
	return 0
}

func extractRootMavenArtifactID(content string) string {
	parentRe := regexp.MustCompile(`(?s)<parent\b[^>]*>.*?</parent>`)
	content = parentRe.ReplaceAllString(content, "")
	artifactRe := regexp.MustCompile(`(?s)<artifactId>\s*([^<]+?)\s*</artifactId>`)
	if match := artifactRe.FindStringSubmatch(content); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractLabeledValue(input string, labels []string) string {
	for _, label := range labels {
		pattern := fmt.Sprintf(`%s\s*(?:[:：]|为)\s*([^\s，,。；;]+)`, regexp.QuoteMeta(label))
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(input); len(match) >= 2 {
			return cleanGroupNameValue(match[1])
		}
	}
	return ""
}

func extractAdjacentGroupName(input string) string {
	re := regexp.MustCompile(`组名\s*([^\s，,。；;]+)`)
	if match := re.FindStringSubmatch(input); len(match) >= 2 {
		return cleanGroupNameValue(match[1])
	}
	return ""
}

func extractNewGroupName(input string) string {
	patterns := []string{
		`(?:新建|创建)\s*(?:组名|项目组|组)\s*(?:[:：]|为)?\s*([^\s，,。；;]+)`,
		`(?:新建|创建)\s*([^\s，,。；;]+?组)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(input); len(match) >= 2 {
			groupName := cleanGroupNameValue(match[1])
			if isGenericGroupPlaceholder(groupName) {
				continue
			}
			return groupName
		}
	}
	return ""
}

func cleanGroupNameValue(groupName string) string {
	groupName = strings.TrimSpace(groupName)
	groupName = strings.TrimSuffix(groupName, "下")
	groupName = strings.TrimSuffix(groupName, "里")
	groupName = strings.TrimSuffix(groupName, "中")
	return strings.TrimSpace(groupName)
}

func isGenericGroupPlaceholder(groupName string) bool {
	groupName = strings.TrimSpace(groupName)
	switch groupName {
	case "一个组", "1个组", "新的组", "新组", "一个项目组", "1个项目组", "新的项目组", "新项目组":
		return true
	default:
		return false
	}
}

func hasExplicitNewGroupNamePhrase(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	re := regexp.MustCompile(`(?:新建|创建)\s*(?:组名|项目组|组)\s*(?:[:：]|为)?\s*[^\s，,。；;]+|(?:新建|创建)\s*[^\s，,。；;]+?组`)
	return re.MatchString(input)
}

func hasGenericNewGroupPhrase(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	re := regexp.MustCompile(`(?:新建|创建)\s*(?:一个|1个|新的)?\s*(?:项目组|组)`)
	return re.MatchString(input)
}

func hasExplicitGroupNamePhrase(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	re := regexp.MustCompile(`(?:组名|项目组)\s*(?:[:：]|为)\s*[^\s，,。；;]+`)
	return re.MatchString(input)
}

func hasUseExistingShortGroupPhrase(input string) bool {
	re := regexp.MustCompile(`(?:^|[\s，,。；;])组\s*(?:[:：]|为)\s*[^\s，,。；;]+`)
	return re.MatchString(strings.TrimSpace(input))
}

func hasUseExistingGroupPhrase(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	placementRe := regexp.MustCompile(`(?:放在|放到|放入|加入到|加入|添加到)\s*(?:已有|现有)?\s*(?:组名|项目组|组)?\s*(?:[:：]|为)?\s*[^\s，,。；;]+(?:下|里|中)?`)
	reuseRe := regexp.MustCompile(`(?:复用|使用)\s*(?:已有|现有)?\s*(?:组名|项目组|组)`)
	return placementRe.MatchString(input) || reuseRe.MatchString(input)
}

func extractPlacementGroupName(input string, path string) string {
	candidates := []string{input}
	if idx := strings.Index(input, path); idx >= 0 {
		candidates = []string{input[idx+len(path):], input[:idx]}
	}
	for _, text := range candidates {
		if groupName := extractGroupNameFromPlacementText(text); groupName != "" {
			return groupName
		}
	}
	return ""
}

func extractGroupNameFromPlacementText(text string) string {
	re := regexp.MustCompile(`(?:放在|放到|放入|加入到|加入|添加到)\s*(?:已有|现有)?\s*(?:组名|项目组|组)?\s*(?:[:：]|为)?\s*([^\s，,。；;]+?)(?:下|里|中)`)
	if match := re.FindStringSubmatch(text); len(match) >= 2 {
		groupName := cleanGroupNameValue(match[1])
		if isConcreteGroupName(groupName) {
			return groupName
		}
	}
	return ""
}

func isConcreteGroupName(groupName string) bool {
	groupName = strings.TrimSpace(groupName)
	return groupName != "" && groupName != "这个组" && groupName != "该组" && groupName != "此组" && groupName != "这个项目组" && groupName != "该项目组"
}

func extractInPhraseGroupName(input string, path string) string {
	prefix := input
	if idx := strings.Index(input, path); idx >= 0 {
		prefix = input[:idx]
	}
	if groupName := extractGroupNameFromInPhraseText(prefix); groupName != "" {
		return groupName
	}
	if idx := strings.Index(input, path); idx >= 0 {
		suffix := input[idx+len(path):]
		if groupName := extractGroupNameFromInPhraseText(suffix); groupName != "" {
			return groupName
		}
	}
	return ""
}

func extractGroupNameFromInPhraseText(text string) string {
	re := regexp.MustCompile(`在\s*([^\s，,。；;]+?)\s*(?:里|中|下)`)
	if match := re.FindStringSubmatch(text); len(match) >= 2 {
		groupName := strings.TrimSpace(match[1])
		groupName = strings.TrimSuffix(groupName, "生成")
		if groupName != "" && !strings.HasPrefix(groupName, "/") {
			return groupName
		}
	}
	return ""
}

func containsAny(input string, needles []string) bool {
	lower := strings.ToLower(input)
	for _, needle := range needles {
		if strings.Contains(lower, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func (s *DeployToolService) defaultDeployGroupName(projectName string, hasExisting bool, existingGroupId uint) string {
	base := strings.TrimSpace(projectName)
	if hasExisting && existingGroupId != 0 {
		var group system.TbProjectGroup
		if err := global.GVA_DB.First(&group, existingGroupId).Error; err == nil && strings.TrimSpace(group.GroupName) != "" {
			base = group.GroupName
		}
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = "默认项目组"
	}
	if hasExisting && !strings.Contains(strings.ToLower(base), "compare") && !strings.Contains(base, "对比") {
		base += "_compare"
	}
	return base
}

func nextAvailableProjectGroupName(baseName string) string {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		baseName = "默认项目组"
	}

	var count int64
	if err := global.GVA_DB.Model(&system.TbProjectGroup{}).Where("group_name = ?", baseName).Count(&count).Error; err != nil || count == 0 {
		return baseName
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s_%d", baseName, i)
		count = 0
		if err := global.GVA_DB.Model(&system.TbProjectGroup{}).Where("group_name = ?", candidate).Count(&count).Error; err != nil || count == 0 {
			return candidate
		}
	}
	return fmt.Sprintf("%s_%d", baseName, 1000)
}

func normalizeDeployLanguage(language string) string {
	normalized := strings.ToLower(strings.TrimSpace(language))
	switch normalized {
	case "react", "vue", "go", "java", "python", "docker-compose":
		return normalized
	case "前后端 docker-compose":
		return "docker-compose"
	default:
		return normalized
	}
}

func frontendLanguageDisplayName(language string) string {
	switch normalizeDeployLanguage(language) {
	case "vue":
		return "Vue"
	case "react":
		return "React"
	default:
		return language
	}
}

// ─── Tool 3: list_projects ─────────────────────────────────────────────

// ListAllProjects 获取所有项目列表
func (s *DeployToolService) ListAllProjects() ([]system.TbProject, error) {
	var projects []system.TbProject
	err := global.GVA_DB.Preload("Routes").Order("id desc").Find(&projects).Error
	return projects, err
}

// ─── 内部模板方法 ──────────────────────────────────────────────────────

type templateContext struct {
	ProjectName   string
	ContainerName string
	AppPort       int
	ImageName     string
	BaseImageName string
}

const (
	localFullRouteColor        = "bg-emerald-100 text-emerald-700 hover:bg-emerald-200"
	localIncrementalRouteColor = "bg-sky-100 text-sky-700 hover:bg-sky-200"
	remoteRouteColor           = "bg-purple-100 text-purple-700 hover:bg-purple-200"
	dependencyRouteColor       = "bg-amber-100 text-amber-800 hover:bg-amber-200"
)

func deployRouteColor(routeKey, routeName, buildType string, serverId int) string {
	key := strings.ToLower(strings.TrimSpace(routeKey))
	name := strings.TrimSpace(routeName)
	build := strings.ToLower(strings.TrimSpace(buildType))

	if serverId != 0 || strings.Contains(key, "remote") || strings.Contains(name, "远程") {
		return remoteRouteColor
	}
	if strings.Contains(name, "依赖增量") {
		return dependencyRouteColor
	}
	if strings.Contains(key, "incremental") || strings.Contains(name, "增量") {
		return localIncrementalRouteColor
	}
	if strings.Contains(key, "full") || strings.Contains(name, "全量") {
		return localFullRouteColor
	}
	if strings.Contains(key, "compose") || strings.Contains(strings.ToLower(name), "compose") || build == "docker_compose_deploy" {
		return localIncrementalRouteColor
	}
	return localFullRouteColor
}

func (s *DeployToolService) createRoute(tx *gorm.DB, projectId uint, localPath, routeKey, routeName, executeCmd, buildType string) (uint, error) {
	route := system.TbProjectRoute{
		ProjectId:           int(projectId),
		RouteKey:            routeKey,
		RouteName:           routeName,
		ServerId:            0,
		LocalProjectPath:    localPath,
		LocalExecuteCommand: executeCmd,
		BuildType:           buildType,
		Color:               deployRouteColor(routeKey, routeName, buildType, 0),
	}
	if err := tx.Create(&route).Error; err != nil {
		return 0, err
	}
	return route.ID, nil
}

func (s *DeployToolService) createScriptWithType(tx *gorm.DB, projectId uint, routeId uint, fileName, content string, scriptType uint) error {
	script := system.TbProjectScript{
		ProjectId:    int(projectId),
		RouteId:      int(routeId),
		ScriptType:   scriptType,
		FileName:     fileName,
		FileNickName: fileName,
		Content:      content,
	}
	return tx.Create(&script).Error
}

func (s *DeployToolService) createTemplateDeployConfigs(tx *gorm.DB, projectId uint, localPath string, result *QuickInitResult, ctx deployTemplateContext, language string, deployTypes []string) error {
	for _, deployType := range deployTypes {
		rendered, err := loadDeployTemplate(language, deployType, ctx)
		if err != nil {
			return err
		}

		routeId, err := s.createRoute(tx, projectId, localPath, rendered.Route.RouteKey, rendered.Route.RouteName, rendered.Route.LocalExecuteCommand, rendered.Route.BuildType)
		if err != nil {
			return err
		}
		if rendered.Route.LocalStopCommand != "" {
			if err := tx.Model(&system.TbProjectRoute{}).Where("id = ?", routeId).Update("local_stop_command", rendered.Route.LocalStopCommand).Error; err != nil {
				return err
			}
		}
		routeUpdates := map[string]interface{}{}
		if rendered.Route.ServerExecuteCommand != "" {
			routeUpdates["server_execute_command"] = rendered.Route.ServerExecuteCommand
		}
		if rendered.Route.FileName != "" {
			routeUpdates["file_name"] = rendered.Route.FileName
		}
		if len(routeUpdates) > 0 {
			if err := tx.Model(&system.TbProjectRoute{}).Where("id = ?", routeId).Updates(routeUpdates).Error; err != nil {
				return err
			}
		}
		result.RoutesCreated++

		for _, script := range rendered.Scripts {
			if strings.ToLower(language) == "python" && script.FileName == "docker-compose.yml" {
				script.Content = normalizePythonComposeForDeploy(script.Content, ctx.AppPort, localPath)
			}
			if err := s.createScriptWithType(tx, projectId, routeId, script.FileName, script.Content, rendered.Route.ScriptType); err != nil {
				return err
			}
			result.ScriptsCreated++
		}
	}
	return nil
}

func buildDeployTemplateContext(ctx templateContext, frontendDeployPort int, language string) deployTemplateContext {
	packageManager, installCommand, buildCommand, copyLockFileCommand := detectFrontendPackageCommands("")
	renderCtx := deployTemplateContext{
		ProjectName:                    ctx.ProjectName,
		ContainerName:                  ctx.ContainerName,
		ImageName:                      ctx.ImageName,
		BaseImageName:                  ctx.BaseImageName,
		AppPort:                        ctx.AppPort,
		FrontendDeployPort:             normalizeFrontendDeployPort(frontendDeployPort, ctx.AppPort),
		BackendDeployPort:              ctx.AppPort,
		WebSocketDeployPort:            ctx.AppPort + 1,
		WebSocketPort:                  9090,
		HasWebSocket:                   false,
		DatabaseName:                   sanitizeDeployIdentifier(ctx.ProjectName),
		DatabaseUsername:               "conchi",
		DatabasePassword:               "conchi123456",
		RedisHost:                      "host.docker.internal",
		RedisPort:                      6379,
		RedisPassword:                  "conchi123456",
		PackageManager:                 packageManager,
		InstallCommand:                 installCommand,
		BuildCommand:                   buildCommand,
		DistDir:                        "dist",
		NodeVersion:                    "20",
		GoVersion:                      "1.26",
		GoConfigCopyCommand:            "COPY config.yaml config.yaml",
		JavaVersion:                    "21",
		PythonVersion:                  "3.11",
		CopyLockFileCommand:            copyLockFileCommand,
		PythonDependencyCopyCommand:    "# no Python dependency file found",
		PythonDependencyInstallCommand: "RUN python -m pip install --upgrade pip",
		PythonStartCommand:             `CMD ["python", "main.py"]`,
	}
	return renderCtx
}

func deployImageName(deployName string) string {
	return fmt.Sprintf("%s:%s", deployName, deployImageVersion)
}

func deployBaseImageName(deployName string) string {
	return deployImageName(deployName)
}

func deploymentBaseName(localPath string, projectName string, language string, preserveName bool) string {
	baseName := strings.TrimSpace(projectName)
	cleanPath := strings.TrimSpace(localPath)
	if preserveName {
		if baseName == "" && cleanPath != "" {
			baseName = filepath.Base(filepath.Clean(cleanPath))
		}
		return normalizeDockerComposeServiceIdentifier(baseName)
	}
	if cleanPath != "" {
		currentDir := filepath.Base(filepath.Clean(cleanPath))
		if isDeployRoleDir(currentDir, language) {
			parentDir := filepath.Base(filepath.Dir(filepath.Clean(cleanPath)))
			if strings.TrimSpace(parentDir) != "" && parentDir != "." && parentDir != string(filepath.Separator) {
				baseName = parentDir
			}
		}
	}

	normalized := normalizeDockerIdentifier(baseName)
	if language == "vue" || language == "react" {
		if !strings.HasSuffix(normalized, "-web") {
			normalized += "-web"
		}
	}
	return normalized
}

func composeChildDeployName(localPath string) string {
	return normalizeDockerComposeServiceIdentifier(filepath.Base(filepath.Clean(strings.TrimSpace(localPath))))
}

func normalizeDockerComposeServiceIdentifier(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || normalized == "." {
		return "app"
	}

	var builder strings.Builder
	lastSeparator := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
			lastSeparator = false
			continue
		}
		if r == '-' || r == '.' {
			if !lastSeparator {
				builder.WriteRune(r)
				lastSeparator = true
			}
			continue
		}
		if !lastSeparator {
			builder.WriteRune('-')
			lastSeparator = true
		}
	}
	result := strings.Trim(builder.String(), "-.")
	if result == "" {
		return "app"
	}
	return result
}

func deployProjectDescription(language string) string {
	switch language {
	case "vue", "react":
		return "前端项目"
	default:
		return "后端项目"
	}
}

func isDeployRoleDir(dir string, language string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dir))
	switch language {
	case "vue", "react":
		switch normalized {
		case "web", "frontend", "front", "ui", "client":
			return true
		}
	default:
		switch normalized {
		case "server", "backend", "back", "api", "service":
			return true
		}
	}
	return false
}

func normalizeDockerIdentifier(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "app"
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "app"
	}
	return result
}

type javaDeployConfig struct {
	AppPort          int
	WebSocketPort    int
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	RedisHost        string
	RedisPort        int
	RedisPassword    string
}

func applyJavaDeployConfig(ctx *deployTemplateContext, config javaDeployConfig) {
	if config.DatabaseName != "" {
		ctx.DatabaseName = config.DatabaseName
	}
	if config.DatabaseUsername != "" {
		ctx.DatabaseUsername = config.DatabaseUsername
	}
	if config.DatabasePassword != "" {
		ctx.DatabasePassword = config.DatabasePassword
	}
	if config.RedisHost != "" {
		ctx.RedisHost = normalizeHostForContainer(config.RedisHost)
	}
	if config.RedisPort > 0 {
		ctx.RedisPort = config.RedisPort
	}
	if config.RedisPassword != "" {
		ctx.RedisPassword = config.RedisPassword
	}
	if config.WebSocketPort > 0 {
		ctx.WebSocketPort = config.WebSocketPort
	}
}

func detectJavaHasWebSocket(localPath string, config javaDeployConfig) bool {
	if config.WebSocketPort > 0 {
		return true
	}
	return directoryContainsAny(localPath, []string{
		"WebSocketServerProtocolHandler",
		"@ServerEndpoint",
		"@EnableWebSocket",
		"WebSocketConfigurer",
		"TextWebSocketFrame",
		"websocketx",
		"NettyWebSocket",
	})
}

func detectFrontendUsesWebSocket(localPath string) bool {
	return directoryContainsAny(localPath, []string{
		"new WebSocket",
		"WebSocket(",
		"/ws",
		"ws://",
		"wss://",
		"socket.io",
		"SockJS",
		"stomp",
	})
}

func detectFrontendAPIProxyStripPrefix(localPath string) bool {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return false
	}

	candidates := []string{
		filepath.Join(localPath, "vite.config.js"),
		filepath.Join(localPath, "vite.config.ts"),
		filepath.Join(localPath, "vite.config.mjs"),
		filepath.Join(localPath, "vite.config.mts"),
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		content := strings.ReplaceAll(string(data), " ", "")
		content = strings.ReplaceAll(content, "\n", "")
		content = strings.ReplaceAll(content, "\t", "")
		if strings.Contains(content, "replace(/^\\/api/,'')") ||
			strings.Contains(content, `replace(/^\/api/,"")`) ||
			strings.Contains(content, "replace(/^\\/api/,\"\")") {
			return true
		}
	}
	return false
}

func directoryContainsAny(root string, needles []string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return false
	}
	allowedExt := map[string]bool{
		".java": true, ".kt": true, ".xml": true, ".yml": true, ".yaml": true, ".properties": true,
		".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".vue": true,
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "dist", "build", "target", ".idea", ".vscode":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !allowedExt[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		for _, needle := range needles {
			if strings.Contains(content, needle) {
				found = true
				break
			}
		}
		return nil
	})
	return found
}

func detectJavaDeployConfig(localPath string) javaDeployConfig {
	config := javaDeployConfig{}
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return config
	}

	for _, candidate := range javaApplicationConfigCandidates(localPath) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		config = mergeJavaDeployConfig(config, parseJavaDeployConfigContent(string(data)))
	}
	return config
}

func javaApplicationConfigCandidates(localPath string) []string {
	resourceDir := filepath.Join(localPath, "src", "main", "resources")
	names := []string{
		"application.yml",
		"application.yaml",
		"application.properties",
		"application-dev.yml",
		"application-dev.yaml",
		"application-prod.yml",
		"application-prod.yaml",
	}
	candidates := make([]string, 0, len(names))
	for _, name := range names {
		candidates = append(candidates, filepath.Join(resourceDir, name))
	}
	return candidates
}

func mergeJavaDeployConfig(base, next javaDeployConfig) javaDeployConfig {
	if next.AppPort > 0 {
		base.AppPort = next.AppPort
	}
	if next.WebSocketPort > 0 {
		base.WebSocketPort = next.WebSocketPort
	}
	if next.DatabaseName != "" {
		base.DatabaseName = next.DatabaseName
	}
	if next.DatabaseUsername != "" {
		base.DatabaseUsername = next.DatabaseUsername
	}
	if next.DatabasePassword != "" {
		base.DatabasePassword = next.DatabasePassword
	}
	if next.RedisHost != "" {
		base.RedisHost = next.RedisHost
	}
	if next.RedisPort > 0 {
		base.RedisPort = next.RedisPort
	}
	if next.RedisPassword != "" {
		base.RedisPassword = next.RedisPassword
	}
	return base
}

func parseJavaDeployConfigContent(content string) javaDeployConfig {
	config := javaDeployConfig{
		AppPort: parseJavaApplicationPort(content),
	}
	config.WebSocketPort = parseNestedIntValue(content, []string{"netty", "websocket"}, "port")
	if config.WebSocketPort == 0 {
		config.WebSocketPort = parsePropertiesIntValue(content, "netty.websocket.port")
	}

	datasourceURL := parseNestedStringValue(content, []string{"spring", "datasource"}, "url")
	if datasourceURL == "" {
		datasourceURL = parsePropertiesStringValue(content, "spring.datasource.url")
	}
	config.DatabaseName = parseDatabaseNameFromJDBCURL(datasourceURL)
	config.DatabaseUsername = parseNestedStringValue(content, []string{"spring", "datasource"}, "username")
	if config.DatabaseUsername == "" {
		config.DatabaseUsername = parsePropertiesStringValue(content, "spring.datasource.username")
	}
	config.DatabasePassword = parseNestedStringValue(content, []string{"spring", "datasource"}, "password")
	if config.DatabasePassword == "" {
		config.DatabasePassword = parsePropertiesStringValue(content, "spring.datasource.password")
	}

	config.RedisHost = parseNestedStringValue(content, []string{"spring", "data", "redis"}, "host")
	if config.RedisHost == "" {
		config.RedisHost = parseNestedStringValue(content, []string{"spring", "redis"}, "host")
	}
	if config.RedisHost == "" {
		config.RedisHost = parsePropertiesStringValue(content, "spring.data.redis.host")
	}
	if config.RedisHost == "" {
		config.RedisHost = parsePropertiesStringValue(content, "spring.redis.host")
	}
	config.RedisPort = parseNestedIntValue(content, []string{"spring", "data", "redis"}, "port")
	if config.RedisPort == 0 {
		config.RedisPort = parseNestedIntValue(content, []string{"spring", "redis"}, "port")
	}
	if config.RedisPort == 0 {
		config.RedisPort = parsePropertiesIntValue(content, "spring.data.redis.port")
	}
	if config.RedisPort == 0 {
		config.RedisPort = parsePropertiesIntValue(content, "spring.redis.port")
	}
	config.RedisPassword = parseNestedStringValue(content, []string{"spring", "data", "redis"}, "password")
	if config.RedisPassword == "" {
		config.RedisPassword = parseNestedStringValue(content, []string{"spring", "redis"}, "password")
	}
	if config.RedisPassword == "" {
		config.RedisPassword = parsePropertiesStringValue(content, "spring.data.redis.password")
	}
	if config.RedisPassword == "" {
		config.RedisPassword = parsePropertiesStringValue(content, "spring.redis.password")
	}
	return config
}

func parseNestedStringValue(content string, path []string, key string) string {
	lines := strings.Split(content, "\n")
	pathIndex := 0
	indents := make([]int, 0, len(path))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		for len(indents) > 0 && indent <= indents[len(indents)-1] {
			indents = indents[:len(indents)-1]
			pathIndex = len(indents)
		}
		if pathIndex < len(path) && strings.HasPrefix(trimmed, path[pathIndex]+":") {
			indents = append(indents, indent)
			pathIndex++
			continue
		}
		if pathIndex == len(path) && strings.HasPrefix(trimmed, key+":") {
			return cleanConfigValue(strings.TrimSpace(strings.TrimPrefix(trimmed, key+":")))
		}
	}
	return ""
}

func parseNestedIntValue(content string, path []string, key string) int {
	return parseValidPort(parseNestedStringValue(content, path, key))
}

func parsePropertiesStringValue(content string, key string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*(.*?)\s*$`)
	if match := re.FindStringSubmatch(content); len(match) == 2 {
		return cleanConfigValue(match[1])
	}
	return ""
}

func parsePropertiesIntValue(content string, key string) int {
	return parseValidPort(parsePropertiesStringValue(content, key))
}

func cleanConfigValue(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return strings.Trim(value, `"'`)
}

func parseDatabaseNameFromJDBCURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	re := regexp.MustCompile(`^jdbc:[^:]+://[^/]+/([^?;]+)`)
	if match := re.FindStringSubmatch(raw); len(match) >= 2 {
		return sanitizeDeployIdentifier(match[1])
	}
	return ""
}

func normalizeHostForContainer(host string) string {
	host = strings.TrimSpace(host)
	if host == "127.0.0.1" || host == "localhost" || host == "0.0.0.0" {
		return "host.docker.internal"
	}
	return host
}

func detectJavaApplicationPort(localPath string, fallback int) int {
	if fallback <= 0 {
		fallback = 8080
	}
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return fallback
	}

	for _, candidate := range javaApplicationConfigCandidates(localPath) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		if port := parseJavaApplicationPort(string(data)); port > 0 {
			return port
		}
	}
	return fallback
}

func detectGoApplicationPort(localPath string, fallback int) int {
	if fallback <= 0 {
		fallback = 8080
	}
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return fallback
	}

	candidates := []string{
		filepath.Join(localPath, "config.docker.yaml"),
		filepath.Join(localPath, "config.docker.yml"),
		filepath.Join(localPath, "config.yaml"),
		filepath.Join(localPath, "config.yml"),
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		if port := parseGoApplicationPort(string(data)); port > 0 {
			return port
		}
	}
	return fallback
}

func detectGoConfigCopyCommand(localPath string) string {
	localPath = strings.TrimSpace(localPath)
	if localPath != "" {
		for _, candidate := range []string{"config.docker.yaml", "config.docker.yml"} {
			if fileExists(filepath.Join(localPath, candidate)) {
				return fmt.Sprintf("COPY %s config.yaml", candidate)
			}
		}
	}
	return "COPY config.yaml config.yaml"
}

func parseGoApplicationPort(content string) int {
	lines := strings.Split(content, "\n")
	inSystemBlock := false
	systemIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if strings.HasPrefix(trimmed, "system:") {
			inSystemBlock = true
			systemIndent = indent
			continue
		}
		if inSystemBlock && indent <= systemIndent {
			inSystemBlock = false
		}
		if inSystemBlock && strings.HasPrefix(trimmed, "addr:") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "addr:"))
			return parseValidPort(strings.Trim(value, `"'`))
		}
	}
	return 0
}

func detectPythonDeployCommands(localPath string) (copyCommand string, installCommand string, startCommand string) {
	localPath = strings.TrimSpace(localPath)
	startCommand = `CMD ["python", "main.py"]`
	if localPath != "" {
		switch {
		case fileExists(filepath.Join(localPath, "main.py")):
			startCommand = `CMD ["python", "main.py"]`
		case fileExists(filepath.Join(localPath, "app.py")):
			startCommand = `CMD ["python", "app.py"]`
		case fileExists(filepath.Join(localPath, "manage.py")):
			startCommand = `CMD ["python", "manage.py"]`
		}
	}

	switch {
	default:
		return "COPY requirements.txt ./",
			"RUN python -m pip install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple",
			startCommand
	}
}

func detectSnailJobPythonProject(localPath string) bool {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return false
	}
	if info, err := os.Stat(filepath.Join(localPath, "snailjob")); err == nil && info.IsDir() {
		return true
	}
	for _, fileName := range []string{"main.py", "app.py", "manage.py", "requirements.txt", "pyproject.toml", ".env"} {
		if fileContainsAny(filepath.Join(localPath, fileName), []string{
			"snailjob",
			"snail-job-python",
			"SNAIL_SERVER_HOST",
			"SNAIL_HOST_IP",
		}) {
			return true
		}
	}
	return false
}

func fileContainsAny(path string, needles []string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	for _, needle := range needles {
		if strings.Contains(content, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func parseJavaApplicationPort(content string) int {
	propertiesPattern := regexp.MustCompile(`(?m)^\s*server\.port\s*=\s*([0-9]{1,5})\s*$`)
	if matches := propertiesPattern.FindStringSubmatch(content); len(matches) == 2 {
		return parseValidPort(matches[1])
	}

	lines := strings.Split(content, "\n")
	inServerBlock := false
	serverIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if strings.HasPrefix(trimmed, "server:") {
			inServerBlock = true
			serverIndent = indent
			continue
		}
		if inServerBlock && indent <= serverIndent {
			inServerBlock = false
		}
		if inServerBlock && strings.HasPrefix(trimmed, "port:") {
			return parseValidPort(strings.TrimSpace(strings.TrimPrefix(trimmed, "port:")))
		}
	}
	return 0
}

func parseValidPort(value string) int {
	var port int
	if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
		return 0
	}
	if port <= 0 || port > 65535 {
		return 0
	}
	return port
}

func sanitizeDeployIdentifier(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "app"
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteRune('_')
			lastUnderscore = true
		}
	}

	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "app"
	}
	return result
}

// ─── React 前端部署配置 ─────────────────────────────────────────────────

func (s *DeployToolService) createReactDeployConfig(tx *gorm.DB, projectId uint, localPath string, groupID uint, ctx templateContext, frontendDeployPort int, result *QuickInitResult) error {
	renderCtx := buildDeployTemplateContext(ctx, frontendDeployPort, "react")
	packageManager, installCommand, buildCommand, copyLockFileCommand := detectFrontendPackageCommands(localPath)
	renderCtx.PackageManager = packageManager
	renderCtx.InstallCommand = installCommand
	renderCtx.BuildCommand = buildCommand
	renderCtx.CopyLockFileCommand = copyLockFileCommand
	renderCtx.HasWebSocket = detectFrontendUsesWebSocket(localPath)
	renderCtx.APIProxyStripPrefix = detectFrontendAPIProxyStripPrefix(localPath)
	renderCtx.ObjectStorageProxyPrefixes = objectStorageProxyPrefixesForGroupDB(tx, groupID)

	return s.createTemplateDeployConfigs(tx, projectId, localPath, result, renderCtx, "react", []string{"local-full"})
}

func (s *DeployToolService) createVueDeployConfig(tx *gorm.DB, projectId uint, localPath string, groupID uint, ctx templateContext, frontendDeployPort int, result *QuickInitResult) error {
	renderCtx := buildDeployTemplateContext(ctx, frontendDeployPort, "vue")
	packageManager, installCommand, buildCommand, copyLockFileCommand := detectFrontendPackageCommands(localPath)
	renderCtx.PackageManager = packageManager
	renderCtx.InstallCommand = installCommand
	renderCtx.BuildCommand = buildCommand
	renderCtx.CopyLockFileCommand = copyLockFileCommand
	renderCtx.HasWebSocket = detectFrontendUsesWebSocket(localPath)
	renderCtx.APIProxyStripPrefix = detectFrontendAPIProxyStripPrefix(localPath)
	renderCtx.ObjectStorageProxyPrefixes = objectStorageProxyPrefixesForGroupDB(tx, groupID)

	return s.createTemplateDeployConfigs(tx, projectId, localPath, result, renderCtx, "vue", []string{"local-full"})
}

func objectStorageProxyPrefixesForGroupDB(db *gorm.DB, groupID uint) []objectStorageProxyPrefix {
	if groupID == 0 || db == nil {
		return nil
	}

	var projects []system.TbProject
	if err := db.
		Where("group_id = ? AND computer_language NOT IN ?", groupID, []string{"react", "vue"}).
		Order("id desc").
		Find(&projects).Error; err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var prefixes []objectStorageProxyPrefix
	for _, project := range projects {
		for _, prefix := range detectObjectStorageProxyPrefixes(project.LocalProjectPath) {
			key := prefix.Prefix + "\x00" + prefix.Endpoint
			if seen[key] {
				continue
			}
			seen[key] = true
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

func detectObjectStorageProxyPrefixes(projectPath string) []objectStorageProxyPrefix {
	if strings.TrimSpace(projectPath) == "" {
		return nil
	}

	var fallback *objectStorageProxyPrefix
	for _, name := range []string{"config.docker.yaml", "config.yaml"} {
		configPath := filepath.Join(projectPath, name)
		if !fileExists(configPath) {
			continue
		}

		v := viper.New()
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			continue
		}

		bucketName := strings.Trim(strings.TrimSpace(v.GetString("minio.bucket-name")), "/")
		endpoint, frontendReachable := normalizeObjectStorageProxyEndpointForFrontend(v.GetString("minio.endpoint"), v.GetBool("minio.use-ssl"))
		if bucketName == "" || endpoint == "" {
			continue
		}
		prefix := objectStorageProxyPrefix{Prefix: bucketName, Endpoint: endpoint}
		if frontendReachable {
			return []objectStorageProxyPrefix{prefix}
		}
		if fallback == nil {
			fallback = &prefix
		}
	}
	if fallback != nil {
		return []objectStorageProxyPrefix{*fallback}
	}
	return nil
}

func normalizeObjectStorageProxyEndpointForFrontend(endpoint string, useSSL bool) (string, bool) {
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		return "", false
	}

	withScheme := normalized
	if !strings.HasPrefix(withScheme, "http://") && !strings.HasPrefix(withScheme, "https://") {
		if useSSL {
			withScheme = "https://" + withScheme
		} else {
			withScheme = "http://" + withScheme
		}
	}

	parsed, err := url.Parse(withScheme)
	if err != nil || parsed.Host == "" {
		return normalizeObjectStorageProxyEndpoint(endpoint, useSSL), false
	}

	host := parsed.Hostname()
	switch {
	case isLoopbackObjectStorageHost(host):
		port := parsed.Port()
		parsed.Host = "host.docker.internal"
		if port != "" {
			parsed.Host += ":" + port
		}
		return parsed.String(), true
	case strings.EqualFold(host, "host.docker.internal"):
		return parsed.String(), true
	case isContainerOnlyObjectStorageHost(host):
		return parsed.String(), false
	default:
		return parsed.String(), true
	}
}

func isLoopbackObjectStorageHost(host string) bool {
	normalized := strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if normalized == "localhost" {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}

func isContainerOnlyObjectStorageHost(host string) bool {
	normalized := strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if normalized == "" {
		return false
	}
	if net.ParseIP(normalized) != nil {
		return false
	}
	return !strings.Contains(normalized, ".")
}

func normalizeObjectStorageProxyEndpoint(endpoint string, useSSL bool) string {
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		return normalized
	}
	if useSSL {
		return "https://" + normalized
	}
	return "http://" + normalized
}

func detectFrontendPackageCommands(localPath string) (packageManager string, installCommand string, buildCommand string, copyLockFileCommand string) {
	switch {
	case localPath != "" && fileExists(filepath.Join(localPath, "pnpm-lock.yaml")):
		return "pnpm", "corepack enable && pnpm install --frozen-lockfile", "pnpm run build", "COPY pnpm-lock.yaml ./"
	case localPath != "" && fileExists(filepath.Join(localPath, "yarn.lock")):
		return "yarn", "corepack enable && yarn install --frozen-lockfile", "yarn build", "COPY yarn.lock ./"
	case localPath != "" && fileExists(filepath.Join(localPath, "package-lock.json")):
		return "npm", "npm ci", "npm run build", "COPY package-lock.json ./"
	default:
		return "npm", "npm install", "npm run build", "# no lock file found"
	}
}

func normalizeFrontendDeployPort(frontendDeployPort int, backendDeployPort int) int {
	if frontendDeployPort > 0 && frontendDeployPort != backendDeployPort && frontendDeployPort != 80 {
		return frontendDeployPort
	}
	if backendDeployPort == 6001 {
		return 6002
	}
	return 6001
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
