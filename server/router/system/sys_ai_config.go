package system

import "github.com/gin-gonic/gin"

type AIConfigRouter struct{}

func (r *AIConfigRouter) InitAIConfigRouter(router *gin.RouterGroup) {
	aiConfigRouter := router.Group("ai")
	{
		aiConfigRouter.GET("config", aiConfigApi.GetConfig)
		aiConfigRouter.POST("config", aiConfigApi.SaveConfig)
		aiConfigRouter.POST("config/active", aiConfigApi.SaveActiveConfig)
	}
}
