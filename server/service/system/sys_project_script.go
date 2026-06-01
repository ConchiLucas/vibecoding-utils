package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type ProjectScriptService struct{}

var ProjectScriptServiceApp = new(ProjectScriptService)

// GetProjectScriptPage 分页获取项目脚本列表
func (s *ProjectScriptService) GetProjectScriptPage(req request.TbProjectScriptSearch) (list []system.TbProjectScript, total int64, err error) {
	db := global.GVA_DB.Model(&system.TbProjectScript{})
	if req.ProjectId != 0 {
		db = db.Where("project_id = ?", req.ProjectId)
	}
	if req.RouteId != 0 {
		db = db.Where("route_id = ?", req.RouteId)
	}
	if req.FileName != "" {
		db = db.Where("file_name LIKE ?", "%"+req.FileName+"%")
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Scopes(req.Paginate()).Order("id desc").Find(&list).Error
	return
}

// GetProjectScriptList 获取项目脚本列表（不分页）
func (s *ProjectScriptService) GetProjectScriptList(projectId int, routeId int) (list []system.TbProjectScript, err error) {
	db := global.GVA_DB.Model(&system.TbProjectScript{})
	if projectId != 0 {
		db = db.Where("project_id = ?", projectId)
	}
	if routeId != 0 {
		db = db.Where("route_id = ?", routeId)
	}
	err = db.Find(&list).Error
	return
}

// GetProjectScriptById 根据ID获取项目脚本
func (s *ProjectScriptService) GetProjectScriptById(id uint) (script system.TbProjectScript, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&script).Error
	return
}

// SaveOrUpdateProjectScript 新增或修改项目脚本
func (s *ProjectScriptService) SaveOrUpdateProjectScript(script system.TbProjectScript) (err error) {
	if script.ID != 0 {
		err = global.GVA_DB.Model(&system.TbProjectScript{}).Where("id = ?", script.ID).Updates(map[string]interface{}{
			"project_id":     script.ProjectId,
			"route_id":       script.RouteId,
			"content":        script.Content,
			"file_name":      script.FileName,
			"file_nick_name": script.FileNickName,
		}).Error
	} else {
		err = global.GVA_DB.Create(&script).Error
	}
	return
}

// GetProjectScriptByIds 根据ID列表批量获取项目脚本
func (s *ProjectScriptService) GetProjectScriptByIds(ids []int) (list []system.TbProjectScript, err error) {
	err = global.GVA_DB.Where("id IN ?", ids).Find(&list).Error
	return
}

// DeleteProjectScript 批量删除项目脚本
func (s *ProjectScriptService) DeleteProjectScript(ids []int) (err error) {
	err = global.GVA_DB.Where("id IN ?", ids).Unscoped().Delete(&system.TbProjectScript{}).Error
	return
}

// UpdateScriptContent 更新脚本文件内容（DB直接存储）
func (s *ProjectScriptService) UpdateScriptContent(id uint, content string) (err error) {
	err = global.GVA_DB.Model(&system.TbProjectScript{}).Where("id = ?", id).Update("content", content).Error
	return
}
