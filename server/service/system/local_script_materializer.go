package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"gorm.io/gorm"
)

type localScriptMaterializationRequest struct {
	Project    modelSystem.TbProject
	RouteID    uint
	ScriptPath string
}

type preparedLocalScript struct {
	Script          modelSystem.TbProjectScript
	DatabaseScripts []modelSystem.TbProjectScript
	Content         string
	FilePath        string
}

type localScriptPublishState struct {
	TargetPath string
	TempPath   string
	BackupPath string
	Existed    bool
	Published  bool
}

func loadLocalScriptsForMaterialization(db *gorm.DB, requests []localScriptMaterializationRequest) ([]preparedLocalScript, error) {
	prepared := make([]preparedLocalScript, 0)
	targetIndexes := make(map[string]int)
	for _, request := range requests {
		var scripts []modelSystem.TbProjectScript
		query := db.Where("project_id = ? AND script_type <> ?", request.Project.ID, 2)
		if request.RouteID != 0 {
			query = query.Where("route_id = ?", request.RouteID)
		}
		if err := query.Order("id asc").Find(&scripts).Error; err != nil {
			return nil, fmt.Errorf("读取本地部署脚本失败(project=%d route=%d): %w", request.Project.ID, request.RouteID, err)
		}

		localCount := 0
		for _, script := range scripts {
			if script.Content == "" {
				continue
			}
			localCount++
			filePath, err := safeLocalScriptFilePath(request.ScriptPath, script.FileName)
			if err != nil {
				return nil, fmt.Errorf("脚本路径无效(project=%d route=%d script=%d file=%s): %w", request.Project.ID, request.RouteID, script.ID, script.FileName, err)
			}
			content, err := normalizeLocalScriptForDeploy(request.Project, script)
			if err != nil {
				return nil, fmt.Errorf("规范化本地部署脚本失败(project=%d name=%s route=%d script=%d file=%s): %w", request.Project.ID, request.Project.ProjectName, request.RouteID, script.ID, script.FileName, err)
			}
			if existingIndex, exists := targetIndexes[filePath]; exists {
				if prepared[existingIndex].Content != content {
					return nil, fmt.Errorf("部署脚本目标文件冲突(path=%s first_script=%d second_script=%d)", filePath, prepared[existingIndex].Script.ID, script.ID)
				}
				prepared[existingIndex].DatabaseScripts = append(prepared[existingIndex].DatabaseScripts, script)
				continue
			}
			targetIndexes[filePath] = len(prepared)
			prepared = append(prepared, preparedLocalScript{
				Script:          script,
				DatabaseScripts: []modelSystem.TbProjectScript{script},
				Content:         content,
				FilePath:        filePath,
			})
		}
		if localCount == 0 {
			return nil, fmt.Errorf("本地部署路线没有可落盘脚本(project=%d route=%d path=%s)", request.Project.ID, request.RouteID, request.ScriptPath)
		}
	}
	return prepared, nil
}

func normalizeLocalScriptForDeploy(project modelSystem.TbProject, script modelSystem.TbProjectScript) (string, error) {
	content := script.Content
	if isFrontendDeployProject(project) && script.FileName == "nginx.conf" {
		backendPort := inferBackendPortForFrontendDeploy(project, content)
		content = normalizeFrontendNginxScriptForDeploy(content, backendPort, project.LocalProjectPath)
	}
	if isPythonDeployProject(project) && isPythonDependencyDockerfile(script.FileName) {
		content = normalizePythonDependencyDockerfileForDeploy(content)
	}
	if isPythonDeployProject(project) && script.FileName == "docker-compose.yml" {
		appPort := inferPythonAppPortForDeploy(project, content)
		content = normalizePythonComposeForDeploy(content, appPort, project.LocalProjectPath)
	}
	if isComposeFileName(script.FileName) {
		normalized, _, err := normalizeComposeSharedNetwork(content)
		if err != nil {
			return "", err
		}
		content = normalized
	}
	return content, nil
}

func publishPreparedLocalScripts(db *gorm.DB, prepared []preparedLocalScript, beforePublish func(int) error) error {
	states := make([]localScriptPublishState, len(prepared))
	createdDirectories := make([]string, 0)
	createdDirectorySet := make(map[string]bool)
	stageFailed := true
	defer func() {
		if stageFailed {
			cleanupPublishedLocalScripts(states)
			removeCreatedScriptDirectories(createdDirectories)
		}
	}()

	for index, item := range prepared {
		created, err := createScriptDirectory(filepath.Dir(item.FilePath))
		if err != nil {
			return fmt.Errorf("创建脚本目录失败(script=%d file=%s): %w", item.Script.ID, item.Script.FileName, err)
		}
		for _, directory := range created {
			if !createdDirectorySet[directory] {
				createdDirectorySet[directory] = true
				createdDirectories = append(createdDirectories, directory)
			}
		}
		temporary, err := os.CreateTemp(filepath.Dir(item.FilePath), ".vibedeploy-script-*")
		if err != nil {
			return fmt.Errorf("创建脚本临时文件失败(script=%d file=%s): %w", item.Script.ID, item.Script.FileName, err)
		}
		states[index] = localScriptPublishState{TargetPath: item.FilePath, TempPath: temporary.Name()}
		if _, err := temporary.WriteString(item.Content); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("写入脚本临时文件失败(script=%d file=%s): %w", item.Script.ID, item.Script.FileName, err)
		}
		if err := temporary.Chmod(0o644); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("设置脚本文件权限失败(script=%d file=%s): %w", item.Script.ID, item.Script.FileName, err)
		}
		if err := temporary.Close(); err != nil {
			return fmt.Errorf("关闭脚本临时文件失败(script=%d file=%s): %w", item.Script.ID, item.Script.FileName, err)
		}
	}
	stageFailed = false

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, item := range prepared {
			for _, script := range item.DatabaseScripts {
				if item.Content == script.Content {
					continue
				}
				if err := updateStoredComposeScriptContent(tx, script, item.Content); err != nil {
					return err
				}
			}
		}
		for index := range states {
			if beforePublish != nil {
				if err := beforePublish(index); err != nil {
					return err
				}
			}
			if err := publishLocalScriptState(&states[index]); err != nil {
				return fmt.Errorf("发布脚本文件失败(script=%d file=%s): %w", prepared[index].Script.ID, prepared[index].Script.FileName, err)
			}
		}
		return nil
	})
	if err != nil {
		rollbackErr := rollbackPublishedLocalScripts(states)
		removeCreatedScriptDirectories(createdDirectories)
		if rollbackErr != nil {
			return fmt.Errorf("%w; 文件回滚失败: %v", err, rollbackErr)
		}
		return err
	}

	cleanupPublishedLocalScripts(states)
	for _, item := range prepared {
		if global.GVA_LOG != nil {
			global.GVA_LOG.Info(fmt.Sprintf("文件 %s 已从数据库加载到 %s", item.Script.FileName, item.FilePath))
		}
	}
	return nil
}

func publishLocalScriptState(state *localScriptPublishState) error {
	info, err := os.Lstat(state.TargetPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("目标路径不是普通文件: %s", state.TargetPath)
		}
		backup, err := os.CreateTemp(filepath.Dir(state.TargetPath), ".vibedeploy-backup-*")
		if err != nil {
			return err
		}
		state.BackupPath = backup.Name()
		if err := backup.Close(); err != nil {
			return err
		}
		if err := os.Remove(state.BackupPath); err != nil {
			return err
		}
		if err := os.Rename(state.TargetPath, state.BackupPath); err != nil {
			return err
		}
		state.Existed = true
	}
	if err := os.Rename(state.TempPath, state.TargetPath); err != nil {
		return err
	}
	state.TempPath = ""
	state.Published = true
	return nil
}

func rollbackPublishedLocalScripts(states []localScriptPublishState) error {
	var failures []string
	for index := len(states) - 1; index >= 0; index-- {
		state := &states[index]
		if state.Published {
			if err := os.Remove(state.TargetPath); err != nil && !os.IsNotExist(err) {
				failures = append(failures, err.Error())
			}
		}
		if state.Existed && state.BackupPath != "" {
			if err := os.Rename(state.BackupPath, state.TargetPath); err != nil {
				failures = append(failures, err.Error())
			} else {
				state.BackupPath = ""
			}
		}
		if state.TempPath != "" {
			if err := os.Remove(state.TempPath); err != nil && !os.IsNotExist(err) {
				failures = append(failures, err.Error())
			}
			state.TempPath = ""
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func cleanupPublishedLocalScripts(states []localScriptPublishState) {
	for _, state := range states {
		if state.TempPath != "" {
			_ = os.Remove(state.TempPath)
		}
		if state.BackupPath != "" {
			_ = os.Remove(state.BackupPath)
		}
	}
}

func createScriptDirectory(directory string) ([]string, error) {
	directory = filepath.Clean(directory)
	missing := make([]string, 0)
	for current := directory; ; current = filepath.Dir(current) {
		if _, err := os.Stat(current); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	return missing, nil
}

func removeCreatedScriptDirectories(directories []string) {
	for _, directory := range directories {
		_ = os.Remove(directory)
	}
}
