package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceLogRouter struct{}

func (s *TbInterfaceLogRouter) InitTbInterfaceLogRouter(Router *gin.RouterGroup) {
	logRouter := Router.Group("interfaceLog")
	logRouterWithoutRecord := Router.Group("interfaceLog")
	var logApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceLogApi
	{
		logRouter.POST("createTbInterfaceLog", logApi.CreateTbInterfaceLog)
		logRouter.DELETE("deleteTbInterfaceLog", logApi.DeleteTbInterfaceLog)
		logRouter.DELETE("deleteTbInterfaceLogByIds", logApi.DeleteTbInterfaceLogByIds)
		logRouter.PUT("updateTbInterfaceLog", logApi.UpdateTbInterfaceLog)
	}
	{
		logRouterWithoutRecord.GET("getTbInterfaceLog", logApi.GetTbInterfaceLog)
		logRouterWithoutRecord.GET("getTbInterfaceLogList", logApi.GetTbInterfaceLogList)
		logRouterWithoutRecord.GET("getParamsPreview", logApi.GetParamsPreview)
	}
}
