package system

import (
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbTableService struct{}

func (s *TbTableService) CreateTbTable(table system.TbTable) (err error) {
	err = global.GVA_DB.Create(&table).Error
	return err
}

func (s *TbTableService) DeleteTbTable(table system.TbTable) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&table).Error
	return err
}

func (s *TbTableService) DeleteTbTableByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbTable{}, "id in ?", ids).Error
	return err
}

func (s *TbTableService) UpdateTbTable(table *system.TbTable) (err error) {
	err = global.GVA_DB.Updates(table).Error
	return err
}

func (s *TbTableService) GetTbTable(id uint) (table system.TbTable, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&table).Error
	return
}

func (s *TbTableService) GetTbTableInfoList(info systemReq.TableSearch) (list []system.TbTable, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbTable{})
	if info.TbName != "" {
		db = db.Where("table_name LIKE ?", "%"+info.TbName+"%")
	}
	if info.DatabaseName != "" {
		db = db.Where("database_name = ?", info.DatabaseName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *TbTableService) TableFuzzyQuery(table system.TbTable, userName string) (resultList []map[string]interface{}, err error) {
	if table.TbName == "" {
		// fetch latest history from prefer
		type PreferResult struct {
			DatabaseName string
			TableName    string
		}
		var prefers []PreferResult
		global.GVA_DB.Model(&system.TbTablePrefer{}).
			Select("database_name, table_name, max(create_time)").
			Where("user_name = ?", userName).
			Group("database_name, table_name").
			Order("max(create_time) desc").
			Limit(30).
			Scan(&prefers)

		for _, p := range prefers {
			resultList = append(resultList, map[string]interface{}{
				"value": p.DatabaseName + ":" + p.TableName,
			})
		}
		return resultList, nil
	}

	parts := strings.Split(table.TbName, ":")
	if len(parts) == 1 {
		table.TbName = strings.TrimSpace(parts[0])
	} else if len(parts) == 2 {
		table.DatabaseName = strings.TrimSpace(parts[0])
		table.TbName = strings.TrimSpace(parts[1])
	}
	table.UserName = userName

	db := global.GVA_DB.Model(&system.TbTable{})
	if table.TbName != "" {
		db = db.Where("table_name LIKE ?", "%"+table.TbName+"%")
	}
	if table.DatabaseName != "" {
		db = db.Where("database_name = ?", table.DatabaseName)
	}
	db = db.Where("user_name = ?", table.UserName)

	var tableEntities []system.TbTable
	err = db.Limit(30).Find(&tableEntities).Error
	if err != nil {
		return nil, err
	}

	var prioritizedList []map[string]interface{}
	var normalList []map[string]interface{}

	for _, ent := range tableEntities {
		val := ent.DatabaseName + ":" + ent.TbName
		m := map[string]interface{}{"value": val}
		if ent.TbName == table.TbName {
			prioritizedList = append(prioritizedList, m)
		} else {
			normalList = append(normalList, m)
		}
	}
	resultList = append(resultList, prioritizedList...)
	resultList = append(resultList, normalList...)

	return resultList, nil
}
