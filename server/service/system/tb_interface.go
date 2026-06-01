package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbInterfaceService struct{}

func (s *TbInterfaceService) CreateTbInterface(iface system.TbInterface) (err error) {
	err = global.GVA_DB.Create(&iface).Error
	return err
}

func (s *TbInterfaceService) DeleteTbInterface(iface system.TbInterface) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&iface).Error
	return err
}

func (s *TbInterfaceService) DeleteTbInterfaceByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbInterface{}, "id in ?", ids).Error
	return err
}

func (s *TbInterfaceService) UpdateTbInterface(iface *system.TbInterface) (err error) {
	err = global.GVA_DB.Updates(iface).Error
	return err
}

func (s *TbInterfaceService) GetTbInterface(id uint) (iface system.TbInterface, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&iface).Error
	return
}

func (s *TbInterfaceService) GetTbInterfaceInfoList(info systemReq.InterfaceSearch) (list []system.TbInterface, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbInterface{})
	if info.InterfaceName != "" {
		db = db.Where("interface_name LIKE ?", "%"+info.InterfaceName+"%")
	}
	if info.ServerName != "" {
		db = db.Where("server_name = ?", info.ServerName)
	}
	if info.ProjectName != "" {
		db = db.Where("project_name = ?", info.ProjectName)
	}
	if info.Paths != "" {
		db = db.Where("paths LIKE ?", "%"+info.Paths+"%")
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	// 排序规则：已点击接口测试的排在前面，按最近测试时间倒序；未测试的排在后面
	err = db.Order("CASE WHEN last_tested_at IS NOT NULL THEN 0 ELSE 1 END ASC, last_tested_at DESC, id DESC").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}
