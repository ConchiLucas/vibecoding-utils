package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbTableRouter struct{}

func (s *TbTableRouter) InitTbTableRouter(Router *gin.RouterGroup) {
	tableRouter := Router.Group("table")
	tableRouterWithoutRecord := Router.Group("table")
	var tableApi = v1.ApiGroupApp.SystemApiGroup.TbTableApi
	{
		tableRouter.POST("createTbTable", tableApi.CreateTbTable)
		tableRouter.DELETE("deleteTbTable", tableApi.DeleteTbTable)
		tableRouter.DELETE("deleteTbTableByIds", tableApi.DeleteTbTableByIds)
		tableRouter.PUT("updateTbTable", tableApi.UpdateTbTable)
	}
	{
		tableRouterWithoutRecord.GET("getTbTable", tableApi.GetTbTable)
		tableRouterWithoutRecord.GET("getTbTableList", tableApi.GetTbTableList)
		tableRouterWithoutRecord.POST("fuzzyQuery", tableApi.TableFuzzyQuery)
	}
}
