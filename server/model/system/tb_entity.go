package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbEntity struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	EntityName     string `json:"entityName" gorm:"column:entity_name;type:varchar(255);comment:实体类名称"`
	RequiredColumn string `json:"requiredColumn" gorm:"column:required_column;type:text;comment:必填字段"`
	ColumnCount    int    `json:"columnCount" gorm:"column:column_count;type:int;comment:字段数"`
	ContainEntity  int    `json:"containEntity" gorm:"column:contain_entity;type:int;comment:字段里是否还有其他实体类 0-否 1-是"`
	UserName       string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	ServerName     string `json:"serverName" gorm:"column:server_name;type:varchar(255);comment:服务名称"`
}

func (TbEntity) TableName() string {
	return "tb_entity"
}
