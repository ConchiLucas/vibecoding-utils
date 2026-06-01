package system

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

type deployTemplateContext struct {
	ProjectName                    string
	ContainerName                  string
	ImageName                      string
	BaseImageName                  string
	AppPort                        int
	FrontendDeployPort             int
	BackendDeployPort              int
	WebSocketDeployPort            int
	WebSocketPort                  int
	HasWebSocket                   bool
	APIProxyStripPrefix            bool
	DatabaseName                   string
	DatabaseUsername               string
	DatabasePassword               string
	RedisHost                      string
	RedisPort                      int
	RedisPassword                  string
	PackageManager                 string
	InstallCommand                 string
	BuildCommand                   string
	DistDir                        string
	NodeVersion                    string
	GoVersion                      string
	GoConfigCopyCommand            string
	JavaVersion                    string
	PythonVersion                  string
	CopyLockFileCommand            string
	PythonDependencyCopyCommand    string
	PythonDependencyInstallCommand string
	PythonStartCommand             string
	ObjectStorageProxyPrefixes     []objectStorageProxyPrefix
}

type objectStorageProxyPrefix struct {
	Prefix   string
	Endpoint string
}

type deployTemplateRoute struct {
	RouteKey             string
	RouteName            string
	LocalExecuteCommand  string
	LocalStopCommand     string
	ServerExecuteCommand string
	BuildType            string
	FileName             string
	ScriptType           uint
}

type deployTemplateScript struct {
	FileName string
	Content  string
}

type renderedDeployTemplate struct {
	Route   deployTemplateRoute
	Scripts []deployTemplateScript
}

func (r renderedDeployTemplate) scriptContent(fileName string) string {
	for _, script := range r.Scripts {
		if script.FileName == fileName {
			return script.Content
		}
	}
	return ""
}

func loadDeployTemplate(language, deployType string, ctx deployTemplateContext) (renderedDeployTemplate, error) {
	root, err := deployTemplateRoot()
	if err != nil {
		return renderedDeployTemplate{}, err
	}

	templateDir := filepath.Join(root, strings.ToLower(language), deployType)
	route, err := loadDeployTemplateRoute(filepath.Join(templateDir, "route.yaml"), ctx)
	if err != nil {
		return renderedDeployTemplate{}, err
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return renderedDeployTemplate{}, fmt.Errorf("读取部署模板目录失败: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	rendered := renderedDeployTemplate{Route: route}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tpl") {
			continue
		}

		content, err := renderTemplateFile(filepath.Join(templateDir, entry.Name()), ctx)
		if err != nil {
			return renderedDeployTemplate{}, err
		}
		rendered.Scripts = append(rendered.Scripts, deployTemplateScript{
			FileName: strings.TrimSuffix(entry.Name(), ".tpl"),
			Content:  content,
		})
	}
	return rendered, nil
}

func deployTemplateRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取工作目录失败: %w", err)
	}

	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "resource", "deploy-templates")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		serverCandidate := filepath.Join(dir, "server", "resource", "deploy-templates")
		if info, err := os.Stat(serverCandidate); err == nil && info.IsDir() {
			return serverCandidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", fmt.Errorf("未找到部署模板目录 resource/deploy-templates")
}

func loadDeployTemplateRoute(path string, ctx deployTemplateContext) (deployTemplateRoute, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return deployTemplateRoute{}, fmt.Errorf("读取路由模板失败: %w", err)
	}
	tpl, err := template.New(filepath.Base(path)).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return deployTemplateRoute{}, fmt.Errorf("解析路由模板失败: %w", err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, ctx); err != nil {
		return deployTemplateRoute{}, fmt.Errorf("渲染路由模板失败: %w", err)
	}

	route := deployTemplateRoute{}
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return deployTemplateRoute{}, fmt.Errorf("路由模板格式错误: %s", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		switch key {
		case "routeKey":
			route.RouteKey = value
		case "routeName":
			route.RouteName = value
		case "localExecuteCommand":
			route.LocalExecuteCommand = value
		case "localStopCommand":
			route.LocalStopCommand = value
		case "serverExecuteCommand":
			route.ServerExecuteCommand = value
		case "buildType":
			route.BuildType = value
		case "fileName":
			route.FileName = value
		case "scriptType":
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return deployTemplateRoute{}, fmt.Errorf("scriptType 格式错误: %w", err)
			}
			route.ScriptType = uint(parsed)
		}
	}

	if route.RouteKey == "" || route.RouteName == "" || route.LocalExecuteCommand == "" {
		return deployTemplateRoute{}, fmt.Errorf("路由模板缺少必要字段")
	}
	return route, nil
}

func renderTemplateFile(path string, ctx deployTemplateContext) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败(%s): %w", filepath.Base(path), err)
	}

	tpl, err := template.New(filepath.Base(path)).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("解析模板文件失败(%s): %w", filepath.Base(path), err)
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, ctx); err != nil {
		return "", fmt.Errorf("渲染模板文件失败(%s): %w", filepath.Base(path), err)
	}
	return out.String(), nil
}
