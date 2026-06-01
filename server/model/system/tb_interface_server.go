package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbInterfaceServer struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectName  string `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);comment:项目名称"`
	ServerName   string `json:"serverName" form:"serverName" gorm:"column:server_name;type:varchar(255);comment:服务名称"`
	UserName     string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
}

func (TbInterfaceServer) TableName() string {
	return "tb_interface_server"
}
