package system

import (
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbInterface struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	InterfaceName string     `json:"interfaceName" form:"interfaceName" gorm:"column:interface_name;type:varchar(255);comment:接口名称"`
	Paths         string     `json:"paths" form:"paths" gorm:"column:paths;type:varchar(255);comment:接口路径"`
	Description   string     `json:"description" form:"description" gorm:"column:description;type:varchar(255);comment:接口描述"`
	Method        string     `json:"method" form:"method" gorm:"column:method;type:varchar(255);comment:接口方法"`
	RequestParam  string     `json:"requestParam" form:"requestParam" gorm:"column:request_param;type:text;comment:请求入参"`
	ResponseParam string     `json:"responseParam" form:"responseParam" gorm:"column:response_param;type:text;comment:返回出参"`
	UserName      string     `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	ServerName    string     `json:"serverName" form:"serverName" gorm:"column:server_name;type:varchar(255);comment:服务名称"`
	ProjectName   string     `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);comment:项目名称"`
	RequestType   string     `json:"requestType" form:"requestType" gorm:"column:request_type;type:varchar(255);comment:请求方式"`
	LastTestedAt  *time.Time `json:"lastTestedAt" form:"lastTestedAt" gorm:"column:last_tested_at;comment:最近一次接口测试时间"`
}

func (TbInterface) TableName() string {
	return "tb_interface"
}
