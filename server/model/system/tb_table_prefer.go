package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbTablePrefer struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectConfigID uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:bigint;index;comment:项目配置ID"`
	DatabaseName    string `json:"databaseName" gorm:"column:database_name;type:varchar(255);comment:数据库名"`
	TbName          string `json:"tableName" gorm:"column:table_name;type:varchar(255);comment:表名"`
	ColumnValue     string `json:"columnValue" gorm:"column:column_value;type:varchar(255);comment:字段值"`
	UserName        string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
}

func (TbTablePrefer) TableName() string {
	return "tb_table_prefer"
}
