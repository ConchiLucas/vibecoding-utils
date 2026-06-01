package system

import "github.com/gin-gonic/gin"

type ProjectGroupRouter struct{}

func (r *ProjectGroupRouter) InitProjectGroupRouter(Router *gin.RouterGroup) {
	groupRouter := Router.Group("projectGroup")
	{
		groupRouter.POST("list", projectGroupApi.GetGroupList)
		groupRouter.POST("saveOrUpdate", projectGroupApi.SaveOrUpdateGroup)
		groupRouter.DELETE("delete/:id", projectGroupApi.DeleteGroup)
	}
}
