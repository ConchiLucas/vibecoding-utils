package system

import (
	v1 "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceServerUserRouter struct{}

func (s *TbInterfaceServerUserRouter) InitTbInterfaceServerUserRouter(Router *gin.RouterGroup) {
	serverUserRouter := Router.Group("serverUser")
	serverUserRouterWithoutRecord := Router.Group("serverUser")
	var serverUserApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceServerUserApi
	{
		serverUserRouter.POST("createTbInterfaceServerUser", serverUserApi.CreateTbInterfaceServerUser)
		serverUserRouter.DELETE("deleteTbInterfaceServerUser", serverUserApi.DeleteTbInterfaceServerUser)
		serverUserRouter.PUT("updateTbInterfaceServerUser", serverUserApi.UpdateTbInterfaceServerUser)
		serverUserRouter.POST("updateClientStatus", serverUserApi.UpdateClientStatus)
	}
	{
		serverUserRouterWithoutRecord.GET("getTbInterfaceServerUserList", serverUserApi.GetTbInterfaceServerUserList)
	}
}
