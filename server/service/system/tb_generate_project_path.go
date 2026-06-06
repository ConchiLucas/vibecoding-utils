package system

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbGenerateProjectPathService struct{}

type GenerateProjectPromptSummaryResult struct {
	ProjectInstanceId  int                       `json:"projectInstanceId"`
	ProjectName        string                    `json:"projectName"`
	DiskPath           string                    `json:"diskPath"`
	PathSet            int                       `json:"pathSet"`
	Module             string                    `json:"module"`
	TableName          string                    `json:"tableName"`
	Prompt             string                    `json:"prompt"`
	ModifyInstructions string                    `json:"modifyInstructions"`
	TargetPaths        []string                  `json:"targetPaths"`
	Files              []GenerateProjectCodeFile `json:"files"`
}

func (s *TbGenerateProjectPathService) CreateTbGenerateProjectPath(req *system.TbGenerateProjectPath) error {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectPathService) DeleteTbGenerateProjectPath(req system.TbGenerateProjectPath) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Where("path_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateProjectPathModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Unscoped().Delete(&req).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *TbGenerateProjectPathService) DeletePathSet(req systemReq.DeleteGenerateProjectPathSetReq) (int, error) {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return 0, errors.New("projectInstanceId is required")
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return 0, err
	}

	var paths []system.TbGenerateProjectPath
	query := tx.Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		query = query.Where("id IN ?", req.PathIds)
	} else {
		query = query.Where("path_set = ?", req.PathSet)
	}
	if err := query.Find(&paths).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	if len(paths) == 0 {
		tx.Rollback()
		return 0, errors.New("paths are empty")
	}

	for _, pathObj := range paths {
		if err := tx.Where("path_id = ?", pathObj.ID).Unscoped().Delete(&system.TbGenerateProjectPathModel{}).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		if err := tx.Unscoped().Delete(&pathObj).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	return len(paths), tx.Commit().Error
}

func (s *TbGenerateProjectPathService) UpdateTbGenerateProjectPath(req *system.TbGenerateProjectPath) error {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateProjectPathService) RenamePathSet(req systemReq.RenameGenerateProjectPathSetReq) (int64, error) {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return 0, errors.New("projectInstanceId is required")
	}

	query := global.GVA_DB.Model(&system.TbGenerateProjectPath{}).
		Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		query = query.Where("id IN ?", req.PathIds)
	} else {
		query = query.Where("path_set = ?", req.PathSet)
	}

	result := query.Update("path_set_name", strings.TrimSpace(req.PathSetName))
	return result.RowsAffected, result.Error
}

func (s *TbGenerateProjectPathService) CopyPathSet(req systemReq.CopyGenerateProjectPathSetReq) (int, error) {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return 0, errors.New("projectInstanceId is required")
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return 0, err
	}

	var maxPathSet int
	if err := tx.Model(&system.TbGenerateProjectPath{}).
		Where("project_instance_id = ?", req.ProjectInstanceId).
		Select("COALESCE(MAX(path_set), 0)").
		Scan(&maxPathSet).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	nextPathSet := maxPathSet + 1

	var sourcePaths []system.TbGenerateProjectPath
	sourceQuery := tx.Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		sourceQuery = sourceQuery.Where("id IN ?", req.PathIds)
	} else {
		sourceQuery = sourceQuery.Where("path_set = ?", req.PathSet)
	}
	if err := sourceQuery.Order("id ASC").Find(&sourcePaths).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	if len(sourcePaths) == 0 {
		tx.Rollback()
		return 0, errors.New("source paths are empty")
	}

	for _, sourcePath := range sourcePaths {
		oldPathId := sourcePath.ID
		newPath := system.TbGenerateProjectPath{
			ProjectId:         sourcePath.ProjectId,
			ProjectInstanceId: req.ProjectInstanceId,
			PathSet:           nextPathSet,
			PathSetName:       sourcePath.PathSetName,
			FileUrl:           sourcePath.FileUrl,
			FileName:          sourcePath.FileName,
			Enabled:           sourcePath.Enabled,
			Incremented:       sourcePath.Incremented,
		}
		if newPath.ProjectId == 0 {
			newPath.ProjectId = req.ProjectId
		}
		if newPath.ProjectId == 0 {
			newPath.ProjectId = req.ProjectInstanceId
		}
		if err := tx.Create(&newPath).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		var sourceModels []system.TbGenerateProjectPathModel
		if err := tx.Where("path_id = ?", oldPathId).Order("id ASC").Find(&sourceModels).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		for _, sourceModel := range sourceModels {
			newModel := system.TbGenerateProjectPathModel{
				PathId:  int(newPath.ID),
				Content: sourceModel.Content,
				Prompt:  sourceModel.Prompt,
			}
			if err := tx.Create(&newModel).Error; err != nil {
				tx.Rollback()
				return 0, err
			}
		}
	}

	return nextPathSet, tx.Commit().Error
}

func (s *TbGenerateProjectPathService) GetTbGenerateProjectPath(id string) (res system.TbGenerateProjectPath, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectPathService) GetTbGenerateProjectPathList(projectId int, projectInstanceId int) (res []system.TbGenerateProjectPath, err error) {
	db := global.GVA_DB.Model(&system.TbGenerateProjectPath{})
	if projectInstanceId > 0 {
		db = db.Where("project_instance_id = ?", projectInstanceId)
	} else if projectId > 0 {
		db = db.Where("project_id = ?", projectId)
	}
	err = db.Order("id ASC").Find(&res).Error
	return
}

func (s *TbGenerateProjectPathService) UpdateEnabled(id uint, enabled int) error {
	return global.GVA_DB.Model(&system.TbGenerateProjectPath{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (s *TbGenerateProjectPathService) BuildPromptSummary(req systemReq.BuildGenerateProjectPromptSummaryReq) (GenerateProjectPromptSummaryResult, error) {
	if req.ProjectInstanceId <= 0 {
		return GenerateProjectPromptSummaryResult{}, errors.New("projectInstanceId is required")
	}
	module := strings.TrimSpace(req.Module)
	tableName := strings.TrimSpace(req.TableName)
	if module == "" {
		return GenerateProjectPromptSummaryResult{}, errors.New("module 必填")
	}
	if tableName == "" {
		return GenerateProjectPromptSummaryResult{}, errors.New("TableName 必填")
	}

	var instance system.TbGenerateProjectInstance
	if err := global.GVA_DB.Where("id = ?", req.ProjectInstanceId).First(&instance).Error; err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}

	diskPathSource := instance.DiskPath
	if strings.TrimSpace(diskPathSource) == "" && instance.TemplateProjectId > 0 {
		var project system.TbGenerateProject
		if err := global.GVA_DB.Where("id = ?", instance.TemplateProjectId).First(&project).Error; err != nil {
			return GenerateProjectPromptSummaryResult{}, err
		}
		diskPathSource = project.DiskPath
	}
	diskPath, err := normalizeCodeGenerationRoot(diskPathSource)
	if err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}

	var paths []system.TbGenerateProjectPath
	query := global.GVA_DB.Where("project_instance_id = ? AND enabled = 1", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		query = query.Where("id IN ?", req.PathIds)
	} else {
		query = query.Where("path_set = ?", req.PathSet)
	}
	if err := query.Order("id ASC").Find(&paths).Error; err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}
	if len(paths) == 0 {
		return GenerateProjectPromptSummaryResult{}, errors.New("当前相对路径没有可生成提示词的启用文件")
	}

	templates, err := loadGenerateProjectPathTemplates(paths)
	if err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}

	vars := buildCodeGenerationVars(module, tableName)
	result := GenerateProjectPromptSummaryResult{
		ProjectInstanceId:  int(instance.ID),
		ProjectName:        instance.ProjectName,
		DiskPath:           diskPath,
		PathSet:            req.PathSet,
		Module:             module,
		TableName:          tableName,
		ModifyInstructions: codeGenerationModifyInstructions,
		TargetPaths:        make([]string, 0, len(paths)),
		Files:              make([]GenerateProjectCodeFile, 0, len(paths)),
	}
	drafts := make([]generateProjectCodeDraft, 0, len(paths))

	for _, pathObj := range paths {
		relativePath, targetPath, err := buildGeneratedFileTarget(
			diskPath,
			renderCodeGenerationText(pathObj.FileUrl, vars),
			renderCodeGenerationText(pathObj.FileName, vars),
		)
		if err != nil {
			return GenerateProjectPromptSummaryResult{}, fmt.Errorf("路径 %d 无效: %w", pathObj.ID, err)
		}

		pathTemplate := templates[int(pathObj.ID)]
		content := renderCodeGenerationText(pathTemplate.Content, vars)
		filePrompt := renderCodeGenerationText(pathTemplate.Prompt, vars)
		file := GenerateProjectCodeFile{
			Path:         targetPath,
			AbsolutePath: targetPath,
			RelativePath: relativePath,
			PathId:       pathObj.ID,
			Status:       "ready",
			Bytes:        len([]byte(content)),
			Instruction:  buildGenerateCodeFileInstruction(relativePath, targetPath),
		}
		result.TargetPaths = append(result.TargetPaths, targetPath)
		result.Files = append(result.Files, file)
		drafts = append(drafts, generateProjectCodeDraft{
			File:            file,
			TemplateContent: content,
			FilePrompt:      filePrompt,
		})
	}

	result.Prompt = buildCodeGenerationTaskPromptContent(module, tableName, req.Overwrite, result.ProjectName, result.DiskPath, result.PathSet, drafts)
	return result, nil
}
