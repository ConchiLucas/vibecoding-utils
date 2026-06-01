package system

import "github.com/flipped-aurora/easy-deploy/server/global"

type TbInterfaceEnv struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectName string `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);comment:所属项目"`
	EnvName     string `json:"envName" form:"envName" gorm:"column:env_name;type:varchar(255);comment:环境名称"`
	BaseURL     string `json:"baseUrl" form:"baseUrl" gorm:"column:base_url;type:varchar(255);comment:环境基础地址"`
	UserName    string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
}

func (TbInterfaceEnv) TableName() string {
	return "tb_interface_env"
}
