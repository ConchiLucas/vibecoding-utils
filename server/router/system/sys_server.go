package system

import (
	"github.com/gin-gonic/gin"
)

type ServerRouter struct{}

func (r *ServerRouter) InitServerRouter(Router *gin.RouterGroup) {
	serverRouter := Router.Group("server")
	{
		serverRouter.POST("page", serverApi.GetServerPage)
		serverRouter.POST("list", serverApi.GetServerList)
		serverRouter.GET("getById/:id", serverApi.GetServerById)
		serverRouter.POST("saveOrUpdate", serverApi.SaveOrUpdateServer)
		serverRouter.DELETE("delete/:ids", serverApi.DeleteServer)
	}
}
