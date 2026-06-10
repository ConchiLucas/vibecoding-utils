package system

import "time"

type TbGenerateFieldSnippet struct {
	ID           uint      `gorm:"primarykey" json:"ID"`
	BusinessType string    `json:"businessType" form:"businessType" gorm:"column:business_type;type:varchar(128);index;default:'';comment:业务类型"`
	Name         string    `json:"name" form:"name" gorm:"type:varchar(255);default:'';comment:记录名称"`
	SourceText   string    `json:"sourceText" gorm:"column:source_text;type:text;comment:表结构或字段内容"`
	Snippets     string    `json:"snippets" gorm:"type:text;comment:字段片段模板 JSON"`
	Rendered     string    `json:"rendered" gorm:"type:text;comment:预览渲染结果 JSON"`
	UserName     string    `json:"userName" gorm:"column:user_name;type:varchar(255);default:'';comment:用户名称"`
	CreatedAt    time.Time `json:"createdAt" gorm:"column:created_at;comment:创建时间"`
}

func (TbGenerateFieldSnippet) TableName() string {
	return "tb_generate_field_snippet"
}
