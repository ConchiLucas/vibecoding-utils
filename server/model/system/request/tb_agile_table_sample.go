package request

type AgileTableSampleScope struct {
	ProjectConfigID uint `json:"projectConfigId" form:"projectConfigId"`
	ConnectionID    uint `json:"connectionId" form:"connectionId"`
}

type AgileTableSampleItem struct {
	DatabaseName string `json:"databaseName" form:"databaseName"`
	TableName    string `json:"tableName" form:"tableName"`
	TableComment string `json:"tableComment" form:"tableComment"`
}

type AgileTableSampleSave struct {
	AgileTableSampleScope
	HistoryName string                 `json:"historyName" form:"historyName"`
	Tables      []AgileTableSampleItem `json:"tables"`
}
