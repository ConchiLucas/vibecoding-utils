package system

import "github.com/flipped-aurora/easy-deploy/server/global"

// TbProjectGroup 项目组表
type TbProjectGroup struct {
	global.GVA_MODEL
	GroupName   string `json:"groupName" form:"groupName" gorm:"column:group_name;comment:项目组名称"`
	Description string `json:"description" form:"description" gorm:"column:description;comment:项目组描述"`
	UserId      uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:创建用户ID"`
}

func (TbProjectGroup) TableName() string {
	return "tb_project_group"
}
