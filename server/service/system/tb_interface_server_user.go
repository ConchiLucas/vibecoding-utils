package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbInterfaceServerUserService struct{}

func (s *TbInterfaceServerUserService) CreateTbInterfaceServerUser(data system.TbInterfaceServerUser) error {
	return global.GVA_DB.Create(&data).Error
}

func (s *TbInterfaceServerUserService) DeleteTbInterfaceServerUser(id uint) error {
	return global.GVA_DB.Unscoped().Delete(&system.TbInterfaceServerUser{}, id).Error
}

func (s *TbInterfaceServerUserService) UpdateTbInterfaceServerUser(data system.TbInterfaceServerUser) error {
	return global.GVA_DB.Model(&system.TbInterfaceServerUser{}).Where("id = ?", data.ID).Updates(&data).Error
}

func (s *TbInterfaceServerUserService) GetTbInterfaceServerUserList(info systemReq.ServerUserSearch) (list []system.TbInterfaceServerUser, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbInterfaceServerUser{})
	if info.ProjectName != "" {
		db = db.Where("project_name = ?", info.ProjectName)
	}
	if info.LoginAccount != "" {
		db = db.Where("login_account LIKE ?", "%"+info.LoginAccount+"%")
	}
	if info.Environment != "" {
		db = db.Where("environment = ?", info.Environment)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return list, total, err
}

// UpdateClientStatus toggles the enable_flag for a test user.
func (s *TbInterfaceServerUserService) UpdateClientStatus(id uint, enableFlag int) error {
	return global.GVA_DB.Model(&system.TbInterfaceServerUser{}).
		Where("id = ?", id).
		Update("enable_flag", enableFlag).Error
}
