package system

import (
	"github.com/gin-gonic/gin"
)

type ProjectRouter struct{}

func (r *ProjectRouter) InitProjectRouter(Router *gin.RouterGroup) {
	projectRouter := Router.Group("project")
	{
		projectRouter.POST("page", projectApi.GetProjectPage)
		projectRouter.POST("list", projectApi.GetProjectList)
		projectRouter.GET("getById/:id", projectApi.GetProjectById)
		projectRouter.GET("nextPort", projectApi.GetNextDeployPort)
		projectRouter.POST("saveOrUpdate", projectApi.SaveOrUpdateProject)
		projectRouter.DELETE("delete/:ids", projectApi.DeleteProject)
		projectRouter.POST("processDeploy/:id", projectApi.ProcessDeploy)
		projectRouter.GET("deployStream/:id", projectApi.ProcessDeployStream)
		projectRouter.GET("stopStream/:id", projectApi.ProcessStopStream)
		projectRouter.GET("dockerLogStream/:id", projectApi.ProcessDockerLogStream)
	}
}
