package system

import (
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbTablePreferService struct{}

func (s *TbTablePreferService) CreateTbTablePrefer(prefer system.TbTablePrefer) (err error) {
	err = global.GVA_DB.Create(&prefer).Error
	return err
}

func (s *TbTablePreferService) DeleteTbTablePrefer(prefer system.TbTablePrefer) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&prefer).Error
	return err
}

func (s *TbTablePreferService) DeleteTbTablePreferByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbTablePrefer{}, "id in ?", ids).Error
	return err
}

func (s *TbTablePreferService) UpdateTbTablePrefer(prefer *system.TbTablePrefer) (err error) {
	err = global.GVA_DB.Updates(prefer).Error
	return err
}

func (s *TbTablePreferService) GetTbTablePrefer(id uint) (prefer system.TbTablePrefer, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&prefer).Error
	return
}

func (s *TbTablePreferService) GetTbTablePreferInfoList(info systemReq.TablePreferSearch) (list []system.TbTablePrefer, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbTablePrefer{})
	if info.TbName != "" {
		db = db.Where("table_name LIKE ?", "%"+info.TbName+"%")
	}
	if info.UserName != "" {
		db = db.Where("user_name = ?", info.UserName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *TbTablePreferService) GetPreferVOByParams(databaseStr string, userName string, projectConfigID uint, connectionID uint) (system.TbTablePrefer, error) {
	var prefer system.TbTablePrefer
	parts := strings.Split(databaseStr, ":")
	if len(parts) >= 2 {
		prefer.DatabaseName = parts[0]
		prefer.TbName = parts[1]
	}
	db := global.GVA_DB.Where("database_name = ? AND table_name = ? AND user_name = ?", prefer.DatabaseName, prefer.TbName, userName)
	if projectConfigID != 0 {
		db = db.Where("project_config_id = ?", projectConfigID)
	}
	if connectionID != 0 {
		db = db.Where("connection_id = ?", connectionID)
	}
	err := db.Order("id desc").First(&prefer).Error
	return prefer, err
}

func (s *TbTablePreferService) GetPreferColumnValueList(databaseStr string, userName string, projectConfigID uint, connectionID uint) ([]map[string]interface{}, error) {
	parts := strings.Split(databaseStr, ":")
	if len(parts) < 2 {
		return []map[string]interface{}{}, nil
	}
	var prefers []system.TbTablePrefer
	db := global.GVA_DB.Where("database_name = ? AND table_name = ? AND user_name = ?", parts[0], parts[1], userName)
	if projectConfigID != 0 {
		db = db.Where("project_config_id = ?", projectConfigID)
	}
	if connectionID != 0 {
		db = db.Where("connection_id = ?", connectionID)
	}
	err := db.Find(&prefers).Error
	if err != nil {
		return nil, err
	}

	uniqueValues := make(map[string]struct{})
	var result []map[string]interface{}
	for _, p := range prefers {
		if _, ok := uniqueValues[p.ColumnValue]; !ok {
			uniqueValues[p.ColumnValue] = struct{}{}
			result = append(result, map[string]interface{}{"value": p.ColumnValue})
		}
	}
	return result, nil
}

// GetHistoryTableNames returns a deduplicated list of "db:table" strings previously queried by the user
func (s *TbTablePreferService) GetHistoryTableNames(userName string, projectConfigID uint, connectionID uint) ([]string, error) {
	type row struct {
		DatabaseName string
		TableName    string
		MaxID        int64
	}
	var rows []row
	// PostgreSQL: must include ORDER BY column in SELECT when using GROUP BY
	db := global.GVA_DB.Model(&system.TbTablePrefer{}).
		Select("database_name, table_name, MAX(id) as max_id").
		Where("user_name = ?", userName)
	if projectConfigID != 0 {
		db = db.Where("project_config_id = ?", projectConfigID)
	}
	if connectionID != 0 {
		db = db.Where("connection_id = ?", connectionID)
	}
	err := db.Group("database_name, table_name").
		Order("max_id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	var result []string
	for _, r := range rows {
		if r.DatabaseName != "" && r.TableName != "" {
			result = append(result, r.DatabaseName+":"+r.TableName)
		} else if r.TableName != "" {
			result = append(result, r.TableName)
		}
	}
	return result, nil
}
