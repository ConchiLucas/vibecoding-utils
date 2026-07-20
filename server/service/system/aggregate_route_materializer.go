package system

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"gorm.io/gorm"
)

var aggregateChildStartPattern = regexp.MustCompile(`^\s*(?:sh|bash)\s+(?:"\$ROOT_DIR/([^"]+/start\.sh)"|'\$ROOT_DIR/([^']+/start\.sh)'|\$ROOT_DIR/([^\s;]+/start\.sh))(?:\s+.*)?\s*$`)
var aggregateAbsoluteStartPattern = regexp.MustCompile(`^\s*(?:sh|bash)\s+["']?(/[^"'\s;]+/start\.sh)["']?(?:\s+.*)?\s*$`)

type aggregateChildRoute struct {
	Project    modelSystem.TbProject
	Route      modelSystem.TbProjectRoute
	ScriptPath string
}

func parseAggregateChildScriptPaths(rootPath, aggregateScriptPath, content string) ([]string, error) {
	rootPath = filepath.Clean(strings.TrimSpace(rootPath))
	aggregateScriptPath = filepath.Clean(strings.TrimSpace(aggregateScriptPath))
	if rootPath == "" || rootPath == "." || !filepath.IsAbs(rootPath) {
		return nil, fmt.Errorf("聚合项目本地路径必须是绝对路径")
	}

	seen := make(map[string]bool)
	paths := make([]string, 0)
	for index, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		matches := aggregateChildStartPattern.FindStringSubmatch(line)
		if matches == nil {
			if aggregateAbsoluteStartPattern.MatchString(line) {
				return nil, fmt.Errorf("聚合脚本第 %d 行不能调用绝对路径 start.sh: %s", index+1, trimmed)
			}
			if strings.Contains(line, "$ROOT_DIR/") && strings.Contains(line, "start.sh") {
				return nil, fmt.Errorf("聚合脚本第 %d 行包含不支持的子路线调用: %s", index+1, trimmed)
			}
			continue
		}

		reference := ""
		for _, candidate := range matches[1:] {
			if candidate != "" {
				reference = candidate
				break
			}
		}
		rawReference := filepath.FromSlash(reference)
		for _, component := range strings.Split(rawReference, string(filepath.Separator)) {
			if component == ".." {
				return nil, fmt.Errorf("聚合脚本第 %d 行的子路线越过项目目录", index+1)
			}
		}
		reference = filepath.Clean(rawReference)
		if reference == "." || filepath.IsAbs(reference) || reference == ".." || strings.HasPrefix(reference, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("聚合脚本第 %d 行的子路线越过项目目录", index+1)
		}
		target := filepath.Clean(filepath.Join(rootPath, filepath.Dir(reference)))
		relative, err := filepath.Rel(rootPath, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("聚合脚本第 %d 行的子路线越过项目目录", index+1)
		}
		if target == aggregateScriptPath {
			return nil, fmt.Errorf("聚合脚本第 %d 行不能引用自身", index+1)
		}
		if !seen[target] {
			seen[target] = true
			paths = append(paths, target)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("聚合路线 start.sh 未引用任何子项目部署路线")
	}
	return paths, nil
}

func loadSingleLocalStartScript(db *gorm.DB, projectID, routeID uint) (modelSystem.TbProjectScript, error) {
	var scripts []modelSystem.TbProjectScript
	err := db.Where(
		"project_id = ? AND route_id = ? AND file_name = ? AND script_type <> ?",
		projectID,
		routeID,
		"start.sh",
		2,
	).Order("id asc").Find(&scripts).Error
	if err != nil {
		return modelSystem.TbProjectScript{}, fmt.Errorf("读取本地 start.sh 失败(project=%d route=%d): %w", projectID, routeID, err)
	}
	if len(scripts) != 1 {
		return modelSystem.TbProjectScript{}, fmt.Errorf("本地 start.sh 数量必须为 1(project=%d route=%d actual=%d)", projectID, routeID, len(scripts))
	}
	if strings.TrimSpace(scripts[0].Content) == "" {
		return modelSystem.TbProjectScript{}, fmt.Errorf("本地 start.sh 内容为空(project=%d route=%d script=%d)", projectID, routeID, scripts[0].ID)
	}
	return scripts[0], nil
}

func resolveAggregateChildRoutes(db *gorm.DB, aggregate modelSystem.TbProject, aggregateRoute modelSystem.TbProjectRoute, references []string) ([]aggregateChildRoute, error) {
	var projects []modelSystem.TbProject
	if err := db.Where("group_id = ?", aggregate.GroupId).Preload("Routes").Order("id asc").Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("读取聚合项目组子项目失败(group=%d): %w", aggregate.GroupId, err)
	}

	children := make([]aggregateChildRoute, 0, len(references))
	for _, reference := range references {
		reference = filepath.Clean(reference)
		matches := make([]aggregateChildRoute, 0, 1)
		for _, project := range projects {
			if project.ID == aggregate.ID || strings.TrimSpace(project.ComputerLanguage) == deployProjectTypeDockerCompose {
				continue
			}
			for _, route := range project.Routes {
				if route.ServerId != 0 {
					continue
				}
				scriptPath := filepath.Clean(resolveLocalScriptPath(route, project.LocalProjectPath))
				if scriptPath == reference {
					matches = append(matches, aggregateChildRoute{Project: project, Route: route, ScriptPath: scriptPath})
				}
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("聚合子路线未找到(project=%d route=%d path=%s)", aggregate.ID, aggregateRoute.ID, reference)
		}
		if len(matches) != 1 {
			candidateIDs := make([]string, 0, len(matches))
			for _, match := range matches {
				candidateIDs = append(candidateIDs, fmt.Sprintf("project=%d/route=%d", match.Project.ID, match.Route.ID))
			}
			return nil, fmt.Errorf("聚合子路线匹配到 %d 条(project=%d route=%d path=%s candidates=%s)", len(matches), aggregate.ID, aggregateRoute.ID, reference, strings.Join(candidateIDs, ","))
		}
		if _, err := loadSingleLocalStartScript(db, matches[0].Project.ID, matches[0].Route.ID); err != nil {
			return nil, fmt.Errorf("聚合子路线入口无效(path=%s): %w", reference, err)
		}
		children = append(children, matches[0])
	}
	return children, nil
}

func (s *DeployService) prepareAggregateChildDeployScripts(project modelSystem.TbProject, route modelSystem.TbProjectRoute, logCh chan string) error {
	if strings.TrimSpace(project.ComputerLanguage) != deployProjectTypeDockerCompose {
		return nil
	}
	sendAggregateDeployLog(logCh, "🧩 解析聚合部署依赖...")
	aggregateStart, err := loadSingleLocalStartScript(global.GVA_DB, project.ID, route.ID)
	if err != nil {
		return fmt.Errorf("读取聚合路线入口失败(project=%d route=%d): %w", project.ID, route.ID, err)
	}
	aggregatePath := resolveLocalScriptPath(route, project.LocalProjectPath)
	references, err := parseAggregateChildScriptPaths(project.LocalProjectPath, aggregatePath, aggregateStart.Content)
	if err != nil {
		return err
	}
	children, err := resolveAggregateChildRoutes(global.GVA_DB, project, route, references)
	if err != nil {
		return err
	}
	sendAggregateDeployLog(logCh, fmt.Sprintf("✅ 发现 %d 条子项目部署路线", len(children)))

	requests := make([]localScriptMaterializationRequest, 0, len(children)+1)
	requests = append(requests, localScriptMaterializationRequest{Project: project, RouteID: route.ID, ScriptPath: aggregatePath})
	for index, child := range children {
		sendAggregateDeployLog(logCh, fmt.Sprintf("📦 [%d/%d] 准备 %s / %s", index+1, len(children), child.Project.ProjectName, child.Route.RouteName))
		requests = append(requests, localScriptMaterializationRequest{Project: child.Project, RouteID: child.Route.ID, ScriptPath: child.ScriptPath})
	}
	prepared, err := loadLocalScriptsForMaterialization(global.GVA_DB, requests)
	if err != nil {
		return err
	}
	if err := publishPreparedLocalScripts(global.GVA_DB, prepared, nil); err != nil {
		return err
	}
	sendAggregateDeployLog(logCh, "✅ 聚合部署依赖脚本已全部落盘")
	return nil
}

func sendAggregateDeployLog(logCh chan string, message string) {
	if logCh == nil {
		return
	}
	select {
	case logCh <- message:
	default:
	}
}
