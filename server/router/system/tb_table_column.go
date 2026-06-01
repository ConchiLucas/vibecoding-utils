package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbTableColumnRouter struct{}

func (s *TbTableColumnRouter) InitTbTableColumnRouter(Router *gin.RouterGroup) {
	tcRouter := Router.Group("tableColumn")
	tcRouterWithoutRecord := Router.Group("tableColumn")
	var tcApi = v1.ApiGroupApp.SystemApiGroup.TbTableColumnApi
	{
		tcRouter.POST("createTbTableColumn", tcApi.CreateTbTableColumn)
		tcRouter.DELETE("deleteTbTableColumn", tcApi.DeleteTbTableColumn)
		tcRouter.DELETE("deleteTbTableColumnByIds", tcApi.DeleteTbTableColumnByIds)
		tcRouter.PUT("updateTbTableColumn", tcApi.UpdateTbTableColumn)
	}
	{
		tcRouterWithoutRecord.GET("getTbTableColumn", tcApi.GetTbTableColumn)
		tcRouterWithoutRecord.GET("getTbTableColumnList", tcApi.GetTbTableColumnList)
	}
}
