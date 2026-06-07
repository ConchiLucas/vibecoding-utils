package system

type TbGenerateProjectPathGroup struct {
	ID                uint   `gorm:"primarykey" json:"ID"`
	ProjectId         int    `json:"projectId" gorm:"column:project_id;comment:项目id"`
	ProjectInstanceId int    `json:"projectInstanceId" gorm:"column:project_instance_id;index;comment:项目实例id"`
	PathSet           int    `json:"pathSet" gorm:"column:path_set;index;default:0;comment:路径配置分组"`
	PathSetName       string `json:"pathSetName" gorm:"column:path_set_name;type:varchar(100);comment:路径配置名称"`
	BasePath          string `json:"basePath" gorm:"column:base_path;type:varchar(500);comment:公共相对路径"`
	Sort              int    `json:"sort" gorm:"column:sort;default:0;comment:排序"`
}

func (TbGenerateProjectPathGroup) TableName() string {
	return "tb_generate_project_path_group"
}
