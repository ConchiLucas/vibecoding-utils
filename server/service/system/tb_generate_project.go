package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"strings"
)

type TbGenerateProjectService struct{}

func (s *TbGenerateProjectService) CreateTbGenerateProject(req *system.TbGenerateProject) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectService) DeleteTbGenerateProject(req system.TbGenerateProject) error {
	tx := global.GVA_DB.Begin()

	// 1. Find all paths belonging to this project
	var paths []system.TbGenerateProjectPath
	tx.Where("project_id = ?", req.ID).Find(&paths)

	// 2. For each path, delete its models, then the path itself
	for _, p := range paths {
		tx.Where("path_id = ?", p.ID).Unscoped().Delete(&system.TbGenerateProjectPathModel{})
		tx.Unscoped().Delete(&p)
	}

	// 3. Delete all project-level placeholders
	tx.Where("project_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateProjectPlaceHolder{})

	// 4. Delete the project itself
	tx.Unscoped().Delete(&req)

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

	var holders []system.TbGenerateProjectPlaceHolder
	global.GVA_DB.Where("project_id = ?", id).Find(&holders)
	for _, h := range holders {
		newHolder := system.TbGenerateProjectPlaceHolder{
			ProjectId:    int(newProject.ID),
			UserName:     h.UserName,
			HolderKey:    h.HolderKey,
			HolderValue:  h.HolderValue,
			HolderDesc:   h.HolderDesc,
			ExampleValue: h.ExampleValue,
		}
		global.GVA_DB.Create(&newHolder)
	}
	return nil
}

func (s *TbGenerateProjectService) ProjectGlobalReplace(id int, formerStr, replaceStr string) error {
	tx := global.GVA_DB.Begin()
	var paths []system.TbGenerateProjectPath
	tx.Where("project_id = ?", id).Find(&paths)

	for _, p := range paths {
		if strings.Contains(p.FileUrl, formerStr) {
			p.FileUrl = strings.ReplaceAll(p.FileUrl, formerStr, replaceStr)
			tx.Save(&p)
		}
		var model system.TbGenerateProjectPathModel
		if err := tx.Where("path_id = ?", p.ID).First(&model).Error; err == nil {
			if strings.Contains(model.Content, formerStr) {
				model.Content = strings.ReplaceAll(model.Content, formerStr, replaceStr)
				tx.Save(&model)
			}
		}
	}

	var holders []system.TbGenerateProjectPlaceHolder
	tx.Where("project_id = ?", id).Find(&holders)
	for _, h := range holders {
		if strings.Contains(h.HolderKey, formerStr) {
			h.HolderKey = strings.ReplaceAll(h.HolderKey, formerStr, replaceStr)
			tx.Save(&h)
		}
	}
	return tx.Commit().Error
}
