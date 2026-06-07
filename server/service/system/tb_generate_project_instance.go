package system

import (
	"errors"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"gorm.io/gorm"
)

type TbGenerateProjectInstanceService struct{}

func (s *TbGenerateProjectInstanceService) CreateTbGenerateProjectInstance(req *system.TbGenerateProjectInstance) error {
	if req.TemplateProjectId <= 0 {
		return errors.New("templateProjectId 必填")
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Create(req).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := s.cloneTemplatePaths(tx, req.TemplateProjectId, int(req.ID)); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *TbGenerateProjectInstanceService) DeleteTbGenerateProjectInstance(req system.TbGenerateProjectInstance) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	if err := deleteGenerateProjectInstancePaths(tx, int(req.ID)); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Unscoped().Delete(&req).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *TbGenerateProjectInstanceService) UpdateTbGenerateProjectInstance(req *system.TbGenerateProjectInstance) error {
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateProjectInstanceService) UpdateSelectedPathSet(projectInstanceId int, selectedPathSetIdentity string) error {
	if projectInstanceId <= 0 {
		return errors.New("projectInstanceId 必填")
	}

	return global.GVA_DB.Model(&system.TbGenerateProjectInstance{}).
		Where("id = ?", projectInstanceId).
		Update("selected_path_set_identity", strings.TrimSpace(selectedPathSetIdentity)).Error
}

func (s *TbGenerateProjectInstanceService) GetTbGenerateProjectInstance(id string) (res system.TbGenerateProjectInstance, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectInstanceService) GetTbGenerateProjectInstanceList(templateProjectId int, ensureDefault bool) (res []system.TbGenerateProjectInstance, err error) {
	db := global.GVA_DB.Model(&system.TbGenerateProjectInstance{})
	if templateProjectId > 0 {
		db = db.Where("template_project_id = ?", templateProjectId)
	}
	err = db.Order("id ASC").Find(&res).Error
	if err != nil {
		return
	}

	if templateProjectId > 0 && ensureDefault && len(res) == 0 {
		var created system.TbGenerateProjectInstance
		created, err = s.createDefaultFromTemplate(templateProjectId)
		if err != nil {
			return
		}
		res = []system.TbGenerateProjectInstance{created}
	}
	return
}

func (s *TbGenerateProjectInstanceService) createDefaultFromTemplate(templateProjectId int) (system.TbGenerateProjectInstance, error) {
	var template system.TbGenerateProject
	if err := global.GVA_DB.Where("id = ?", templateProjectId).First(&template).Error; err != nil {
		return system.TbGenerateProjectInstance{}, err
	}

	instance := system.TbGenerateProjectInstance{
		TemplateProjectId: int(template.ID),
		ProjectName:       template.ProjectName,
		DiskPath:          template.DiskPath,
		Remark:            template.Remark,
		UserName:          template.UserName,
	}
	err := s.CreateTbGenerateProjectInstance(&instance)
	return instance, err
}

func (s *TbGenerateProjectInstanceService) cloneTemplatePaths(tx *gorm.DB, templateProjectId int, instanceProjectId int) error {
	if err := (&TbGenerateProjectPathService{}).ensurePathGroupsForLegacyPathsTx(tx, templateProjectId, 0); err != nil {
		return err
	}

	var templateGroups []system.TbGenerateProjectPathGroup
	if err := tx.Where("project_id = ? AND (project_instance_id = 0 OR project_instance_id IS NULL)", templateProjectId).Order("path_set ASC, sort ASC, id ASC").Find(&templateGroups).Error; err != nil {
		return err
	}
	groupIdMap := make(map[int]int, len(templateGroups))
	for _, templateGroup := range templateGroups {
		newGroup := system.TbGenerateProjectPathGroup{
			ProjectId:         instanceProjectId,
			ProjectInstanceId: instanceProjectId,
			PathSet:           templateGroup.PathSet,
			PathSetName:       templateGroup.PathSetName,
			BasePath:          templateGroup.BasePath,
			Sort:              templateGroup.Sort,
		}
		if err := tx.Create(&newGroup).Error; err != nil {
			return err
		}
		groupIdMap[int(templateGroup.ID)] = int(newGroup.ID)
	}

	var templatePaths []system.TbGenerateProjectPath
	if err := tx.Where("project_id = ? AND (project_instance_id = 0 OR project_instance_id IS NULL)", templateProjectId).Order("id ASC").Find(&templatePaths).Error; err != nil {
		return err
	}

	for _, templatePath := range templatePaths {
		oldPathId := templatePath.ID
		newPath := system.TbGenerateProjectPath{
			ProjectId:         instanceProjectId,
			ProjectInstanceId: instanceProjectId,
			PathSet:           templatePath.PathSet,
			PathSetName:       templatePath.PathSetName,
			PathGroupId:       groupIdMap[templatePath.PathGroupId],
			FileUrl:           templatePath.FileUrl,
			FileName:          templatePath.FileName,
			Enabled:           templatePath.Enabled,
			Incremented:       templatePath.Incremented,
		}
		if err := tx.Create(&newPath).Error; err != nil {
			return err
		}

		var templateModel system.TbGenerateProjectPathModel
		if err := tx.Where("path_id = ?", oldPathId).First(&templateModel).Error; err == nil {
			newModel := system.TbGenerateProjectPathModel{
				PathId:  int(newPath.ID),
				Content: templateModel.Content,
				Prompt:  templateModel.Prompt,
			}
			if err := tx.Create(&newModel).Error; err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func deleteGenerateProjectInstancePaths(tx *gorm.DB, projectId int) error {
	var paths []system.TbGenerateProjectPath
	if err := tx.Where("project_instance_id = ?", projectId).Find(&paths).Error; err != nil {
		return err
	}
	for _, p := range paths {
		if err := tx.Where("path_id = ?", p.ID).Unscoped().Delete(&system.TbGenerateProjectPathModel{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&p).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("project_instance_id = ?", projectId).Unscoped().Delete(&system.TbGenerateProjectPathGroup{}).Error; err != nil {
		return err
	}
	return nil
}
