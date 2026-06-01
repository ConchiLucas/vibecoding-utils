package system

import "github.com/flipped-aurora/easy-deploy/server/global"

// TbAIChatHistory stores AI assistant conversations per user.
type TbAIChatHistory struct {
	global.GVA_MODEL
	ChatID   string `json:"chatId" form:"chatId" gorm:"column:chat_id;size:64;not null;index:idx_ai_chat_user_chat,unique;comment:对话ID"`
	UserID   uint   `json:"userId" form:"userId" gorm:"column:user_id;not null;index:idx_ai_chat_user_chat,unique;index;comment:用户ID"`
	Title    string `json:"title" form:"title" gorm:"column:title;size:128;comment:对话标题"`
	Provider string `json:"provider" form:"provider" gorm:"column:provider;size:64;comment:AI厂商"`
	Messages string `json:"messages" form:"messages" gorm:"column:messages;type:text;comment:对话消息JSON"`
}

func (TbAIChatHistory) TableName() string {
	return "sys_ai_chat_history"
}
