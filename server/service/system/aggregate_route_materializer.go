package system

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var aggregateChildStartPattern = regexp.MustCompile(`^\s*(?:sh|bash)\s+(?:"\$ROOT_DIR/([^"]+/start\.sh)"|'\$ROOT_DIR/([^']+/start\.sh)'|\$ROOT_DIR/([^\s;]+/start\.sh))(?:\s+.*)?\s*$`)

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
		reference = filepath.Clean(filepath.FromSlash(reference))
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
