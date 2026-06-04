package request

import (
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type ServerSearch struct {
	system.TbInterfaceServer
	request.PageInfo
}

type InterfaceEnvSearch struct {
	system.TbInterfaceEnv
	request.PageInfo
}

type InterfaceSearch struct {
	system.TbInterface
	request.PageInfo
}

type ConnectionSearch struct {
	system.TbConnection
	request.PageInfo
}

type TableRecordUpdateChange struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UpdateRemoteTableRecordReq struct {
	ID           uint                      `json:"ID"`
	DatabaseName string                    `json:"databaseName"`
	TableName    string                    `json:"tableName"`
	Offset       int                       `json:"offset"`
	FilterColumn string                    `json:"filterColumn"`
	FilterValue  string                    `json:"filterValue"`
	Changes      []TableRecordUpdateChange `json:"changes"`
}

type DeleteRemoteTableRecordReq struct {
	ID           uint   `json:"ID"`
	DatabaseName string `json:"databaseName"`
	TableName    string `json:"tableName"`
	Offset       int    `json:"offset"`
	FilterColumn string `json:"filterColumn"`
	FilterValue  string `json:"filterValue"`
}

type GenerateRemoteTableDataReq struct {
	ID           uint   `json:"ID"`
	DatabaseName string `json:"databaseName"`
	TableName    string `json:"tableName"`
	Count        int    `json:"count"`
}

type RemoteSQLQueryReq struct {
	ID           uint   `json:"ID"`
	DatabaseName string `json:"databaseName"`
	SQL          string `json:"sql"`
	Limit        int    `json:"limit"`
}

type RemoteSQLHistoryReq struct {
	ID              uint   `json:"id" form:"id"`
	ProjectConfigID uint   `json:"projectConfigId" form:"projectConfigId"`
	EnvName         string `json:"envName" form:"envName"`
	ConnectionID    uint   `json:"connectionId" form:"connectionId"`
	DatabaseName    string `json:"databaseName" form:"databaseName"`
	SQL             string `json:"sql" form:"sql"`
	Limit           int    `json:"limit" form:"limit"`
}

type TableSearch struct {
	system.TbTable
	request.PageInfo
}

type TableColumnSearch struct {
	system.TbTableColumn
	request.PageInfo
}

type TableRelateSearch struct {
	system.TbTableRelate
	request.PageInfo
}

type ImportTableRelationEndpoint struct {
	DatabaseName string `json:"databaseName"`
	TableName    string `json:"tableName"`
	ColumnName   string `json:"columnName"`
	ColumnType   string `json:"columnType"`
}

type ImportTableRelation struct {
	Source ImportTableRelationEndpoint `json:"source"`
	Target ImportTableRelationEndpoint `json:"target"`
}

type ImportTableRelationsRequest struct {
	ProjectConfigID uint                  `json:"projectConfigId"`
	Relations       []ImportTableRelation `json:"relations"`
	UserName        string                `json:"userName"`
}

type EntitySearch struct {
	system.TbEntity
	request.PageInfo
}

type ColumnSearch struct {
	system.TbColumn
	request.PageInfo
}

type ClientSearch struct {
	system.TbClient
	request.PageInfo
}

type InterfaceParamsSearch struct {
	system.TbInterfaceParams
	request.PageInfo
}

type InterfaceLogSearch struct {
	system.TbInterfaceLog
	request.PageInfo
}

type InterfaceTreeQO struct {
	ID       uint   `json:"id" form:"id"`
	Type     int    `json:"type" form:"type"`
	UserName string `json:"userName" form:"userName"`
}

type TablePreferSearch struct {
	system.TbTablePrefer
	request.PageInfo
}

type ServerUserSearch struct {
	system.TbInterfaceServerUser
	request.PageInfo
}

type ClientQueryModel struct {
	DatabaseStr     string `json:"databaseStr"`
	Value           string `json:"value"`
	ConnectionID    uint   `json:"connectionId"`
	ConnectionGroup string `json:"connectionGroup"`
	ProjectConfigID uint   `json:"projectConfigId"`
	UserName        string `json:"userName"`
	Environment     string `json:"environment"`
}
