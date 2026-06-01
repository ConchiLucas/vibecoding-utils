package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

// TbServer 服务器表
type TbServer struct {
	global.GVA_MODEL
	ServerName         string `json:"serverName" form:"serverName" gorm:"column:server_name;comment:服务器名称"`
	ServerIp           string `json:"serverIp" form:"serverIp" gorm:"column:server_ip;comment:服务器ip"`
	ServerInternalIp   string `json:"serverInternalIp" form:"serverInternalIp" gorm:"column:server_internal_ip;comment:服务器内网ip"`
	ServerLoginName    string `json:"serverLoginName" form:"serverLoginName" gorm:"column:server_login_name;comment:服务器登录名称"`
	ServerLoginPassword string `json:"serverLoginPassword" form:"serverLoginPassword" gorm:"column:server_login_password;comment:服务器登录密码"`
	ServerLoginPort    int    `json:"serverLoginPort" form:"serverLoginPort" gorm:"column:server_login_port;comment:服务器登录端口"`
	UserId           uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:项目归属用户ID"`
	ExtendParams       string `json:"extendParams" form:"extendParams" gorm:"column:extend_params;type:text;comment:扩展字段 json结构"`
	Remark             string `json:"remark" form:"remark" gorm:"column:remark;comment:备注"`
}

func (TbServer) TableName() string {
	return "tb_server"
}
