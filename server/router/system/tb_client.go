package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbClientRouter struct{}

func (s *TbClientRouter) InitTbClientRouter(Router *gin.RouterGroup) {
	clientRouter := Router.Group("client")
	clientRouterWithoutRecord := Router.Group("client")
	var clientApi = v1.ApiGroupApp.SystemApiGroup.TbClientApi
	{
		clientRouter.POST("createTbClient", clientApi.CreateTbClient)
		clientRouter.DELETE("deleteTbClient", clientApi.DeleteTbClient)
		clientRouter.DELETE("deleteTbClientByIds", clientApi.DeleteTbClientByIds)
		clientRouter.PUT("updateTbClient", clientApi.UpdateTbClient)
	}
	{
		clientRouterWithoutRecord.GET("getTbClient", clientApi.GetTbClient)
		clientRouterWithoutRecord.GET("getTbClientList", clientApi.GetTbClientList)
	}
}
