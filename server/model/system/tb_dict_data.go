package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

type TbDictData struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	DictType     string `json:"dictType" gorm:"column:dict_type;type:varchar(255);comment:字典类型"`
	DictLabel    string `json:"dictLabel" gorm:"column:dict_label;type:varchar(255);comment:字典标签"`
	DictValue    string `json:"dictValue" gorm:"column:dict_value;type:varchar(255);comment:字典值"`
	LabelClass   string `json:"labelClass" gorm:"column:label_class;type:varchar(100);comment:标签样式"`
	UserName     string `json:"userName" gorm:"column:user_name;type:varchar(100);comment:用户名称"`
	ExtendParams string `json:"extendParams" gorm:"column:extend_params;type:varchar(4096);comment:json扩展字段"`
}

func (TbDictData) TableName() string {
	return "tb_dict_data"
}
