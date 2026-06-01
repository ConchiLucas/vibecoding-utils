package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbInterfaceLogService struct{}

func (s *TbInterfaceLogService) CreateTbInterfaceLog(log system.TbInterfaceLog) (err error) {
	err = global.GVA_DB.Create(&log).Error
	return err
}

func (s *TbInterfaceLogService) DeleteTbInterfaceLog(log system.TbInterfaceLog) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&log).Error
	return err
}

func (s *TbInterfaceLogService) DeleteTbInterfaceLogByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbInterfaceLog{}, "id in ?", ids).Error
	return err
}

func (s *TbInterfaceLogService) UpdateTbInterfaceLog(log *system.TbInterfaceLog) (err error) {
	err = global.GVA_DB.Updates(log).Error
	return err
}

func (s *TbInterfaceLogService) GetTbInterfaceLog(id uint) (log system.TbInterfaceLog, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&log).Error
	return
}

func (s *TbInterfaceLogService) GetTbInterfaceLogInfoList(info systemReq.InterfaceLogSearch) (list []system.TbInterfaceLog, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbInterfaceLog{})
	if info.InterfacePaths != "" {
		db = db.Where("interface_paths = ?", info.InterfacePaths)
	}
	// Note: 0 is the default int value, so if info.IsSuccess is 0 we might not know if it's set or default.
	// For now we don't filter on it unless needed.
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
