package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbInterfaceProjectRouter struct{}

func (s *TbInterfaceProjectRouter) InitTbInterfaceProjectRouter(Router *gin.RouterGroup) {
	projectRouter := Router.Group("project")
	projectRouterWithoutRecord := Router.Group("project")
	var projectApi = v1.ApiGroupApp.SystemApiGroup.TbInterfaceProjectApi
	{
		projectRouter.POST("createTbInterfaceProject", projectApi.CreateTbInterfaceProject)
		projectRouter.DELETE("deleteTbInterfaceProject", projectApi.DeleteTbInterfaceProject)
		projectRouter.PUT("updateTbInterfaceProject", projectApi.UpdateTbInterfaceProject)
	}
	{
		projectRouterWithoutRecord.GET("getTbInterfaceProjectList", projectApi.GetTbInterfaceProjectList)
	}
}
