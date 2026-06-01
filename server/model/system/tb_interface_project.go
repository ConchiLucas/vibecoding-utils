package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbInterfaceProject struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ProjectName string `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);uniqueIndex;comment:项目名称"`
	ProjectDesc string `json:"projectDesc" form:"projectDesc" gorm:"column:project_desc;type:varchar(500);comment:项目介绍"`
	UserName    string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:创建者"`
}

func (TbInterfaceProject) TableName() string {
	return "tb_interface_project"
}
