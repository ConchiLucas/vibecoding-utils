package system

import "github.com/flipped-aurora/easy-deploy/server/global"

type TbAIConfig struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	ConfigKey string `json:"configKey" gorm:"column:config_key;size:64;not null;uniqueIndex;comment:配置键"`
	Active    string `json:"active" gorm:"column:active;size:128;comment:默认AI配置ID"`
	Providers string `json:"providers" gorm:"column:providers;type:text;comment:AI配置JSON"`
}

func (TbAIConfig) TableName() string {
	return "tb_ai_config"
}
