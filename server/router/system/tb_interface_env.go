package system

import (
	v1 "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceEnvRouter struct{}

func (s *TbInterfaceEnvRouter) InitTbInterfaceEnvRouter(Router *gin.RouterGroup) {
	envRouter := Router.Group("interfaceEnv")
	envRouterWithoutRecord := Router.Group("interfaceEnv")
	var envApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceEnvApi
	{
		envRouter.POST("createTbInterfaceEnv", envApi.CreateTbInterfaceEnv)
		envRouter.DELETE("deleteTbInterfaceEnv", envApi.DeleteTbInterfaceEnv)
		envRouter.PUT("updateTbInterfaceEnv", envApi.UpdateTbInterfaceEnv)
	}
	{
		envRouterWithoutRecord.GET("getTbInterfaceEnvList", envApi.GetTbInterfaceEnvList)
	}
}
