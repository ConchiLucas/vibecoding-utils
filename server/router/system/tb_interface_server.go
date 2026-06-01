package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceServerRouter struct{}

func (s *TbInterfaceServerRouter) InitTbInterfaceServerRouter(Router *gin.RouterGroup) {
	serverRouter := Router.Group("server")
	serverRouterWithoutRecord := Router.Group("server")
	var serverApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceServerApi
	{
		serverRouter.POST("createTbInterfaceServer", serverApi.CreateTbInterfaceServer)
		serverRouter.DELETE("deleteTbInterfaceServer", serverApi.DeleteTbInterfaceServer)
		serverRouter.DELETE("deleteTbInterfaceServerByIds", serverApi.DeleteTbInterfaceServerByIds)
		serverRouter.PUT("updateTbInterfaceServer", serverApi.UpdateTbInterfaceServer)
		serverRouter.PUT("upload", serverApi.ImportSwaggerInterfaces)
		serverRouter.PUT("renameServer", serverApi.RenameServer)
	}
	{
		serverRouterWithoutRecord.GET("getTbInterfaceServer", serverApi.GetTbInterfaceServer)
		serverRouterWithoutRecord.GET("getTbInterfaceServerList", serverApi.GetTbInterfaceServerList)
		serverRouterWithoutRecord.POST("buildTree", serverApi.BuildServerTree)
	}
}
