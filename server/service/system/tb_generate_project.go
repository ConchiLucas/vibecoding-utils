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
	AbsolutePath string `json:"absolutePath"`
	RelativePath string `json:"relativePath"`
	PathId       uint   `json:"pathId"`
	Status       string `json:"status"`
	Bytes        int    `json:"bytes"`
	Instruction  string `json:"instruction"`
}

type GenerateProjectCodeResult struct {
	TemplateProjectId  int                       `json:"templateProjectId"`
	ProjectInstanceId  int                       `json:"projectInstanceId"`
	ProjectName        string                    `json:"projectName"`
	DiskPath           string                    `json:"diskPath"`
	PathSet            int                       `json:"pathSet"`
	Prompt             string                    `json:"prompt"`
	PromptUrl          string                    `json:"promptUrl"`
	ModifyInstructions string                    `json:"modifyInstructions"`
	TargetPaths        []string                  `json:"targetPaths"`
	GeneratedCount     int                       `json:"generatedCount"`
	SkippedCount       int                       `json:"skippedCount"`
	Files              []GenerateProjectCodeFile `json:"files"`
}

var codePlaceholderPattern = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}`)

const codeGenerationModifyInstructions = `请读取每个目标文件的绝对路径。目标文件当前由代码模板生成，每个文件还有独立的文件提示词，请结合产品文档把模板改造成最终可用代码。生成最终代码时删除模板提示说明，只保留必要的业务注释；package、import、类名、SQL id、字段和方法都按实际模块与项目上下文调整；字段只保留前端或业务真正使用的字段，不要机械罗列数据库所有字段；示例字段只作为结构参考，不符合业务时替换。`

type generateProjectCodeDraft struct {
	File            GenerateProjectCodeFile
	TemplateContent string
	FilePrompt      string
}

type generateProjectPathTemplate struct {
	Content string
	Prompt  string
}

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

	if err := tx.Where("project_id = ? AND (project_instance_id = 0 OR project_instance_id IS NULL)", req.ID).Unscoped().Delete(&system.TbGenerateProjectPathGroup{}).Error; err != nil {
		tx.Rollback()
		return err
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

	templates, err := loadGenerateProjectPathTemplates(paths)
	if err != nil {
		return GenerateProjectCodeResult{}, err
	}

	vars := buildCodeGenerationVars(module, tableName)
	result := GenerateProjectCodeResult{
		TemplateProjectId:  int(project.ID),
		ProjectInstanceId:  int(instance.ID),
		ProjectName:        instance.ProjectName,
		DiskPath:           diskPath,
		PathSet:            pathSet,
		ModifyInstructions: codeGenerationModifyInstructions,
		TargetPaths:        make([]string, 0, len(paths)),
		Files:              make([]GenerateProjectCodeFile, 0, len(paths)),
	}
	drafts := make([]generateProjectCodeDraft, 0, len(paths))

	for _, pathObj := range paths {
		relativePath, targetPath, err := renderGeneratedFileTarget(diskPath, pathObj.FileUrl, pathObj.FileName, vars)
		if err != nil {
			return GenerateProjectCodeResult{}, fmt.Errorf("路径 %d 无效: %w", pathObj.ID, err)
		}

		pathTemplate := templates[int(pathObj.ID)]
		content := renderCodeGenerationText(pathTemplate.Content, vars)
		filePrompt := renderCodeGenerationText(pathTemplate.Prompt, vars)
		fileResult := GenerateProjectCodeFile{
			Path:         targetPath,
			AbsolutePath: targetPath,
			RelativePath: relativePath,
			PathId:       pathObj.ID,
			Status:       "generated",
			Bytes:        len([]byte(content)),
			Instruction:  buildGenerateCodeFileInstruction(relativePath, targetPath),
		}
		result.TargetPaths = append(result.TargetPaths, targetPath)

		fileStatus := "generated"
		if _, err := os.Stat(targetPath); err == nil {
			if !req.Overwrite {
				result.SkippedCount++
				fileResult.Status = "skipped"
				result.Files = append(result.Files, fileResult)
				drafts = append(drafts, generateProjectCodeDraft{
					File:            fileResult,
					TemplateContent: content,
					FilePrompt:      filePrompt,
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
		fileResult.Status = fileStatus
		result.Files = append(result.Files, fileResult)
		drafts = append(drafts, generateProjectCodeDraft{
			File:            fileResult,
			TemplateContent: content,
			FilePrompt:      filePrompt,
		})
	}

	result.Prompt = buildCodeGenerationTaskPromptContent(module, tableName, req.Overwrite, result.ProjectName, result.DiskPath, result.PathSet, drafts)

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

func (s *TbGenerateProjectService) CopyProject(req systemReq.CopyGenerateProjectReq) (system.TbGenerateProject, error) {
	if req.SourceProjectId <= 0 {
		return system.TbGenerateProject{}, errors.New("sourceProjectId 必填")
	}

	var project system.TbGenerateProject
	if err := global.GVA_DB.Where("id = ?", req.SourceProjectId).First(&project).Error; err != nil {
		return system.TbGenerateProject{}, err
	}

	newProject := system.TbGenerateProject{
		ProjectConfigId: 0,
		BusinessType:    strings.TrimSpace(firstNonEmptyString(req.BusinessType, project.BusinessType)),
		ProjectType:     strings.TrimSpace(firstNonEmptyString(req.ProjectType, project.ProjectType)),
		ProjectName:     strings.TrimSpace(firstNonEmptyString(req.ProjectName, project.ProjectName+"_copy")),
		DiskPath:        project.DiskPath,
		Remark:          strings.TrimSpace(firstNonEmptyString(req.Remark, project.Remark)),
		UserName:        strings.TrimSpace(firstNonEmptyString(req.UserName, project.UserName)),
	}
	if newProject.ProjectName == "" {
		return system.TbGenerateProject{}, errors.New("projectName 必填")
	}
	if newProject.ProjectType == "" {
		newProject.ProjectType = "backend"
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return system.TbGenerateProject{}, err
	}
	if err := tx.Create(&newProject).Error; err != nil {
		tx.Rollback()
		return system.TbGenerateProject{}, err
	}

	pathService := &TbGenerateProjectPathService{}
	if err := pathService.copyGenerateProjectPathScope(tx, int(project.ID), 0, int(newProject.ID), 0); err != nil {
		tx.Rollback()
		return system.TbGenerateProject{}, err
	}
	if err := s.copyProjectInstancesTx(tx, project, &newProject); err != nil {
		tx.Rollback()
		return system.TbGenerateProject{}, err
	}
	if err := s.copyProjectDbTemplatesTx(tx, int(project.ID), int(newProject.ID)); err != nil {
		tx.Rollback()
		return system.TbGenerateProject{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return system.TbGenerateProject{}, err
	}
	return newProject, nil
}

func (s *TbGenerateProjectService) copyProjectInstancesTx(tx *gorm.DB, sourceProject system.TbGenerateProject, targetProject *system.TbGenerateProject) error {
	var instances []system.TbGenerateProjectInstance
	if err := tx.Where("template_project_id = ?", sourceProject.ID).Order("id ASC").Find(&instances).Error; err != nil {
		return err
	}

	selectedInstanceMap := make(map[int]int, len(instances))
	for _, instance := range instances {
		newInstance := system.TbGenerateProjectInstance{
			TemplateProjectId:       int(targetProject.ID),
			ProjectName:             instance.ProjectName,
			DiskPath:                instance.DiskPath,
			Remark:                  instance.Remark,
			UserName:                instance.UserName,
			SelectedPathSetIdentity: instance.SelectedPathSetIdentity,
		}
		if err := tx.Create(&newInstance).Error; err != nil {
			return err
		}
		selectedInstanceMap[int(instance.ID)] = int(newInstance.ID)

		if err := (&TbGenerateProjectPathService{}).copyGenerateProjectPathScope(tx, int(instance.ID), int(instance.ID), int(newInstance.ID), int(newInstance.ID)); err != nil {
			return err
		}
	}

	if nextSelectedInstanceId := selectedInstanceMap[sourceProject.SelectedProjectInstanceId]; nextSelectedInstanceId > 0 {
		targetProject.SelectedProjectInstanceId = nextSelectedInstanceId
		return tx.Model(targetProject).Update("selected_project_instance_id", nextSelectedInstanceId).Error
	}
	return nil
}

func (s *TbGenerateProjectService) copyProjectDbTemplatesTx(tx *gorm.DB, sourceProjectId int, targetProjectId int) error {
	var templateTypes []system.TbGenerateDbTemplateType
	if err := tx.Where("project_id = ?", sourceProjectId).Order("sort ASC, id ASC").Find(&templateTypes).Error; err != nil {
		return err
	}
	for _, templateType := range templateTypes {
		oldTypeId := templateType.ID
		newType := system.TbGenerateDbTemplateType{
			ProjectId: targetProjectId,
			TypeName:  templateType.TypeName,
			Prompt:    templateType.Prompt,
			Sort:      templateType.Sort,
		}
		if err := tx.Create(&newType).Error; err != nil {
			return err
		}

		var scripts []system.TbGenerateDbTemplateScript
		if err := tx.Where("type_id = ?", oldTypeId).Order("sort ASC, id ASC").Find(&scripts).Error; err != nil {
			return err
		}
		for _, script := range scripts {
			newScript := system.TbGenerateDbTemplateScript{
				ProjectId:  targetProjectId,
				TypeId:     int(newType.ID),
				ScriptName: script.ScriptName,
				ScriptKind: script.ScriptKind,
				Content:    script.Content,
				Sort:       script.Sort,
			}
			if err := tx.Create(&newScript).Error; err != nil {
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

func loadGenerateProjectPathTemplates(paths []system.TbGenerateProjectPath) (map[int]generateProjectPathTemplate, error) {
	pathIds := make([]uint, 0, len(paths))
	for _, pathObj := range paths {
		pathIds = append(pathIds, pathObj.ID)
	}

	var models []system.TbGenerateProjectPathModel
	if err := global.GVA_DB.Where("path_id IN ?", pathIds).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	templates := make(map[int]generateProjectPathTemplate, len(paths))
	for _, model := range models {
		if _, exists := templates[model.PathId]; !exists {
			templates[model.PathId] = generateProjectPathTemplate{
				Content: model.Content,
				Prompt:  model.Prompt,
			}
		}
	}
	return templates, nil
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

func renderGeneratedFileTarget(root string, fileUrl string, fileName string, vars map[string]string) (string, string, error) {
	return buildGeneratedFileTarget(
		root,
		renderCodeGenerationText(fileUrl, vars),
		renderCodeGenerationText(fileName, vars),
	)
}

func buildCodeGenerationTaskPromptContent(module string, tableName string, overwrite bool, projectName string, diskPath string, pathSet int, drafts []generateProjectCodeDraft) string {
	var builder strings.Builder
	builder.WriteString("# Codex 代码生成任务\n\n")
	builder.WriteString("请按下面的绝对路径逐个读取目标文件，把每个带提示词或模板内容的文件改造成最终可用代码。\n\n")

	builder.WriteString("## 输入参数\n\n")
	builder.WriteString(fmt.Sprintf("- 项目: %s\n", markdownInline(projectName)))
	builder.WriteString(fmt.Sprintf("- module: `%s`\n", module))
	builder.WriteString(fmt.Sprintf("- TableName: `%s`\n", tableName))
	builder.WriteString(fmt.Sprintf("- 磁盘输出路径: `%s`\n", diskPath))
	builder.WriteString(fmt.Sprintf("- pathSet: `%d`\n", pathSet))
	builder.WriteString(fmt.Sprintf("- 覆盖已存在文件: `%t`\n\n", overwrite))

	builder.WriteString("## 总体修改规则\n\n")
	for index, rule := range []string{
		"读取每个目标文件的绝对路径，目标文件当前是代码模板生成的文件或已有代码。",
		"每个目标文件如果提供了“文件提示词”，优先按该文件提示词理解这个文件的职责和改造边界。",
		"生成最终代码时删除“根据实际情况修改”“示例”“提示词”等说明，只保留必要业务注释。",
		"package、import、类名、SQL id、字段和方法都要按当前项目上下文、module 和 TableName 调整。",
		"字段只保留前端或业务真正使用的字段，不要机械罗列数据库所有字段。",
		"id、stationCode 等示例字段只作为参考，不符合实际业务时替换。",
		"状态为 skipped 的目标文件没有被本次写入模板内容，如需参考模板，请看本提示词中对应的“模板内容”。",
	} {
		builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, rule))
	}
	builder.WriteString("\n")

	builder.WriteString("## 目标文件\n\n")
	for index, draft := range drafts {
		builder.WriteString(fmt.Sprintf("### %d. `%s`\n\n", index+1, draft.File.RelativePath))
		builder.WriteString(fmt.Sprintf("- 绝对路径: `%s`\n", draft.File.AbsolutePath))
		builder.WriteString(fmt.Sprintf("- 状态: `%s`\n", draft.File.Status))
		builder.WriteString(fmt.Sprintf("- 修改方式: %s\n\n", draft.File.Instruction))
		if strings.TrimSpace(draft.FilePrompt) != "" {
			builder.WriteString("文件提示词:\n\n")
			appendMarkdownFence(&builder, "text", draft.FilePrompt)
			builder.WriteString("\n")
		}
		builder.WriteString("模板内容:\n\n")
		appendMarkdownFence(&builder, guessMarkdownFenceLanguage(draft.File.RelativePath), draft.TemplateContent)
		builder.WriteString("\n")
	}

	builder.WriteString("## 输出要求\n\n")
	builder.WriteString("- 直接修改上面列出的目标文件。\n")
	builder.WriteString("- 不要新增无关文件。\n")
	builder.WriteString("- 修改完成后说明哪些文件已生成或更新。\n")
	return builder.String()
}

func buildGenerateCodeFileInstruction(relativePath string, absolutePath string) string {
	return fmt.Sprintf("读取 `%s`，结合文件提示词、代码模板和产品文档改写为最终业务代码；最终文件路径保持为 `%s`。", absolutePath, relativePath)
}

func markdownInline(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return strings.ReplaceAll(value, "\n", " ")
}

func guessMarkdownFenceLanguage(relativePath string) string {
	switch strings.ToLower(filepath.Ext(relativePath)) {
	case ".java":
		return "java"
	case ".sql":
		return "sql"
	case ".xml":
		return "xml"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".vue":
		return "vue"
	case ".go":
		return "go"
	default:
		return "text"
	}
}

func appendMarkdownFence(builder *strings.Builder, language string, content string) {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	builder.WriteString(fence)
	if language != "" {
		builder.WriteString(language)
	}
	builder.WriteString("\n")
	builder.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString(fence)
	builder.WriteString("\n")
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
