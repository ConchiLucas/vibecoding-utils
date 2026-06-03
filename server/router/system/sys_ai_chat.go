package system

import (
	"github.com/gin-gonic/gin"
)

type AIChatRouter struct{}

func (r *AIChatRouter) InitAIChatRouter(Router *gin.RouterGroup) {
	aiChatRouter := Router.Group("ai")
	{
		aiChatRouter.POST("chat", aiChatApi.ChatStream) // SSE 流式对话
		aiChatRouter.GET("models", aiChatApi.GetModels) // 获取当前模型信息
	}
}

type AIChatHistoryRouter struct{}

func (r *AIChatHistoryRouter) InitAIChatHistoryRouter(Router *gin.RouterGroup) {
	aiChatRouter := Router.Group("ai")
	{
		aiChatRouter.POST("chat/history", aiChatHistoryApi.SaveOrUpdate)  // 保存或更新对话历史
		aiChatRouter.GET("chat/history", aiChatHistoryApi.List)           // 查询对话历史列表
		aiChatRouter.GET("chat/history/:chatId", aiChatHistoryApi.Detail) // 查询对话历史详情
	}
}

func (r *AIChatRouter) InitAIProviderRouter(Router *gin.RouterGroup) {
	aiChatRouter := Router.Group("ai")
	{
		aiChatRouter.GET("providers", aiChatApi.GetProviders) // 获取 AI 厂商配置
	}
}

func (r *AIChatRouter) InitAIConfigRouter(Router *gin.RouterGroup) {
	aiChatRouter := Router.Group("ai")
	{
		aiChatRouter.GET("config", aiChatApi.GetConfig)                // 获取完整 AI 配置
		aiChatRouter.POST("config", aiChatApi.SaveConfig)              // 保存完整 AI 配置
		aiChatRouter.POST("config/active", aiChatApi.SaveActiveConfig) // 保存默认 AI 配置
	}
}
