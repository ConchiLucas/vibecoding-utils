package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateProjectPathRouter struct{}

func (s *TbGenerateProjectPathRouter) InitTbGenerateProjectPathRouter(Router *gin.RouterGroup) {
	tbGenerateProjectPathRouter := Router.Group("tbgenerateprojectpath")
	tbGenerateProjectPathApi := api.ApiGroupApp.SystemApiGroup.TbGenerateProjectPathApi
	{
		tbGenerateProjectPathRouter.POST("createTbGenerateProjectPath", tbGenerateProjectPathApi.CreateTbGenerateProjectPath)
		tbGenerateProjectPathRouter.POST("createPathGroup", tbGenerateProjectPathApi.CreatePathGroup)
		tbGenerateProjectPathRouter.DELETE("deleteTbGenerateProjectPath", tbGenerateProjectPathApi.DeleteTbGenerateProjectPath)
		tbGenerateProjectPathRouter.DELETE("deletePathGroup", tbGenerateProjectPathApi.DeletePathGroup)
		tbGenerateProjectPathRouter.POST("deletePathSet", tbGenerateProjectPathApi.DeletePathSet)
		tbGenerateProjectPathRouter.POST("renamePathSet", tbGenerateProjectPathApi.RenamePathSet)
		tbGenerateProjectPathRouter.PUT("updateTbGenerateProjectPath", tbGenerateProjectPathApi.UpdateTbGenerateProjectPath)
		tbGenerateProjectPathRouter.PUT("updatePathGroup", tbGenerateProjectPathApi.UpdatePathGroup)
		tbGenerateProjectPathRouter.GET("getTbGenerateProjectPath", tbGenerateProjectPathApi.GetTbGenerateProjectPath)
		tbGenerateProjectPathRouter.GET("getTbGenerateProjectPathList", tbGenerateProjectPathApi.GetTbGenerateProjectPathList)
		tbGenerateProjectPathRouter.GET("getPathGroupList", tbGenerateProjectPathApi.GetPathGroupList)
		tbGenerateProjectPathRouter.POST("updateEnabled", tbGenerateProjectPathApi.UpdateEnabled)
		tbGenerateProjectPathRouter.POST("copyPathSet", tbGenerateProjectPathApi.CopyPathSet)
		tbGenerateProjectPathRouter.POST("buildPromptSummary", tbGenerateProjectPathApi.BuildPromptSummary)
	}
}
