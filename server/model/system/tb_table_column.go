package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbTableColumn struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ConnectionId int    `json:"connectionId" gorm:"column:connection_id;type:int;comment:数据源id"`
	TableId      string `json:"tableId" gorm:"column:table_id;type:varchar(255);comment:sm_table主键id"`
	ColumnName   string `json:"columnName" gorm:"column:column_name;type:varchar(255);comment:字段名称"`
	ColumnType   string `json:"columnType" gorm:"column:column_type;type:varchar(255);comment:字段类型"`
	ColumnSize   string `json:"columnSize" gorm:"column:column_size;type:varchar(255);comment:字段长度"`
	IsEmpty      int    `json:"isEmpty" gorm:"column:is_empty;type:int;comment:是否为空(0:否,1:是)"`
	DefaultValue string `json:"defaultValue" gorm:"column:default_value;type:varchar(255);comment:默认字段"`
	Description  string `json:"description" gorm:"column:description;type:varchar(255);comment:字段注释"`
}

func (TbTableColumn) TableName() string {
	return "tb_table_column"
}
