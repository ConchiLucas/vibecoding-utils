package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbInterfaceParams struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	InterfacePaths string `json:"interfacePaths" gorm:"column:interface_paths;type:varchar(255);comment:接口路径"`
	UserName       string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	Environment    string `json:"environment" gorm:"column:environment;type:varchar(255);comment:环境"`
	Identity       string `json:"identity" gorm:"column:identity;type:varchar(255);comment:用户身份"`
	InterfaceParams string `json:"interfaceParams" gorm:"column:interface_params;type:text;comment:接口请求json参数"`
	ResponseParams string `json:"responseParams" gorm:"column:response_params;type:text;comment:接口返回参数"`
}

func (TbInterfaceParams) TableName() string {
	return "tb_interface_params"
}
