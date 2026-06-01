package system

import (
	"errors"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type ProjectGroupService struct{}

var ProjectGroupServiceApp = new(ProjectGroupService)

// GetGroupList 获取所有项目组
func (s *ProjectGroupService) GetGroupList(userId uint) (list []system.TbProjectGroup, err error) {
	db := global.GVA_DB.Model(&system.TbProjectGroup{})
	if userId != 0 {
		db = db.Where("user_id = ?", userId)
	}
	err = db.Order("id asc").Find(&list).Error
	return
}

// SaveOrUpdateGroup 新增或修改项目组
func (s *ProjectGroupService) SaveOrUpdateGroup(group system.TbProjectGroup) (result system.TbProjectGroup, err error) {
	if group.ID != 0 {
		err = global.GVA_DB.Model(&system.TbProjectGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{
			"group_name":  group.GroupName,
			"description": group.Description,
		}).Error
		result = group
	} else {
		err = global.GVA_DB.Create(&group).Error
		result = group
	}
	return
}

// DeleteGroup 删除项目组（需确认下面没有项目）
func (s *ProjectGroupService) DeleteGroup(id int) error {
	var count int64
	global.GVA_DB.Model(&system.TbProject{}).Where("group_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("该项目组下还有项目，请先迁移或删除相关项目")
	}
	return global.GVA_DB.Where("id = ?", id).Unscoped().Delete(&system.TbProjectGroup{}).Error
}

// RenameGroup 仅更新组名
func (s *ProjectGroupService) RenameGroup(id int, name string) error {
	return global.GVA_DB.Model(&system.TbProjectGroup{}).Where("id = ?", id).
		Update("group_name", strings.TrimSpace(name)).Error
}
