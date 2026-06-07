package system

type TbGenerateDbTemplateType struct {
	ID        uint   `gorm:"primarykey" json:"ID"`
	ProjectId int    `json:"projectId" gorm:"column:project_id;comment:代码生成项目ID"`
	TypeName  string `json:"typeName" gorm:"column:type_name;type:varchar(255);comment:业务类型名称"`
	Prompt    string `json:"prompt" gorm:"type:text;comment:业务类型提示词"`
	Sort      int    `json:"sort" gorm:"comment:排序"`
}

func (TbGenerateDbTemplateType) TableName() string {
	return "tb_generate_db_template_type"
}
