package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateDbTemplateRouter struct{}

func (s *TbGenerateDbTemplateRouter) InitTbGenerateDbTemplateRouter(Router *gin.RouterGroup) {
	typeRouter := Router.Group("tbgeneratedbtemplatetype")
	scriptRouter := Router.Group("tbgeneratedbtemplatescript")
	typeApi := api.ApiGroupApp.SystemApiGroup.TbGenerateDbTemplateTypeApi
	scriptApi := api.ApiGroupApp.SystemApiGroup.TbGenerateDbTemplateScriptApi
	{
		typeRouter.POST("create", typeApi.Create)
		typeRouter.DELETE("delete", typeApi.Delete)
		typeRouter.PUT("update", typeApi.Update)
		typeRouter.GET("get", typeApi.Get)
		typeRouter.GET("list", typeApi.List)

		scriptRouter.POST("create", scriptApi.Create)
		scriptRouter.DELETE("delete", scriptApi.Delete)
		scriptRouter.PUT("update", scriptApi.Update)
		scriptRouter.GET("get", scriptApi.Get)
		scriptRouter.GET("list", scriptApi.List)
	}
}
