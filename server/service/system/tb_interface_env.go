package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbInterfaceEnvService struct{}

func (s *TbInterfaceEnvService) CreateTbInterfaceEnv(data system.TbInterfaceEnv) error {
	return global.GVA_DB.Create(&data).Error
}

func (s *TbInterfaceEnvService) DeleteTbInterfaceEnv(id uint) error {
	return global.GVA_DB.Unscoped().Delete(&system.TbInterfaceEnv{}, id).Error
}

func (s *TbInterfaceEnvService) UpdateTbInterfaceEnv(data system.TbInterfaceEnv) error {
	return global.GVA_DB.Model(&system.TbInterfaceEnv{}).Where("id = ?", data.ID).Updates(&data).Error
}

func (s *TbInterfaceEnvService) GetTbInterfaceEnvList(info systemReq.InterfaceEnvSearch) (list []system.TbInterfaceEnv, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbInterfaceEnv{})
	if info.ProjectName != "" {
		db = db.Where("project_name = ?", info.ProjectName)
	}
	if info.EnvName != "" {
		db = db.Where("env_name LIKE ?", "%"+info.EnvName+"%")
	}
	if info.UserName != "" {
		db = db.Where("user_name = ?", info.UserName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return list, total, err
}
