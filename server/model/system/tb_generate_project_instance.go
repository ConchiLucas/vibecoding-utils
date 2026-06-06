package system

type TbGenerateProjectInstance struct {
	ID                      uint   `gorm:"primarykey" json:"ID"`
	TemplateProjectId       int    `json:"templateProjectId" form:"templateProjectId" gorm:"column:template_project_id;type:int;index;comment:所属代码生成模板卡片ID"`
	ProjectName             string `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);comment:项目名称"`
	DiskPath                string `json:"diskPath" form:"diskPath" gorm:"column:disk_path;type:varchar(500);comment:磁盘输出路径"`
	Remark                  string `json:"remark" form:"remark" gorm:"column:remark;type:varchar(500);comment:备注"`
	UserName                string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	SelectedPathSetIdentity string `json:"selectedPathSetIdentity" form:"selectedPathSetIdentity" gorm:"column:selected_path_set_identity;type:varchar(255);default:'';comment:当前选中路径配置标识"`
}

func (TbGenerateProjectInstance) TableName() string {
	return "tb_generate_project_instance"
}
