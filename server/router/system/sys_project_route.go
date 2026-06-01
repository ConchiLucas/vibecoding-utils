package system

import (
	"github.com/gin-gonic/gin"
)

type ProjectRouteRouter struct{}

func (r *ProjectRouteRouter) InitProjectRouteRouter(Router *gin.RouterGroup) {
	projectRouteRouter := Router.Group("projectRoute")
	{
		projectRouteRouter.POST("saveOrUpdate", projectRouteApi.SaveOrUpdateRoute)
		projectRouteRouter.DELETE("deleteRoute", projectRouteApi.DeleteRoute)
	}
}
