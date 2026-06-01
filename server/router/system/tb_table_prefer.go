package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbTablePreferRouter struct{}

func (s *TbTablePreferRouter) InitTbTablePreferRouter(Router *gin.RouterGroup) {
	preferRouter := Router.Group("tablePrefer")
	preferRouterWithoutRecord := Router.Group("tablePrefer")
	var preferApi = v1.ApiGroupApp.SystemApiGroup.TbTablePreferApi
	{
		preferRouter.POST("createTbTablePrefer", preferApi.CreateTbTablePrefer)
		preferRouter.DELETE("deleteTbTablePrefer", preferApi.DeleteTbTablePrefer)
		preferRouter.DELETE("deleteTbTablePreferByIds", preferApi.DeleteTbTablePreferByIds)
		preferRouter.PUT("updateTbTablePrefer", preferApi.UpdateTbTablePrefer)
	}
	{
		preferRouterWithoutRecord.GET("getTbTablePrefer", preferApi.GetTbTablePrefer)
		preferRouterWithoutRecord.GET("getTbTablePreferList", preferApi.GetTbTablePreferList)
		preferRouterWithoutRecord.GET("getHistoryTableNames", preferApi.GetHistoryTableNames)
		preferRouterWithoutRecord.POST("getPreferVOByParams", preferApi.GetPreferVOByParams)
		preferRouterWithoutRecord.POST("getPreferColumnValueList", preferApi.GetPreferColumnValueList)
	}
}
