package system

import "github.com/gin-gonic/gin"

type LogManagerRouter struct{}

func (r *LogManagerRouter) InitLogManagerRouter(Router *gin.RouterGroup) {
	logRouter := Router.Group("logManager")
	{
		logRouter.POST("page", logManagerApi.GetLogProjectPage)
		logRouter.GET("groups", logManagerApi.GetLogProjectGroups)
		logRouter.POST("saveOrUpdateProject", logManagerApi.SaveOrUpdateLogProject)
		logRouter.DELETE("deleteProject/:ids", logManagerApi.DeleteLogProject)
		logRouter.POST("saveOrUpdateRoute", logManagerApi.SaveOrUpdateLogRoute)
		logRouter.DELETE("deleteRoute/:id", logManagerApi.DeleteLogRoute)
		logRouter.GET("dockerServices/:id", logManagerApi.ListDockerServices)
		logRouter.GET("serviceStatusStream/:id", logManagerApi.ServiceStatusStream)
		logRouter.GET("serviceGroupStream/:id", logManagerApi.ServiceGroupStream)
		logRouter.GET("deployStream/:id", logManagerApi.DeployStream)
		logRouter.GET("stopStream/:id", logManagerApi.StopStream)
		logRouter.GET("restartStream/:id", logManagerApi.RestartStream)
		logRouter.GET("dockerLogStream/:id", logManagerApi.DockerLogStream)
	}
}
