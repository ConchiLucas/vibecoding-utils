package system



type TbGenerateProject struct {
	ID              uint   `gorm:"primarykey" json:"ID"`
	ProjectConfigId int    `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;type:int;default:0;comment:所属项目配置ID"`
	ProjectName     string `json:"projectName" gorm:"type:varchar(255);comment:项目名称"`
	DiskPath        string `json:"diskPath" gorm:"type:varchar(255);comment:磁盘路径"`
	Remark          string `json:"remark" gorm:"type:varchar(255);comment:备注"`
	UserName        string `json:"userName" gorm:"type:varchar(255);comment:用户名称"`
}

func (TbGenerateProject) TableName() string {
	return "tb_generate_project"
}
