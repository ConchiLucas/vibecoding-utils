package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

// TbProject 项目表
type TbProject struct {
	global.GVA_MODEL
	GroupId          uint   `json:"groupId" form:"groupId" gorm:"column:group_id;comment:项目组ID"`
	ComputerLanguage string `json:"computerLanguage" form:"computerLanguage" gorm:"column:computer_language;comment:语言类型"`
	ProjectName      string `json:"projectName" form:"projectName" gorm:"column:project_name;comment:项目名称"`
	Description      string `json:"description" form:"description" gorm:"column:description;comment:项目描述"`
	AccessUrl        string `json:"accessUrl" form:"accessUrl" gorm:"column:access_url;comment:访问路径"`
	LocalProjectPath string `json:"localProjectPath" form:"localProjectPath" gorm:"column:local_project_path;comment:本地项目全局路径"`
	UserId           uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:项目归属用户ID"`

	// 一对多环境配置
	Routes []TbProjectRoute `json:"routes" gorm:"foreignKey:ProjectId;references:ID"`
}

func (TbProject) TableName() string {
	return "tb_project"
}
