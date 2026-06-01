package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	systemRes "github.com/flipped-aurora/easy-deploy/server/model/system/response"
)

type TbColumnService struct{}

func (s *TbColumnService) CreateTbColumn(col system.TbColumn) (err error) {
	err = global.GVA_DB.Create(&col).Error
	return err
}

func (s *TbColumnService) DeleteTbColumn(col system.TbColumn) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&col).Error
	return err
}

func (s *TbColumnService) DeleteTbColumnByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbColumn{}, "id in ?", ids).Error
	return err
}

func (s *TbColumnService) UpdateTbColumn(col *system.TbColumn) (err error) {
	err = global.GVA_DB.Updates(col).Error
	return err
}

func (s *TbColumnService) GetTbColumn(id uint) (col system.TbColumn, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&col).Error
	return
}

func (s *TbColumnService) GetTbColumnInfoList(info systemReq.ColumnSearch) (list []system.TbColumn, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbColumn{})
	if info.EntityName != "" {
		db = db.Where("entity_name = ?", info.EntityName)
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

func (s *TbColumnService) GetColumnTree(info systemReq.InterfaceTreeQO) (treeList []*systemRes.ColumnTreeVO, err error) {
	var iface system.TbInterface
	err = global.GVA_DB.Where("id = ?", info.ID).First(&iface).Error
	if err != nil {
		return nil, err
	}

	serverName := iface.ServerName
	var entityName string
	if info.Type == 1 && iface.RequestParam != "" {
		entityName = iface.RequestParam
	} else if info.Type == 2 && iface.ResponseParam != "" {
		entityName = iface.ResponseParam
	}

	if entityName == "" {
		return []*systemRes.ColumnTreeVO{}, nil
	}

	var rootColumns []system.TbColumn
	err = global.GVA_DB.Where("server_name = ? AND entity_name = ?", serverName, entityName).Find(&rootColumns).Error
	if err != nil {
		return nil, err
	}

	treeList = s.entityListToTreeList(rootColumns)
	err = s.processColumnList(treeList, serverName)
	return treeList, err
}

func (s *TbColumnService) processColumnList(treeList []*systemRes.ColumnTreeVO, serverName string) error {
	for _, node := range treeList {
		if node.ColumnRef != "" && node.ColumnRef != node.EntityName {
			var childColumns []system.TbColumn
			err := global.GVA_DB.Where("server_name = ? AND entity_name = ?", serverName, node.ColumnRef).Find(&childColumns).Error
			if err != nil {
				return err
			}

			if len(childColumns) > 0 {
				childTreeList := s.entityListToTreeList(childColumns)
				for _, child := range childTreeList {
					child.Pid = node.ID
				}
				err = s.processColumnList(childTreeList, serverName)
				if err != nil {
					return err
				}
				node.Children = childTreeList
			}
		}
	}
	return nil
}

func (s *TbColumnService) entityListToTreeList(entities []system.TbColumn) []*systemRes.ColumnTreeVO {
	list := make([]*systemRes.ColumnTreeVO, 0, len(entities))
	for _, col := range entities {
		list = append(list, &systemRes.ColumnTreeVO{
			ID:           col.ID,
			Pid:          0,
			EntityName:   col.EntityName,
			ColumnName:   col.ColumnName,
			ColumnType:   col.ColumnType,
			Description:  col.Description,
			DefaultValue: col.DefaultValue,
			ColumnRef:    col.ColumnRef,
			UserName:     col.UserName,
			ServerName:   col.ServerName,
			Children:     nil,
		})
	}
	return list
}
