package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbColumn struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	EntityName   string `json:"entityName" gorm:"column:entity_name;type:varchar(255);comment:实体类名称"`
	ColumnName   string `json:"columnName" gorm:"column:column_name;type:varchar(255);comment:字段名称"`
	ColumnType   string `json:"columnType" gorm:"column:column_type;type:varchar(32);comment:字段类型"`
	Description  string `json:"description" gorm:"column:description;type:varchar(1024);comment:字段描述"`
	DefaultValue string `json:"defaultValue" gorm:"column:default_value;type:varchar(32);comment:字段默认值"`
	FormatValue  string `json:"formatValue" gorm:"column:format_value;type:varchar(32);comment:format"`
	MaxLength    int    `json:"maxLength" gorm:"column:max_length;type:int;comment:最大长度"`
	MinLength    int    `json:"minLength" gorm:"column:min_length;type:int;comment:最小长度"`
	Required     int    `json:"required" gorm:"column:required;type:int;comment:是否必填"`
	EnumValue    string `json:"enumValue" gorm:"column:enum_value;type:text;comment:枚举值list"`
	ColumnRef    string `json:"columnRef" gorm:"column:column_ref;type:varchar(255);comment:字段关联值"`
	UserName     string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	ServerName   string `json:"serverName" gorm:"column:server_name;type:varchar(255);comment:服务名称"`
}

func (TbColumn) TableName() string {
	return "tb_column"
}
