package response

type ColumnTreeVO struct {
	ID           uint            `json:"id"`
	Pid          uint            `json:"pid"`
	EntityName   string          `json:"entityName"`
	ColumnName   string          `json:"columnName"`
	ColumnType   string          `json:"columnType"`
	Description  string          `json:"description"`
	DefaultValue string          `json:"defaultValue"`
	ColumnRef    string          `json:"columnRef"`
	UserName     string          `json:"userName"`
	ServerName   string          `json:"serverName"`
	Children     []*ColumnTreeVO `json:"children"`
}
