package system

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
)

// inferGroupBackendHTTPPort is part of the normal deployment path: frontend
// nginx generation uses it to discover the backend port in the same group.
func inferGroupBackendHTTPPort(groupID uint) (int, bool) {
	if groupID == 0 || global.GVA_DB == nil {
		return 0, false
	}

	var projects []modelSystem.TbProject
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

// detectFrontendAPIProxyStripPrefix preserves the proxy behavior of an
// existing Vite project when normal deployment scripts are materialized.
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

// detectSnailJobPythonProject preserves environment injection used by the
// normal Python Docker deployment path.
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
