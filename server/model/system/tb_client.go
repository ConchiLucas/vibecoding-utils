package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbClient struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	LoginName             string `json:"loginName" gorm:"column:login_name;type:varchar(50);comment:登录用户名"`
	Password              string `json:"password" gorm:"column:password;type:varchar(100);comment:用户密码"`
	NickName              string `json:"nickName" gorm:"column:nick_name;type:varchar(255);comment:用户名称"`
	UserExtendParams      string `json:"userExtendParams" gorm:"column:user_extend_params;type:varchar(255);comment:用户扩展参数 json结构"`
	Environment           string `json:"environment" gorm:"column:environment;type:varchar(255);comment:环境 字典"`
	Identity              string `json:"identity" gorm:"column:identity;type:varchar(255);comment:用户身份 字典"`
	RequestDemo           string `json:"requestDemo" gorm:"column:request_demo;type:varchar(255);comment:请求代码"`
	RequestExtendParams   string `json:"requestExtendParams" gorm:"column:request_extend_params;type:varchar(255);comment:请求扩展参数 json结构"`
	EnableFlag            int    `json:"enableFlag" gorm:"column:enable_flag;type:int;comment:状态  0:停用 1:启用"`
	InterfaceRequestHeader string `json:"interfaceRequestHeader" gorm:"column:interface_request_header;type:varchar(255);comment:接口请求头"`
	Remark                string `json:"remark" gorm:"column:remark;type:varchar(256);comment:备注"`
	UserName              string `json:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
}

func (TbClient) TableName() string {
	return "tb_client"
}
