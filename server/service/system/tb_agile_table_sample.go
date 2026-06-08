package system

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	systemResp "github.com/flipped-aurora/easy-deploy/server/model/system/response"
	"gorm.io/gorm"
)

type TbAgileTableSampleService struct{}

func (s *TbAgileTableSampleService) List(scope systemReq.AgileTableSampleScope, userName string) ([]system.TbAgileTableSample, error) {
	if err := ensureAgileTableSampleTable(); err != nil {
		return nil, err
	}
	if scope.ConnectionID == 0 {
		return []system.TbAgileTableSample{}, nil
	}
	var list []system.TbAgileTableSample
	err := global.GVA_DB.
		Where("connection_id = ? AND user_name = ?", scope.ConnectionID, userName).
		Order("sort_index asc, id asc").
		Find(&list).Error
	return list, err
}

func (s *TbAgileTableSampleService) Save(req systemReq.AgileTableSampleSave, userName string) ([]system.TbAgileTableSample, error) {
	if err := ensureAgileTableSampleTable(); err != nil {
		return nil, err
	}
	if req.ConnectionID == 0 {
		return nil, fmt.Errorf("缺少数据源")
	}

	returned := make([]system.TbAgileTableSample, 0, len(req.Tables))
	seen := map[string]struct{}{}
	businessName := strings.TrimSpace(req.HistoryName)
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("connection_id = ? AND user_name = ?", req.ConnectionID, userName).
			Delete(&system.TbAgileTableSample{}).Error; err != nil {
			return err
		}

		for _, table := range req.Tables {
			databaseName := strings.TrimSpace(table.DatabaseName)
			tableName := strings.TrimSpace(table.TableName)
			if databaseName == "" || tableName == "" {
				continue
			}
			key := databaseName + "\x00" + tableName
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			record := system.TbAgileTableSample{
				ProjectConfigID: req.ProjectConfigID,
				ConnectionID:    req.ConnectionID,
				DatabaseName:    databaseName,
				DBTableName:     tableName,
				TableComment:    strings.TrimSpace(table.TableComment),
				UserName:        userName,
				BusinessName:    businessName,
				SortIndex:       len(returned),
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			returned = append(returned, record)
		}
		if businessName != "" {
			historyTables := make([]systemReq.AgileTableSampleItem, 0, len(returned))
			for _, record := range returned {
				historyTables = append(historyTables, systemReq.AgileTableSampleItem{
					DatabaseName: record.DatabaseName,
					TableName:    record.DBTableName,
					TableComment: record.TableComment,
				})
			}
			snapshot, err := json.Marshal(historyTables)
			if err != nil {
				return err
			}
			history := system.TbAgileTableSampleHistory{
				ProjectConfigID: req.ProjectConfigID,
				ConnectionID:    req.ConnectionID,
				UserName:        userName,
				BusinessName:    businessName,
			}
			if err := tx.
				Where("connection_id = ? AND user_name = ? AND business_name = ?", req.ConnectionID, userName, businessName).
				Assign(system.TbAgileTableSampleHistory{
					ProjectConfigID: req.ProjectConfigID,
					TableCount:      len(historyTables),
					TableSnapshot:   string(snapshot),
				}).
				FirstOrCreate(&history).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return returned, nil
}

func (s *TbAgileTableSampleService) History(scope systemReq.AgileTableSampleScope, userName string) ([]systemResp.AgileTableSampleHistory, error) {
	if err := ensureAgileTableSampleTable(); err != nil {
		return nil, err
	}
	if scope.ConnectionID == 0 {
		return []systemResp.AgileTableSampleHistory{}, nil
	}

	var records []system.TbAgileTableSampleHistory
	if err := global.GVA_DB.
		Where("connection_id = ? AND user_name = ? AND business_name <> ?", scope.ConnectionID, userName, "").
		Order("id desc").
		Find(&records).Error; err != nil {
		return nil, err
	}

	histories := make([]systemResp.AgileTableSampleHistory, 0, len(records))
	seenNames := map[string]struct{}{}
	for _, record := range records {
		name := strings.TrimSpace(record.BusinessName)
		if name == "" {
			continue
		}
		if _, ok := seenNames[name]; ok {
			continue
		}
		seenNames[name] = struct{}{}
		var tables []systemResp.AgileTableSampleItem
		if strings.TrimSpace(record.TableSnapshot) != "" {
			_ = json.Unmarshal([]byte(record.TableSnapshot), &tables)
		}
		histories = append(histories, systemResp.AgileTableSampleHistory{
			ID:              record.ID,
			ProjectConfigID: record.ProjectConfigID,
			ConnectionID:    record.ConnectionID,
			UserName:        record.UserName,
			HistoryName:     record.BusinessName,
			TableCount:      record.TableCount,
			Tables:          tables,
			CreatedAt:       record.CreatedAt,
			UpdatedAt:       record.UpdatedAt,
		})
	}
	return histories, nil
}

func ensureAgileTableSampleTable() error {
	if global.GVA_DB == nil {
		return fmt.Errorf("database is not initialized")
	}
	return global.GVA_DB.AutoMigrate(&system.TbAgileTableSample{}, &system.TbAgileTableSampleHistory{})
}
