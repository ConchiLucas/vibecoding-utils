package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbTableRelateRouter struct{}

func (s *TbTableRelateRouter) InitTbTableRelateRouter(Router *gin.RouterGroup) {
	trRouter := Router.Group("tableRelate")
	trRouterWithoutRecord := Router.Group("tableRelate")
	var trApi = v1.ApiGroupApp.SystemApiGroup.TbTableRelateApi
	{
		trRouter.POST("createTbTableRelate", trApi.CreateTbTableRelate)
		trRouter.DELETE("deleteTbTableRelate", trApi.DeleteTbTableRelate)
		trRouter.DELETE("deleteTbTableRelateByIds", trApi.DeleteTbTableRelateByIds)
		trRouter.PUT("updateTbTableRelate", trApi.UpdateTbTableRelate)
	}
	{
		trRouterWithoutRecord.GET("getTbTableRelate", trApi.GetTbTableRelate)
		trRouterWithoutRecord.GET("getTbTableRelateList", trApi.GetTbTableRelateList)
		trRouterWithoutRecord.POST("getClientData", trApi.GetClientData)
		trRouterWithoutRecord.POST("getRemoteColumns", trApi.GetRemoteColumns)
		trRouterWithoutRecord.POST("getTableComments", trApi.GetTableComments)
	}
}
