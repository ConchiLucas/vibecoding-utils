package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceParamsRouter struct{}

func (s *TbInterfaceParamsRouter) InitTbInterfaceParamsRouter(Router *gin.RouterGroup) {
	paramsRouter := Router.Group("interfaceParams")
	paramsRouterWithoutRecord := Router.Group("interfaceParams")
	var paramsApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceParamsApi
	{
		paramsRouter.POST("createTbInterfaceParams", paramsApi.CreateTbInterfaceParams)
		paramsRouter.DELETE("deleteTbInterfaceParams", paramsApi.DeleteTbInterfaceParams)
		paramsRouter.DELETE("deleteTbInterfaceParamsByIds", paramsApi.DeleteTbInterfaceParamsByIds)
		paramsRouter.PUT("updateTbInterfaceParams", paramsApi.UpdateTbInterfaceParams)
	}
	{
		paramsRouterWithoutRecord.GET("getTbInterfaceParams", paramsApi.GetTbInterfaceParams)
		paramsRouterWithoutRecord.GET("getTbInterfaceParamsList", paramsApi.GetTbInterfaceParamsList)
		paramsRouterWithoutRecord.POST("getParamsEntity", paramsApi.GetParamsEntity)
	}
}
