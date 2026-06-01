package system

import "github.com/flipped-aurora/easy-deploy/server/global"

type TbAgileTableSample struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectConfigID uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:int;index:idx_agile_table_sample_scope,priority:1;comment:项目配置ID"`
	ConnectionID    uint   `json:"connectionId" form:"connectionId" gorm:"column:connection_id;type:int;index:idx_agile_table_sample_scope,priority:2;comment:数据源ID"`
	DatabaseName    string `json:"databaseName" form:"databaseName" gorm:"column:database_name;type:varchar(255);index:idx_agile_table_sample_scope,priority:3;comment:数据库名称"`
	DBTableName     string `json:"tableName" form:"tableName" gorm:"column:table_name;type:varchar(255);index:idx_agile_table_sample_scope,priority:4;comment:表名称"`
	UserName        string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(100);index:idx_agile_table_sample_scope,priority:5;comment:用户名"`
	BusinessName    string `json:"businessName" form:"businessName" gorm:"column:business_name;type:varchar(255);comment:业务名称"`
	TableComment    string `json:"tableComment" form:"tableComment" gorm:"column:table_comment;type:text;comment:表注释"`
	SortIndex       int    `json:"sortIndex" form:"sortIndex" gorm:"column:sort_index;type:int;comment:排序"`
}

func (TbAgileTableSample) TableName() string {
	return "tb_agile_table_sample"
}

type TbAgileTableSampleHistory struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectConfigID uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:int;index:idx_agile_table_sample_history_scope,priority:1;comment:项目配置ID"`
	ConnectionID    uint   `json:"connectionId" form:"connectionId" gorm:"column:connection_id;type:int;index:idx_agile_table_sample_history_scope,priority:2;comment:数据源ID"`
	UserName        string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(100);index:idx_agile_table_sample_history_scope,priority:3;comment:用户名"`
	BusinessName    string `json:"historyName" form:"historyName" gorm:"column:business_name;type:varchar(255);comment:业务名称"`
	TableCount      int    `json:"tableCount" form:"tableCount" gorm:"column:table_count;type:int;comment:表数量"`
	TableSnapshot   string `json:"tableSnapshot" form:"tableSnapshot" gorm:"column:table_snapshot;type:text;comment:表选择快照JSON"`
}

func (TbAgileTableSampleHistory) TableName() string {
	return "tb_agile_table_sample_history"
}
