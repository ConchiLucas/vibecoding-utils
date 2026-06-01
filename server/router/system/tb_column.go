package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbColumnRouter struct{}

func (s *TbColumnRouter) InitTbColumnRouter(Router *gin.RouterGroup) {
	colRouter := Router.Group("column")
	colRouterWithoutRecord := Router.Group("column")
	var colApi = v1.ApiGroupApp.SystemApiGroup.TbColumnApi
	{
		colRouter.POST("createTbColumn", colApi.CreateTbColumn)
		colRouter.DELETE("deleteTbColumn", colApi.DeleteTbColumn)
		colRouter.DELETE("deleteTbColumnByIds", colApi.DeleteTbColumnByIds)
		colRouter.PUT("updateTbColumn", colApi.UpdateTbColumn)
	}
	{
		colRouterWithoutRecord.GET("getTbColumn", colApi.GetTbColumn)
		colRouterWithoutRecord.GET("getTbColumnList", colApi.GetTbColumnList)
		colRouterWithoutRecord.POST("getColumnTree", colApi.GetColumnTree)
	}
}
