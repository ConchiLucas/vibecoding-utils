package response

import "time"

type AgileTableSampleHistory struct {
	ID              uint                   `json:"ID"`
	ProjectConfigID uint                   `json:"projectConfigId"`
	ConnectionID    uint                   `json:"connectionId"`
	UserName        string                 `json:"userName"`
	HistoryName     string                 `json:"historyName"`
	TableCount      int                    `json:"tableCount"`
	Tables          []AgileTableSampleItem `json:"tables"`
	CreatedAt       time.Time              `json:"CreatedAt"`
	UpdatedAt       time.Time              `json:"UpdatedAt"`
}

type AgileTableSampleItem struct {
	DatabaseName string `json:"databaseName"`
	TableName    string `json:"tableName"`
	TableComment string `json:"tableComment"`
}
