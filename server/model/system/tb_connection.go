package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbConnection struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ConnectionName    string `json:"connectionName" form:"connectionName" gorm:"column:connection_name;type:varchar(100);comment:数据源名称"`
	ConnectionType    string `json:"connectionType" form:"connectionType" gorm:"column:connection_type;type:varchar(100);comment:数据源类型"`
	ConnectionUrl     string `json:"connectionUrl" form:"connectionUrl" gorm:"column:connection_url;type:varchar(500);comment:数据源地址"`
	ConnectionGroup   string `json:"connectionGroup" form:"connectionGroup" gorm:"column:connection_group;type:varchar(100);comment:数据源分组"`
	DatabaseName      string `json:"databaseName" form:"databaseName" gorm:"column:database_name;type:varchar(100);comment:数据库名称"`
	Port              int    `json:"port" form:"port" gorm:"column:port;type:int;comment:数据库端口号"`
	DbLoginName       string `json:"dbLoginName" form:"dbLoginName" gorm:"column:db_login_name;type:varchar(100);comment:用户名称"`
	DbLoginPassword   string `json:"dbLoginPassword" form:"dbLoginPassword" gorm:"column:db_login_password;type:varchar(255);comment:用户密码"`
	UserName          string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(100);comment:创建者用户名"`
	EnvName           string `json:"envName" form:"envName" gorm:"column:env_name;type:varchar(100);comment:环境名称"`
}

func (TbConnection) TableName() string {
	return "tb_connection"
}
