package system

type TbGenerateProject struct {
	ID                        uint   `gorm:"primarykey" json:"ID"`
	ProjectConfigId           int    `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:int;default:0;comment:所属项目配置ID"`
	BusinessType              string `json:"businessType" form:"businessType" gorm:"column:business_type;type:varchar(128);default:'';comment:业务类型"`
	ProjectName               string `json:"projectName" gorm:"type:varchar(255);comment:项目名称"`
	DiskPath                  string `json:"diskPath" gorm:"type:varchar(255);comment:磁盘路径"`
	Remark                    string `json:"remark" gorm:"type:varchar(255);comment:备注"`
	UserName                  string `json:"userName" gorm:"type:varchar(255);comment:用户名称"`
	SelectedProjectInstanceId int    `json:"selectedProjectInstanceId" form:"selectedProjectInstanceId" gorm:"column:selected_project_instance_id;type:int;default:0;comment:当前选中项目实例ID"`
}

func (TbGenerateProject) TableName() string {
	return "tb_generate_project"
}
