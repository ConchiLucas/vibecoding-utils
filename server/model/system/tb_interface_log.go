package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbInterfaceLog struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	InterfacePaths string `json:"interfacePaths" form:"interfacePaths" gorm:"column:interface_paths;type:varchar(255);comment:接口路径"`
	UserName       string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	IsSuccess      int    `json:"isSuccess" form:"isSuccess" gorm:"column:is_success;type:int;comment:是否成功(1 成功 0失败)"`
	ReqParams      string `json:"reqParams" form:"reqParams" gorm:"column:req_params;type:text;comment:请求参数"`
	ResParams      string `json:"resParams" form:"resParams" gorm:"column:res_params;type:text;comment:返回参数"`
	Environment    string `json:"environment" form:"environment" gorm:"column:environment;type:varchar(255);comment:环境"`
	Identity       string `json:"identity" form:"identity" gorm:"column:identity;type:varchar(255);comment:用户身份"`
}

func (TbInterfaceLog) TableName() string {
	return "tb_interface_log"
}
