package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceRouter struct{}

func (s *TbInterfaceRouter) InitTbInterfaceRouter(Router *gin.RouterGroup) {
	interfaceRouter := Router.Group("interface")
	interfaceRouterWithoutRecord := Router.Group("interface")
	var interfaceApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceApi
	{
		interfaceRouter.POST("createTbInterface", interfaceApi.CreateTbInterface)
		interfaceRouter.DELETE("deleteTbInterface", interfaceApi.DeleteTbInterface)
		interfaceRouter.DELETE("deleteTbInterfaceByIds", interfaceApi.DeleteTbInterfaceByIds)
		interfaceRouter.PUT("updateTbInterface", interfaceApi.UpdateTbInterface)
		interfaceRouter.POST("forwardInterface", interfaceApi.ForwardInterface)
	}
	{
		interfaceRouterWithoutRecord.GET("getTbInterface", interfaceApi.GetTbInterface)
		interfaceRouterWithoutRecord.GET("getTbInterfaceList", interfaceApi.GetTbInterfaceList)
	}
}
