package system

type TbGenerateProjectPath struct {
	ID                  uint   `gorm:"primarykey" json:"ID"`
	ProjectId           int    `json:"projectId" gorm:"column:project_id;comment:项目id"`
	ProjectInstanceId   int    `json:"projectInstanceId" gorm:"column:project_instance_id;index;comment:项目实例id"`
	PathSet             int    `json:"pathSet" gorm:"column:path_set;index;default:0;comment:路径配置分组"`
	PathSetName         string `json:"pathSetName" gorm:"column:path_set_name;type:varchar(100);comment:路径配置名称"`
	PathGroupId         int    `json:"pathGroupId" gorm:"column:path_group_id;index;default:0;comment:路径分组id"`
	FileUrl             string `json:"fileUrl" gorm:"type:varchar(500);comment:文件路径"`
	FileName            string `json:"fileName" gorm:"type:varchar(255);comment:文件名（可用占位符）"`
	DynamicPlaceholders string `json:"dynamicPlaceholders" gorm:"column:dynamic_placeholders;type:text;comment:动态占位符配置"`
	Enabled             int    `json:"enabled" gorm:"comment:是否可用 0-不可用 1-可用"`
	Incremented         int    `json:"incremented" gorm:"comment:是否增量 0-否 1-是"`
}

func (TbGenerateProjectPath) TableName() string {
	return "tb_generate_project_path"
}
