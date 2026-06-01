package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbEntityService struct{}

func (s *TbEntityService) CreateTbEntity(entity system.TbEntity) (err error) {
	err = global.GVA_DB.Create(&entity).Error
	return err
}

func (s *TbEntityService) DeleteTbEntity(entity system.TbEntity) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&entity).Error
	return err
}

func (s *TbEntityService) DeleteTbEntityByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbEntity{}, "id in ?", ids).Error
	return err
}

func (s *TbEntityService) UpdateTbEntity(entity *system.TbEntity) (err error) {
	err = global.GVA_DB.Updates(entity).Error
	return err
}

func (s *TbEntityService) GetTbEntity(id uint) (entity system.TbEntity, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&entity).Error
	return
}

func (s *TbEntityService) GetTbEntityInfoList(info systemReq.EntitySearch) (list []system.TbEntity, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbEntity{})
	if info.EntityName != "" {
		db = db.Where("entity_name LIKE ?", "%"+info.EntityName+"%")
	}
	if info.ServerName != "" {
		db = db.Where("server_name = ?", info.ServerName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}
