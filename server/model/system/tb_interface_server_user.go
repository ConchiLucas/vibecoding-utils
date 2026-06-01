package system

import "github.com/flipped-aurora/easy-deploy/server/global"

type TbInterfaceServerUser struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectName   string `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);comment:所属项目"`
	LoginAccount  string `json:"loginAccount" form:"loginAccount" gorm:"column:login_account;type:varchar(255);not null;comment:登陆账号"`
	LoginPassword string `json:"loginPassword" form:"loginPassword" gorm:"column:login_password;type:varchar(255);not null;comment:登陆密码"`
	UserNickname  string `json:"userNickname" form:"userNickname" gorm:"column:user_nickname;type:varchar(255);comment:用户昵称"`
	RoleCode      string `json:"roleCode" form:"roleCode" gorm:"column:role_code;type:varchar(255);comment:角色编码"`
	RoleName      string `json:"roleName" form:"roleName" gorm:"column:role_name;type:varchar(255);comment:角色名称"`
	Environment   string `json:"environment" form:"environment" gorm:"column:environment;type:varchar(255);comment:所属环境"`
	RequestHeader string `json:"requestHeader" form:"requestHeader" gorm:"column:request_header;type:text;comment:请求Header"`
	EnableFlag    int    `json:"enableFlag" form:"enableFlag" gorm:"column:enable_flag;type:int;default:1;comment:启用标志(1启用 0禁用)"`
}

func (TbInterfaceServerUser) TableName() string {
	return "tb_interface_server_user"
}
