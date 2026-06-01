package system



type TbGenerateProjectPlaceHolder struct {
	ID           uint   `gorm:"primarykey" json:"ID"`
	ProjectId    int    `json:"projectId" gorm:"comment:项目id"`
	UserName     string `json:"userName" gorm:"type:varchar(255);comment:用户名"`
	HolderKey    string `json:"holderKey" gorm:"type:varchar(255);comment:占位符key"`
	HolderValue  string `json:"holderValue" gorm:"type:varchar(255);comment:占位符value"`
	HolderDesc   string `json:"holderDesc" gorm:"type:varchar(255);comment:占位符描述"`
	ExampleValue string `json:"exampleValue" gorm:"type:varchar(255);comment:示例值"`
}

func (TbGenerateProjectPlaceHolder) TableName() string {
	return "tb_generate_project_place_holder"
}
