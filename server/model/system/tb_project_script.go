package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

// TbProjectScript 项目脚本表
type TbProjectScript struct {
	global.GVA_MODEL
	ProjectId    int    `json:"projectId" form:"projectId" gorm:"column:project_id;comment:项目id"`
	RouteId      int    `json:"routeId" form:"routeId" gorm:"column:route_id;default:0;comment:专属关联路由id"`
	ScriptType   uint   `json:"scriptType" form:"scriptType" gorm:"column:script_type;default:0;comment:脚本环境 (0=通用, 1=仅本地, 2=仅服务器)"`
	Content      string `json:"content" form:"content" gorm:"type:text;comment:脚本文本内容"`
	FileName     string `json:"fileName" form:"fileName" gorm:"column:file_name;comment:文件名称"`
	FileNickName string `json:"fileNickName" form:"fileNickName" gorm:"column:file_nick_name;comment:文件别称"`
}

func (TbProjectScript) TableName() string {
	return "tb_project_script"
}
