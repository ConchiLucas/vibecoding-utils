package system

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gorm.io/gorm"
)

const (
	defaultRemoteSQLHistoryLimit = 50
	maxRemoteSQLHistoryLimit     = 200
)

type RemoteSQLHistoryVO struct {
	ID              uint   `json:"id"`
	ProjectConfigID uint   `json:"projectConfigId"`
	EnvName         string `json:"envName"`
	ConnectionID    uint   `json:"connectionId"`
	ConnectionName  string `json:"connectionName"`
	ConnectionType  string `json:"connectionType"`
	DatabaseName    string `json:"databaseName"`
	SQL             string `json:"sql"`
	CreatedAt       string `json:"createdAt"`
}

type remoteSQLHistoryScope struct {
	ProjectConfigID uint
	EnvName         string
	ConnectionID    uint
	ConnectionName  string
	ConnectionType  string
	DatabaseName    string
	UserName        string
}

func (s *TbConnectionService) ListRemoteSQLHistory(req systemReq.RemoteSQLHistoryReq, userName string) ([]RemoteSQLHistoryVO, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultRemoteSQLHistoryLimit
	}
	if limit > maxRemoteSQLHistoryLimit {
		limit = maxRemoteSQLHistoryLimit
	}

	var records []system.TbSQLQueryHistory
	err := remoteSQLHistoryListQuery(req, userName).
		Order("id desc").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return toRemoteSQLHistoryVOs(records), nil
}

func (s *TbConnectionService) SaveRemoteSQLHistory(req systemReq.RemoteSQLHistoryReq, userName string) ([]RemoteSQLHistoryVO, error) {
	scope, err := s.resolveRemoteSQLHistoryScope(req, userName)
	if err != nil {
		return nil, err
	}
	sqlText := strings.TrimSpace(req.SQL)
	if sqlText == "" {
		return nil, fmt.Errorf("SQL 语句不能为空")
	}

	sqlHash := hashSQLHistorySQL(sqlText)
	if err := remoteSQLHistoryScopeQuery(scope).
		Where("sql_hash = ?", sqlHash).
		Delete(&system.TbSQLQueryHistory{}).Error; err != nil {
		return nil, err
	}

	record := system.TbSQLQueryHistory{
		ProjectConfigID: scope.ProjectConfigID,
		EnvName:         scope.EnvName,
		ConnectionID:    scope.ConnectionID,
		ConnectionName:  scope.ConnectionName,
		ConnectionType:  scope.ConnectionType,
		DatabaseName:    scope.DatabaseName,
		SQLContent:      sqlText,
		SQLHash:         sqlHash,
		UserName:        scope.UserName,
	}
	if err := global.GVA_DB.Create(&record).Error; err != nil {
		return nil, err
	}
	if err := pruneRemoteSQLHistory(scope, defaultRemoteSQLHistoryLimit); err != nil {
		return nil, err
	}

	req.Limit = defaultRemoteSQLHistoryLimit
	req.ConnectionID = 0
	req.EnvName = ""
	req.DatabaseName = ""
	return s.ListRemoteSQLHistory(req, userName)
}

func (s *TbConnectionService) DeleteRemoteSQLHistory(req systemReq.RemoteSQLHistoryReq, userName string) ([]RemoteSQLHistoryVO, error) {
	if req.ID == 0 {
		return nil, fmt.Errorf("缺少历史记录 ID")
	}
	if err := remoteSQLHistoryListQuery(req, userName).
		Where("id = ?", req.ID).
		Delete(&system.TbSQLQueryHistory{}).Error; err != nil {
		return nil, err
	}
	req.Limit = defaultRemoteSQLHistoryLimit
	req.ID = 0
	req.ConnectionID = 0
	req.EnvName = ""
	req.DatabaseName = ""
	return s.ListRemoteSQLHistory(req, userName)
}

func (s *TbConnectionService) ClearRemoteSQLHistory(req systemReq.RemoteSQLHistoryReq, userName string) error {
	return remoteSQLHistoryListQuery(req, userName).Delete(&system.TbSQLQueryHistory{}).Error
}

func (s *TbConnectionService) resolveRemoteSQLHistoryScope(req systemReq.RemoteSQLHistoryReq, userName string) (remoteSQLHistoryScope, error) {
	if req.ConnectionID == 0 {
		return remoteSQLHistoryScope{}, fmt.Errorf("缺少数据源 ID")
	}
	var conn system.TbConnection
	if err := global.GVA_DB.Where("id = ?", req.ConnectionID).First(&conn).Error; err != nil {
		return remoteSQLHistoryScope{}, fmt.Errorf("连接配置不存在: %w", err)
	}

	envName := strings.TrimSpace(req.EnvName)
	if envName == "" {
		envName = strings.TrimSpace(conn.EnvName)
	}
	databaseName := strings.TrimSpace(req.DatabaseName)
	if databaseName == "" {
		databaseName = strings.TrimSpace(conn.DatabaseName)
	}

	return remoteSQLHistoryScope{
		ProjectConfigID: req.ProjectConfigID,
		EnvName:         envName,
		ConnectionID:    req.ConnectionID,
		ConnectionName:  conn.ConnectionName,
		ConnectionType:  conn.ConnectionType,
		DatabaseName:    databaseName,
		UserName:        userName,
	}, nil
}

func remoteSQLHistoryScopeQuery(scope remoteSQLHistoryScope) *gorm.DB {
	return global.GVA_DB.Model(&system.TbSQLQueryHistory{}).
		Where("project_config_id = ?", scope.ProjectConfigID).
		Where("env_name = ?", scope.EnvName).
		Where("connection_id = ?", scope.ConnectionID).
		Where("database_name = ?", scope.DatabaseName).
		Where("user_name = ?", scope.UserName)
}

func remoteSQLHistoryListQuery(req systemReq.RemoteSQLHistoryReq, userName string) *gorm.DB {
	db := global.GVA_DB.Model(&system.TbSQLQueryHistory{}).
		Where("project_config_id = ?", req.ProjectConfigID).
		Where("user_name = ?", userName)
	if strings.TrimSpace(req.EnvName) != "" {
		db = db.Where("env_name = ?", strings.TrimSpace(req.EnvName))
	}
	if req.ConnectionID != 0 {
		db = db.Where("connection_id = ?", req.ConnectionID)
	}
	if strings.TrimSpace(req.DatabaseName) != "" {
		db = db.Where("database_name = ?", strings.TrimSpace(req.DatabaseName))
	}
	return db
}

func pruneRemoteSQLHistory(scope remoteSQLHistoryScope, keep int) error {
	var ids []uint
	if err := remoteSQLHistoryScopeQuery(scope).Order("id desc").Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) <= keep {
		return nil
	}
	return global.GVA_DB.Delete(&system.TbSQLQueryHistory{}, ids[keep:]).Error
}

func toRemoteSQLHistoryVOs(records []system.TbSQLQueryHistory) []RemoteSQLHistoryVO {
	result := make([]RemoteSQLHistoryVO, 0, len(records))
	for _, record := range records {
		result = append(result, RemoteSQLHistoryVO{
			ID:              record.ID,
			ProjectConfigID: record.ProjectConfigID,
			EnvName:         record.EnvName,
			ConnectionID:    record.ConnectionID,
			ConnectionName:  record.ConnectionName,
			ConnectionType:  record.ConnectionType,
			DatabaseName:    record.DatabaseName,
			SQL:             record.SQLContent,
			CreatedAt:       record.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

func hashSQLHistorySQL(sqlText string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(sqlText), " "))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
