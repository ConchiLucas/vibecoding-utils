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
		tbGenerateProjectPathRouter.DELETE("deleteTbGenerateProjectPath", tbGenerateProjectPathApi.DeleteTbGenerateProjectPath)
		tbGenerateProjectPathRouter.POST("deletePathSet", tbGenerateProjectPathApi.DeletePathSet)
		tbGenerateProjectPathRouter.POST("renamePathSet", tbGenerateProjectPathApi.RenamePathSet)
		tbGenerateProjectPathRouter.PUT("updateTbGenerateProjectPath", tbGenerateProjectPathApi.UpdateTbGenerateProjectPath)
		tbGenerateProjectPathRouter.GET("getTbGenerateProjectPath", tbGenerateProjectPathApi.GetTbGenerateProjectPath)
		tbGenerateProjectPathRouter.GET("getTbGenerateProjectPathList", tbGenerateProjectPathApi.GetTbGenerateProjectPathList)
		tbGenerateProjectPathRouter.POST("updateEnabled", tbGenerateProjectPathApi.UpdateEnabled)
		tbGenerateProjectPathRouter.POST("copyPathSet", tbGenerateProjectPathApi.CopyPathSet)
		tbGenerateProjectPathRouter.POST("buildPromptSummary", tbGenerateProjectPathApi.BuildPromptSummary)
	}
}
