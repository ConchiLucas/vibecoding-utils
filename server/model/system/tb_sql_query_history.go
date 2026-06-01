package system

import "github.com/flipped-aurora/easy-deploy/server/global"

type TbSQLQueryHistory struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectConfigID uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:bigint;index:idx_sql_query_history_scope,priority:1;comment:项目配置ID"`
	EnvName         string `json:"envName" form:"envName" gorm:"column:env_name;type:varchar(100);index:idx_sql_query_history_scope,priority:2;comment:环境名称"`
	ConnectionID    uint   `json:"connectionId" form:"connectionId" gorm:"column:connection_id;type:bigint;index:idx_sql_query_history_scope,priority:3;comment:数据源ID"`
	ConnectionName  string `json:"connectionName" form:"connectionName" gorm:"column:connection_name;type:varchar(100);comment:数据源名称"`
	ConnectionType  string `json:"connectionType" form:"connectionType" gorm:"column:connection_type;type:varchar(100);comment:数据源类型"`
	DatabaseName    string `json:"databaseName" form:"databaseName" gorm:"column:database_name;type:varchar(255);index:idx_sql_query_history_scope,priority:4;comment:数据库名称"`
	SQLContent      string `json:"sql" form:"sql" gorm:"column:sql_content;type:text;comment:SQL语句"`
	SQLHash         string `json:"-" form:"-" gorm:"column:sql_hash;type:varchar(64);index;comment:归一化SQL哈希"`
	UserName        string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(100);index:idx_sql_query_history_scope,priority:5;comment:执行用户"`
}

func (TbSQLQueryHistory) TableName() string {
	return "tb_sql_query_history"
}
