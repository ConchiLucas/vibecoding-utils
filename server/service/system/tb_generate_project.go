package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gorm.io/gorm"
)

type TbGenerateProjectService struct{}

type GenerateProjectCodeFile struct {
	Path         string `json:"path"`
	RelativePath string `json:"relativePath"`
	PathId       uint   `json:"pathId"`
	Status       string `json:"status"`
	Bytes        int    `json:"bytes"`
}

type GenerateProjectCodeResult struct {
	TemplateProjectId int                       `json:"templateProjectId"`
	ProjectInstanceId int                       `json:"projectInstanceId"`
	ProjectName       string                    `json:"projectName"`
	DiskPath          string                    `json:"diskPath"`
	PathSet           int                       `json:"pathSet"`
	GeneratedCount    int                       `json:"generatedCount"`
	SkippedCount      int                       `json:"skippedCount"`
	Files             []GenerateProjectCodeFile `json:"files"`
}

var codePlaceholderPattern = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}`)

func (s *TbGenerateProjectService) CreateTbGenerateProject(req *system.TbGenerateProject) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectService) DeleteTbGenerateProject(req system.TbGenerateProject) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	// 1. Delete project instances under this template card and their independent paths.
	var instances []system.TbGenerateProjectInstance
	if err := tx.Where("template_project_id = ?", req.ID).Find(&instances).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, instance := range instances {
		if err := deleteGenerateProjectInstancePaths(tx, int(instance.ID)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Unscoped().Delete(&instance).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 2. Find all legacy/template paths belonging to this project card.
	var paths []system.TbGenerateProjectPath
	if err := tx.Where("project_id = ? AND (project_instance_id = 0 OR project_instance_id IS NULL)", req.ID).Find(&paths).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 3. For each path, delete its models, then the path itself.
	for _, p := range paths {
		if err := tx.Where("path_id = ?", p.ID).Unscoped().Delete(&system.TbGenerateProjectPathModel{}).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Unscoped().Delete(&p).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 4. Delete database template examples belonging to this project.
	if err := tx.Where("project_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateDbTemplateScript{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("project_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateDbTemplateType{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 5. Delete the template card itself.
	if err := tx.Unscoped().Delete(&req).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *TbGenerateProjectService) UpdateTbGenerateProject(req *system.TbGenerateProject) error {
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateProjectService) GenerateCode(req systemReq.GenerateProjectCodeReq) (GenerateProjectCodeResult, error) {
	if req.TemplateProjectId <= 0 {
		return GenerateProjectCodeResult{}, errors.New("templateProjectId 必填")
	}
	module := strings.TrimSpace(req.Module)
	tableName := strings.TrimSpace(req.TableName)
	if module == "" {
		return GenerateProjectCodeResult{}, errors.New("module 必填")
	}
	if tableName == "" {
		return GenerateProjectCodeResult{}, errors.New("TableName 必填")
	}

	var project system.TbGenerateProject
	if err := global.GVA_DB.Where("id = ?", req.TemplateProjectId).First(&project).Error; err != nil {
		return GenerateProjectCodeResult{}, err
	}

	instance, err := s.resolveGenerateProjectInstance(project, req.ProjectInstanceId)
	if err != nil {
		return GenerateProjectCodeResult{}, err
	}

	diskPath, err := normalizeCodeGenerationRoot(firstNonEmptyString(instance.DiskPath, project.DiskPath))
	if err != nil {
		return GenerateProjectCodeResult{}, err
	}
	if err := os.MkdirAll(diskPath, 0o755); err != nil {
		return GenerateProjectCodeResult{}, fmt.Errorf("创建磁盘输出路径失败: %w", err)
	}

	pathSet, pathIds := resolveGenerateProjectPathFilter(req, instance.SelectedPathSetIdentity)
	paths, err := loadGenerateProjectPaths(int(instance.ID), pathSet, pathIds)
	if err != nil {
		return GenerateProjectCodeResult{}, err
	}
	if len(paths) == 0 {
		return GenerateProjectCodeResult{}, errors.New("当前相对路径配置没有可生成的启用文件")
	}

	contents, err := loadGenerateProjectPathContents(paths)
	if err != nil {
		return GenerateProjectCodeResult{}, err
	}

	vars := buildCodeGenerationVars(module, tableName)
	result := GenerateProjectCodeResult{
		TemplateProjectId: int(project.ID),
		ProjectInstanceId: int(instance.ID),
		ProjectName:       instance.ProjectName,
		DiskPath:          diskPath,
		PathSet:           pathSet,
		Files:             make([]GenerateProjectCodeFile, 0, len(paths)),
	}

	for _, pathObj := range paths {
		relativePath, targetPath, err := buildGeneratedFileTarget(diskPath, renderCodeGenerationText(pathObj.FileUrl, vars), renderCodeGenerationText(pathObj.FileName, vars))
		if err != nil {
			return GenerateProjectCodeResult{}, fmt.Errorf("路径 %d 无效: %w", pathObj.ID, err)
		}

		content := renderCodeGenerationText(contents[int(pathObj.ID)], vars)
		fileStatus := "generated"
		if _, err := os.Stat(targetPath); err == nil {
			if !req.Overwrite {
				result.SkippedCount++
				result.Files = append(result.Files, GenerateProjectCodeFile{
					Path:         targetPath,
					RelativePath: relativePath,
					PathId:       pathObj.ID,
					Status:       "skipped",
					Bytes:        len([]byte(content)),
				})
				continue
			}
			fileStatus = "overwritten"
		} else if err != nil && !os.IsNotExist(err) {
			return GenerateProjectCodeResult{}, fmt.Errorf("检查文件失败(%s): %w", targetPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return GenerateProjectCodeResult{}, fmt.Errorf("创建文件目录失败(%s): %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return GenerateProjectCodeResult{}, fmt.Errorf("写入文件失败(%s): %w", targetPath, err)
		}

		result.GeneratedCount++
		result.Files = append(result.Files, GenerateProjectCodeFile{
			Path:         targetPath,
			RelativePath: relativePath,
			PathId:       pathObj.ID,
			Status:       fileStatus,
			Bytes:        len([]byte(content)),
		})
	}

	return result, nil
}

func (s *TbGenerateProjectService) UpdateSelectedProjectInstance(templateProjectId int, projectInstanceId int) error {
	if templateProjectId <= 0 {
		return errors.New("templateProjectId 必填")
	}
	if projectInstanceId <= 0 {
		return errors.New("projectInstanceId 必填")
	}

	var count int64
	if err := global.GVA_DB.Model(&system.TbGenerateProjectInstance{}).
		Where("id = ? AND template_project_id = ?", projectInstanceId, templateProjectId).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("项目实例不存在")
	}

	return global.GVA_DB.Model(&system.TbGenerateProject{}).
		Where("id = ?", templateProjectId).
		Update("selected_project_instance_id", projectInstanceId).Error
}

func (s *TbGenerateProjectService) resolveGenerateProjectInstance(project system.TbGenerateProject, projectInstanceId int) (system.TbGenerateProjectInstance, error) {
	var instance system.TbGenerateProjectInstance

	if projectInstanceId > 0 {
		err := global.GVA_DB.Where("id = ? AND template_project_id = ?", projectInstanceId, project.ID).First(&instance).Error
		return instance, err
	}

	if project.SelectedProjectInstanceId > 0 {
		err := global.GVA_DB.Where("id = ? AND template_project_id = ?", project.SelectedProjectInstanceId, project.ID).First(&instance).Error
		if err == nil {
			return instance, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return instance, err
		}
	}

	err := global.GVA_DB.Where("template_project_id = ?", project.ID).Order("id ASC").First(&instance).Error
	if err == nil {
		return instance, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return instance, err
	}

	return (&TbGenerateProjectInstanceService{}).createDefaultFromTemplate(int(project.ID))
}

func (s *TbGenerateProjectService) GetTbGenerateProject(id string) (res system.TbGenerateProject, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectService) GetTbGenerateProjectList(projectConfigId int) (res []system.TbGenerateProject, err error) {
	db := global.GVA_DB.Model(&system.TbGenerateProject{})
	if projectConfigId > 0 {
		db = db.Where("project_config_id = ?", projectConfigId)
	}
	err = db.Order("id DESC").Find(&res).Error
	return
}

func (s *TbGenerateProjectService) CopyProject(id string) error {
	var project system.TbGenerateProject
	if err := global.GVA_DB.Where("id = ?", id).First(&project).Error; err != nil {
		return err
	}

	newProject := system.TbGenerateProject{
		ProjectConfigId: 0,
		BusinessType:    project.BusinessType,
		ProjectName:     project.ProjectName + "_copy",
		DiskPath:        project.DiskPath,
		Remark:          project.Remark,
		UserName:        project.UserName,
	}
	if err := global.GVA_DB.Create(&newProject).Error; err != nil {
		return err
	}

	var paths []system.TbGenerateProjectPath
	global.GVA_DB.Where("project_id = ?", id).Find(&paths)

	for _, p := range paths {
		oldPathId := p.ID
		newPath := system.TbGenerateProjectPath{
			ProjectId:         int(newProject.ID),
			ProjectInstanceId: 0,
			Enabled:           p.Enabled,
			FileUrl:           p.FileUrl,
			FileName:          p.FileName,
			Incremented:       p.Incremented,
		}
		global.GVA_DB.Create(&newPath)
		var oldModel system.TbGenerateProjectPathModel
		if err := global.GVA_DB.Where("path_id = ?", oldPathId).First(&oldModel).Error; err == nil {
			newModel := system.TbGenerateProjectPathModel{
				PathId:  int(newPath.ID),
				Content: oldModel.Content,
			}
			global.GVA_DB.Create(&newModel)
		}
	}

	var templateTypes []system.TbGenerateDbTemplateType
	global.GVA_DB.Where("project_id = ?", id).Find(&templateTypes)
	for _, templateType := range templateTypes {
		oldTypeId := templateType.ID
		newType := system.TbGenerateDbTemplateType{
			ProjectId: int(newProject.ID),
			TypeName:  templateType.TypeName,
			Sort:      templateType.Sort,
		}
		if err := global.GVA_DB.Create(&newType).Error; err != nil {
			return err
		}

		var scripts []system.TbGenerateDbTemplateScript
		global.GVA_DB.Where("type_id = ?", oldTypeId).Find(&scripts)
		for _, script := range scripts {
			newScript := system.TbGenerateDbTemplateScript{
				ProjectId:  int(newProject.ID),
				TypeId:     int(newType.ID),
				ScriptName: script.ScriptName,
				ScriptKind: script.ScriptKind,
				Content:    script.Content,
				Sort:       script.Sort,
			}
			if err := global.GVA_DB.Create(&newScript).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func resolveGenerateProjectPathFilter(req systemReq.GenerateProjectCodeReq, selectedPathSetIdentity string) (int, []int) {
	if len(req.PathIds) > 0 {
		return req.PathSet, req.PathIds
	}

	identity := strings.TrimSpace(firstNonEmptyString(req.PathSetIdentity, selectedPathSetIdentity))
	if strings.HasPrefix(identity, "path-set-0-copy-") {
		return 0, parseGeneratePathIds(strings.TrimPrefix(identity, "path-set-0-copy-"))
	}
	if strings.HasPrefix(identity, "path-set-") {
		if pathSet, err := strconv.Atoi(strings.TrimPrefix(identity, "path-set-")); err == nil {
			return pathSet, nil
		}
	}
	if identity == "path-set-primary" {
		return 0, nil
	}

	return req.PathSet, nil
}

func parseGeneratePathIds(value string) []int {
	var ids []int
	for _, part := range strings.Split(value, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func loadGenerateProjectPaths(projectInstanceId int, pathSet int, pathIds []int) ([]system.TbGenerateProjectPath, error) {
	var paths []system.TbGenerateProjectPath
	db := global.GVA_DB.Where("project_instance_id = ? AND enabled = 1", projectInstanceId)
	if len(pathIds) > 0 {
		db = db.Where("id IN ?", pathIds)
	} else {
		db = db.Where("path_set = ?", pathSet)
	}
	err := db.Order("id ASC").Find(&paths).Error
	return paths, err
}

func loadGenerateProjectPathContents(paths []system.TbGenerateProjectPath) (map[int]string, error) {
	pathIds := make([]uint, 0, len(paths))
	for _, pathObj := range paths {
		pathIds = append(pathIds, pathObj.ID)
	}

	var models []system.TbGenerateProjectPathModel
	if err := global.GVA_DB.Where("path_id IN ?", pathIds).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	contents := make(map[int]string, len(paths))
	for _, model := range models {
		if _, exists := contents[model.PathId]; !exists {
			contents[model.PathId] = model.Content
		}
	}
	return contents, nil
}

func normalizeCodeGenerationRoot(rawPath string) (string, error) {
	root := strings.TrimSpace(rawPath)
	if root == "" {
		return "", errors.New("磁盘输出路径不能为空")
	}
	if strings.HasPrefix(root, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(homeDir, strings.TrimPrefix(root, "~/"))
	}
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		root = absRoot
	}
	return root, nil
}

func buildGeneratedFileTarget(root string, fileUrl string, fileName string) (string, string, error) {
	rawPath := strings.TrimSpace(strings.ReplaceAll(filepath.ToSlash(filepath.Join(fileUrl, fileName)), "\\", "/"))
	rawPath = strings.TrimLeft(rawPath, "/")
	if rawPath == "" || rawPath == "." {
		return "", "", errors.New("文件路径不能为空")
	}

	relativePath := filepath.Clean(filepath.FromSlash(rawPath))
	if relativePath == "." || relativePath == string(filepath.Separator) {
		return "", "", errors.New("文件路径不能为空")
	}
	if filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", "", errors.New("文件路径不能跳出磁盘输出路径")
	}

	targetPath := filepath.Join(root, relativePath)
	relToRoot, err := filepath.Rel(root, targetPath)
	if err != nil {
		return "", "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relToRoot) {
		return "", "", errors.New("文件路径不能跳出磁盘输出路径")
	}

	return filepath.ToSlash(relToRoot), targetPath, nil
}

func buildCodeGenerationVars(module string, tableName string) map[string]string {
	return map[string]string{
		"module":     module,
		"Module":     upperFirst(module),
		"MODULE":     strings.ToUpper(module),
		"moduleName": module,
		"ModuleName": upperFirst(module),
		"TableName":  tableName,
		"tableName":  lowerFirst(tableName),
		"TABLE_NAME": strings.ToUpper(toSnakeCase(tableName)),
		"table_name": toSnakeCase(tableName),
	}
}

func renderCodeGenerationText(text string, vars map[string]string) string {
	rendered := codePlaceholderPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := codePlaceholderPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if value, ok := vars[parts[1]]; ok {
			return value
		}
		return match
	})

	replacer := strings.NewReplacer(
		"{[<moduleName>]}", vars["moduleName"],
		"{[<ModuleName>]}", vars["ModuleName"],
		"{[<TableName>]}", vars["TableName"],
		"{[<tableName>]}", vars["tableName"],
	)
	return replacer.Replace(rendered)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func upperFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func lowerFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func toSnakeCase(value string) string {
	var builder strings.Builder
	var previous rune
	for index, current := range value {
		if unicode.IsUpper(current) {
			if index > 0 && (unicode.IsLower(previous) || unicode.IsDigit(previous)) {
				builder.WriteRune('_')
			}
			builder.WriteRune(unicode.ToLower(current))
		} else if current == '-' || current == ' ' {
			builder.WriteRune('_')
		} else {
			builder.WriteRune(current)
		}
		previous = current
	}
	return builder.String()
}
