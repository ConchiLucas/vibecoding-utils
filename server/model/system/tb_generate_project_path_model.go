package system



type TbGenerateProjectPathModel struct {
	ID      uint   `gorm:"primarykey" json:"ID"`
	PathId  int    `json:"pathId" gorm:"comment:路径id"`
	Content string `json:"content" gorm:"type:text;comment:模板内容"`
}

func (TbGenerateProjectPathModel) TableName() string {
	return "tb_generate_project_path_model"
}
