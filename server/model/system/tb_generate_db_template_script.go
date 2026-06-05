package system

type TbGenerateDbTemplateScript struct {
	ID         uint   `gorm:"primarykey" json:"ID"`
	ProjectId  int    `json:"projectId" gorm:"column:project_id;comment:代码生成项目ID"`
	TypeId     int    `json:"typeId" gorm:"column:type_id;comment:业务类型ID"`
	ScriptName string `json:"scriptName" gorm:"column:script_name;type:varchar(255);comment:脚本名称"`
	ScriptKind string `json:"scriptKind" gorm:"column:script_kind;type:varchar(100);comment:脚本类型"`
	Content    string `json:"content" gorm:"type:text;comment:SQL脚本内容"`
	Sort       int    `json:"sort" gorm:"comment:排序"`
}

func (TbGenerateDbTemplateScript) TableName() string {
	return "tb_generate_db_template_script"
}
