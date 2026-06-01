package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbTableColumnService struct{}

func (s *TbTableColumnService) CreateTbTableColumn(tc system.TbTableColumn) (err error) {
	err = global.GVA_DB.Create(&tc).Error
	return err
}

func (s *TbTableColumnService) DeleteTbTableColumn(tc system.TbTableColumn) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&tc).Error
	return err
}

func (s *TbTableColumnService) DeleteTbTableColumnByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbTableColumn{}, "id in ?", ids).Error
	return err
}

func (s *TbTableColumnService) UpdateTbTableColumn(tc *system.TbTableColumn) (err error) {
	err = global.GVA_DB.Updates(tc).Error
	return err
}

func (s *TbTableColumnService) GetTbTableColumn(id uint) (tc system.TbTableColumn, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&tc).Error
	return
}

func (s *TbTableColumnService) GetTbTableColumnInfoList(info systemReq.TableColumnSearch) (list []system.TbTableColumn, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbTableColumn{})
	if info.TableId != "" {
		db = db.Where("table_id = ?", info.TableId)
	}
	if info.ColumnName != "" {
		db = db.Where("column_name LIKE ?", "%"+info.ColumnName+"%")
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}
