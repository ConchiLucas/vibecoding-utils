package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbEntityRouter struct{}

func (s *TbEntityRouter) InitTbEntityRouter(Router *gin.RouterGroup) {
	entityRouter := Router.Group("entity")
	entityRouterWithoutRecord := Router.Group("entity")
	var entityApi = v1.ApiGroupApp.SystemApiGroup.TbEntityApi
	{
		entityRouter.POST("createTbEntity", entityApi.CreateTbEntity)
		entityRouter.DELETE("deleteTbEntity", entityApi.DeleteTbEntity)
		entityRouter.DELETE("deleteTbEntityByIds", entityApi.DeleteTbEntityByIds)
		entityRouter.PUT("updateTbEntity", entityApi.UpdateTbEntity)
	}
	{
		entityRouterWithoutRecord.GET("getTbEntity", entityApi.GetTbEntity)
		entityRouterWithoutRecord.GET("getTbEntityList", entityApi.GetTbEntityList)
	}
}
