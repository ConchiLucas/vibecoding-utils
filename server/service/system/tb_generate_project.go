package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGenerateProjectService struct{}

func (s *TbGenerateProjectService) CreateTbGenerateProject(req *system.TbGenerateProject) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectService) DeleteTbGenerateProject(req system.TbGenerateProject) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	// 1. Find all paths belonging to this project
	var paths []system.TbGenerateProjectPath
	if err := tx.Where("project_id = ?", req.ID).Find(&paths).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. For each path, delete its models, then the path itself
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

	// 3. Delete database template examples belonging to this project
	if err := tx.Where("project_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateDbTemplateScript{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("project_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateDbTemplateType{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 4. Delete the project itself
	if err := tx.Unscoped().Delete(&req).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *TbGenerateProjectService) UpdateTbGenerateProject(req *system.TbGenerateProject) error {
	return global.GVA_DB.Updates(req).Error
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
		ProjectConfigId: project.ProjectConfigId,
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
			ProjectId:   int(newProject.ID),
			Enabled:     p.Enabled,
			FileUrl:     p.FileUrl,
			FileName:    p.FileName,
			Incremented: p.Incremented,
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
