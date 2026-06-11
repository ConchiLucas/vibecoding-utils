package system

import "github.com/flipped-aurora/easy-deploy/server/global"

// TbDevelopmentPrepare stores project-scoped preparation snippets before development starts.
type TbDevelopmentPrepare struct {
	global.GVA_MODEL
	ProjectConfigId   uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;index;comment:所属项目配置ID"`
	ProjectConfigName string `json:"projectConfigName" form:"projectConfigName" gorm:"column:project_config_name;type:varchar(255);index;comment:所属项目配置名称"`
	BusinessGroup     string `json:"businessGroup" form:"businessGroup" gorm:"column:business_group;type:varchar(128);index;comment:业务分组"`
	Title             string `json:"title" form:"title" gorm:"column:title;type:varchar(255);comment:标题"`
	ItemType          string `json:"itemType" form:"itemType" gorm:"column:item_type;type:varchar(32);index;comment:准备类型"`
	Language          string `json:"language" form:"language" gorm:"column:language;type:varchar(64);comment:代码语言"`
	Tags              string `json:"tags" form:"tags" gorm:"column:tags;type:varchar(255);comment:标签"`
	Summary           string `json:"summary" form:"summary" gorm:"column:summary;type:varchar(512);comment:摘要"`
	Content           string `json:"content" form:"content" gorm:"column:content;type:text;comment:内容"`
	IsPinned          bool   `json:"isPinned" form:"isPinned" gorm:"column:is_pinned;comment:是否置顶"`
	Sort              int    `json:"sort" form:"sort" gorm:"column:sort;comment:排序"`
	UserId            uint   `json:"userId" form:"userId" gorm:"column:user_id;index;comment:归属用户ID"`
}

func (TbDevelopmentPrepare) TableName() string {
	return "tb_development_prepare"
}
