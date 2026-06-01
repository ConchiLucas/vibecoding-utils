package system

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	deployProjectTypeVue           = "Vue"
	deployProjectTypeReact         = "React"
	deployProjectTypePython        = "Python"
	deployProjectTypeJava          = "Java"
	deployProjectTypeGo            = "Go"
	deployProjectTypeDockerCompose = "前后端 docker-compose"
	deployProjectTypeUnknown       = "未知"
)

const deployProjectTypeScanMaxDepth = 3

type DeployProjectTypeCandidate struct {
	Type  string  `json:"type"`
	Score float64 `json:"score"`
}

type DeployProjectTypeResult struct {
	ProjectType     string                       `json:"projectType"`
	Confidence      float64                      `json:"confidence"`
	PrimaryLanguage string                       `json:"primaryLanguage"`
	ProjectName     string                       `json:"projectName"`
	Candidates      []DeployProjectTypeCandidate `json:"candidates"`
	Evidence        []string                     `json:"evidence"`
	Warnings        []string                     `json:"warnings"`
}

type deployProjectTypeDetection struct {
	root          string
	result        *DeployProjectTypeResult
	scores        map[string]float64
	composeSeen   bool
	frontendTypes map[string]bool
	backendTypes  map[string]bool
	rootNameFound bool
}

// DetectDeployProjectType identifies which supported deploy-project category best matches localPath.
func DetectDeployProjectType(localPath string) (*DeployProjectTypeResult, error) {
	localPath = filepath.Clean(strings.TrimSpace(localPath))
	if localPath == "." || localPath == "" {
		return nil, fmt.Errorf("路径不能为空")
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("路径不存在: %s", localPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", localPath)
	}

	detection := &deployProjectTypeDetection{
		root: localPath,
		result: &DeployProjectTypeResult{
			ProjectType: deployProjectTypeUnknown,
			ProjectName: filepath.Base(localPath),
		},
		scores:        make(map[string]float64),
		frontendTypes: make(map[string]bool),
		backendTypes:  make(map[string]bool),
	}

	err = filepath.WalkDir(localPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			detection.warn("跳过无法访问的路径 %s: %v", detection.rel(path), walkErr)
			return nil
		}

		depth := detection.depth(path)
		if entry.IsDir() {
			if path != localPath && shouldSkipDeployProjectTypeDir(entry.Name()) {
				return filepath.SkipDir
			}
			if depth > deployProjectTypeScanMaxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		if depth > deployProjectTypeScanMaxDepth {
			return nil
		}
		detection.inspectFile(path, entry.Name())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("扫描目录失败: %w", err)
	}

	detection.finalize()
	return detection.result, nil
}

func (d *deployProjectTypeDetection) inspectFile(path string, name string) {
	switch name {
	case "go.mod":
		d.addCandidate(deployProjectTypeGo, 0.80, "%s 存在 go.mod", d.rel(path))
		d.backendTypes[deployProjectTypeGo] = true
		d.setProjectNameFromGoMod(path)
	case "pom.xml":
		d.addCandidate(deployProjectTypeJava, 0.82, "%s 存在 pom.xml", d.rel(path))
		d.backendTypes[deployProjectTypeJava] = true
		d.setProjectNameFromPom(path)
	case "build.gradle", "build.gradle.kts":
		d.addCandidate(deployProjectTypeJava, 0.78, "%s 存在 %s", d.rel(filepath.Dir(path)), name)
		d.backendTypes[deployProjectTypeJava] = true
	case "requirements.txt":
		d.addCandidate(deployProjectTypePython, 0.74, "%s 存在 requirements.txt", d.rel(path))
		d.backendTypes[deployProjectTypePython] = true
	case "pyproject.toml":
		d.addCandidate(deployProjectTypePython, 0.82, "%s 存在 pyproject.toml", d.rel(path))
		d.backendTypes[deployProjectTypePython] = true
		d.setProjectNameFromPyproject(path)
	case "package.json":
		d.inspectPackageJSON(path)
	case "main.go":
		d.addCandidate(deployProjectTypeGo, 0.10, "%s 存在 Go 入口文件 main.go", d.rel(path))
		d.backendTypes[deployProjectTypeGo] = true
	case "main.py", "app.py", "manage.py":
		d.addCandidate(deployProjectTypePython, 0.12, "%s 存在 Python 入口文件 %s", d.rel(path), name)
		d.backendTypes[deployProjectTypePython] = true
	default:
		if isDeployProjectTypeComposeFile(name) {
			d.composeSeen = true
			if d.depth(path) == 1 {
				d.rootNameFound = true
			}
			d.addCandidate(deployProjectTypeDockerCompose, 0.55, "%s 存在 %s", d.rel(filepath.Dir(path)), name)
			d.inspectComposeFile(path)
		}
		d.inspectFrontendEntry(path)
	}
}

func (d *deployProjectTypeDetection) inspectPackageJSON(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		d.warn("读取 %s 失败: %v", d.rel(path), err)
		return
	}

	var pkg struct {
		Name            string                 `json:"name"`
		Dependencies    map[string]interface{} `json:"dependencies"`
		DevDependencies map[string]interface{} `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		d.warn("解析 %s 失败: %v", d.rel(path), err)
		return
	}

	if d.shouldPreferProjectName(path) && strings.TrimSpace(pkg.Name) != "" {
		d.result.ProjectName = strings.TrimSpace(pkg.Name)
		d.rootNameFound = d.depth(path) == 1
	}

	if hasPackageDependency(pkg.Dependencies, pkg.DevDependencies, "vue") {
		d.addCandidate(deployProjectTypeVue, 0.85, "%s 依赖包含 vue", d.rel(path))
		d.frontendTypes[deployProjectTypeVue] = true
	}
	if hasPackageDependency(pkg.Dependencies, pkg.DevDependencies, "react") {
		d.addCandidate(deployProjectTypeReact, 0.85, "%s 依赖包含 react", d.rel(path))
		d.frontendTypes[deployProjectTypeReact] = true
	}
}

func (d *deployProjectTypeDetection) inspectFrontendEntry(path string) {
	normalized := filepath.ToSlash(d.rel(path))
	if strings.HasSuffix(normalized, "src/main.ts") || strings.HasSuffix(normalized, "src/main.js") {
		d.addCandidate(deployProjectTypeVue, 0.08, "%s 存在常见 Vue 入口文件", normalized)
	}
	if strings.HasSuffix(normalized, "src/main.tsx") || strings.HasSuffix(normalized, "src/index.tsx") {
		d.addCandidate(deployProjectTypeReact, 0.08, "%s 存在常见 React 入口文件", normalized)
	}
}

func (d *deployProjectTypeDetection) inspectComposeFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		d.warn("读取 %s 失败: %v", d.rel(path), err)
		return
	}
	content := strings.ToLower(string(data))
	if strings.Contains(content, "services:") {
		d.addCandidate(deployProjectTypeDockerCompose, 0.10, "%s 包含 docker compose services 配置", d.rel(path))
	}
	if strings.Contains(content, "postgres") || strings.Contains(content, "mysql") || strings.Contains(content, "redis") {
		d.addCandidate(deployProjectTypeDockerCompose, 0.05, "%s 包含常见基础设施服务", d.rel(path))
	}
}

func (d *deployProjectTypeDetection) setProjectNameFromGoMod(path string) {
	if !d.shouldPreferProjectName(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		d.warn("读取 %s 失败: %v", d.rel(path), err)
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			parts := strings.Split(moduleName, "/")
			if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) != "" {
				d.result.ProjectName = strings.TrimSpace(parts[len(parts)-1])
				d.rootNameFound = d.depth(path) == 1
			}
			return
		}
	}
}

func (d *deployProjectTypeDetection) setProjectNameFromPom(path string) {
	if !d.shouldPreferProjectName(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		d.warn("读取 %s 失败: %v", d.rel(path), err)
		return
	}
	if projectName := extractRootMavenArtifactID(string(data)); projectName != "" {
		d.result.ProjectName = projectName
		d.rootNameFound = d.depth(path) == 1
	}
}

func (d *deployProjectTypeDetection) setProjectNameFromPyproject(path string) {
	if !d.shouldPreferProjectName(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		d.warn("读取 %s 失败: %v", d.rel(path), err)
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			projectName := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if projectName != "" {
				d.result.ProjectName = projectName
				d.rootNameFound = d.depth(path) == 1
			}
			return
		}
	}
}

func (d *deployProjectTypeDetection) finalize() {
	hasFrontend := len(d.frontendTypes) > 0
	hasBackend := len(d.backendTypes) > 0
	if d.composeSeen && hasFrontend && hasBackend {
		d.addCandidate(deployProjectTypeDockerCompose, 0.40, "同时发现前端项目、后端项目和 docker-compose 编排")
	}

	for projectType, score := range d.scores {
		d.result.Candidates = append(d.result.Candidates, DeployProjectTypeCandidate{
			Type:  projectType,
			Score: roundDeployProjectTypeScore(score),
		})
	}
	sort.Slice(d.result.Candidates, func(i, j int) bool {
		if d.result.Candidates[i].Score == d.result.Candidates[j].Score {
			return d.result.Candidates[i].Type < d.result.Candidates[j].Type
		}
		return d.result.Candidates[i].Score > d.result.Candidates[j].Score
	})

	if len(d.result.Candidates) == 0 {
		d.result.ProjectType = deployProjectTypeUnknown
		d.result.Confidence = 0
		d.result.PrimaryLanguage = ""
		return
	}

	best := d.result.Candidates[0]
	if d.composeSeen && hasFrontend && hasBackend {
		for _, candidate := range d.result.Candidates {
			if candidate.Type == deployProjectTypeDockerCompose {
				best = candidate
				break
			}
		}
	}
	d.result.ProjectType = best.Type
	d.result.Confidence = best.Score
	d.result.PrimaryLanguage = d.primaryLanguage()
	if d.result.ProjectType == deployProjectTypeDockerCompose && d.result.PrimaryLanguage == "" {
		d.result.PrimaryLanguage = deployProjectTypeDockerCompose
	}
}

func (d *deployProjectTypeDetection) addCandidate(projectType string, score float64, format string, args ...interface{}) {
	next := d.scores[projectType] + score
	if next > 0.95 {
		next = 0.95
	}
	d.scores[projectType] = next

	evidence := fmt.Sprintf(format, args...)
	if evidence != "" && !containsString(d.result.Evidence, evidence) {
		d.result.Evidence = append(d.result.Evidence, evidence)
	}
}

func (d *deployProjectTypeDetection) primaryLanguage() string {
	ordered := []string{
		deployProjectTypeGo,
		deployProjectTypeJava,
		deployProjectTypePython,
		deployProjectTypeReact,
		deployProjectTypeVue,
	}
	var parts []string
	for _, projectType := range ordered {
		if d.backendTypes[projectType] || d.frontendTypes[projectType] {
			parts = append(parts, projectType)
		}
	}
	if len(parts) == 0 && d.result.ProjectType != deployProjectTypeUnknown {
		return d.result.ProjectType
	}
	return strings.Join(parts, " + ")
}

func (d *deployProjectTypeDetection) shouldPreferProjectName(path string) bool {
	return !d.rootNameFound || d.depth(path) == 1
}

func (d *deployProjectTypeDetection) depth(path string) int {
	rel := d.rel(path)
	if rel == "." {
		return 0
	}
	return len(strings.Split(filepath.ToSlash(rel), "/"))
}

func (d *deployProjectTypeDetection) rel(path string) string {
	rel, err := filepath.Rel(d.root, path)
	if err != nil {
		return path
	}
	if rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func (d *deployProjectTypeDetection) warn(format string, args ...interface{}) {
	d.result.Warnings = append(d.result.Warnings, fmt.Sprintf(format, args...))
}

func shouldSkipDeployProjectTypeDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", "target", "vendor", ".venv", "venv", ".idea":
		return true
	default:
		return false
	}
}

func isDeployProjectTypeComposeFile(name string) bool {
	switch name {
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		return true
	default:
		return false
	}
}

func hasPackageDependency(dependencies map[string]interface{}, devDependencies map[string]interface{}, name string) bool {
	if _, ok := dependencies[name]; ok {
		return true
	}
	if _, ok := devDependencies[name]; ok {
		return true
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func roundDeployProjectTypeScore(score float64) float64 {
	return float64(int(score*100+0.5)) / 100
}
