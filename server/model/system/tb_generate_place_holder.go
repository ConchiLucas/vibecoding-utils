package system



type TbGeneratePlaceHolder struct {
	ID           uint   `gorm:"primarykey" json:"ID"`
	UserName     string `json:"userName" gorm:"type:varchar(255);comment:用户名"`
	HolderKey    string `json:"holderKey" gorm:"type:varchar(255);comment:字典key"`
	HolderValue  string `json:"holderValue" gorm:"type:varchar(255);comment:字典value"`
	HolderDesc   string `json:"holderDesc" gorm:"type:varchar(255);comment:字典描述"`
	ExampleValue string `json:"exampleValue" gorm:"type:varchar(255);comment:示例值"`
	IsEditable   int    `json:"isEditable" gorm:"comment:是否允许修改 0-否 1-是"`
}

func (TbGeneratePlaceHolder) TableName() string {
	return "tb_generate_place_holder"
}
