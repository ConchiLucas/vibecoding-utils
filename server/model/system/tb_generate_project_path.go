package system



type TbGenerateProjectPath struct {
	ID          uint   `gorm:"primarykey" json:"ID"`
	ProjectId   int    `json:"projectId" gorm:"comment:项目id"`
	FileUrl     string `json:"fileUrl" gorm:"type:varchar(500);comment:文件路径"`
	FileName    string `json:"fileName" gorm:"type:varchar(255);comment:文件名（可用占位符）"`
	Enabled     int    `json:"enabled" gorm:"comment:是否可用 0-不可用 1-可用"`
	Incremented int    `json:"incremented" gorm:"comment:是否增量 0-否 1-是"`
}

func (TbGenerateProjectPath) TableName() string {
	return "tb_generate_project_path"
}
