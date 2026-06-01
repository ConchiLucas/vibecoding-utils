package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbTableRelate struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectConfigID    uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:bigint;index;comment:项目配置ID"`
	DatabaseName       string `json:"databaseName" gorm:"column:database_name;type:varchar(255);comment:数据库名"`
	TbName             string `json:"tableName" gorm:"column:table_name;type:varchar(255);comment:表名"`
	ColumnName         string `json:"columnName" gorm:"column:column_name;type:varchar(255);comment:字段名称"`
	RelateDatabaseName string `json:"relateDatabaseName" gorm:"column:relate_database_name;type:varchar(255);comment:关联数据库名"`
	RelateTableName    string `json:"relateTableName" gorm:"column:relate_table_name;type:varchar(255);comment:关联表名"`
	RelateColumnName   string `json:"relateColumnName" gorm:"column:relate_column_name;type:varchar(255);comment:关联字段名称"`
	RelateColumnType   string `json:"relateColumnType" gorm:"column:relate_column_type;type:varchar(255);comment:关联字段类型"`
	ColumnType         string `json:"columnType" gorm:"column:column_type;type:varchar(255);comment:字段类型"`
	UserName           string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
}

func (TbTableRelate) TableName() string {
	return "tb_table_relate"
}
