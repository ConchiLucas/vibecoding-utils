package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbTable struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	DatabaseName string `json:"databaseName" gorm:"column:database_name;type:varchar(255);comment:数据库名"`
	TbName       string `json:"tableName" gorm:"column:table_name;type:varchar(255);comment:表名"`
	Description  string `json:"description" gorm:"column:description;type:varchar(255);comment:表注释"`
	ConnectionId int    `json:"connectionId" gorm:"column:connection_id;type:int;comment:数据源id"`
	UserName     string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
}

func (TbTable) TableName() string {
	return "tb_table"
}
