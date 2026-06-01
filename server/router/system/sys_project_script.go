package system

import (
	"github.com/gin-gonic/gin"
)

type ProjectScriptRouter struct{}

func (r *ProjectScriptRouter) InitProjectScriptRouter(Router *gin.RouterGroup) {
	scriptRouter := Router.Group("script")
	{
		scriptRouter.POST("page", scriptApi.GetProjectScriptPage)
		scriptRouter.POST("list", scriptApi.GetProjectScriptList)
		scriptRouter.GET("getById/:id", scriptApi.GetProjectScriptById)
		scriptRouter.POST("saveOrUpdate", scriptApi.SaveOrUpdateProjectScript)
		scriptRouter.DELETE("delete/:ids", scriptApi.DeleteProjectScript)
		scriptRouter.POST("upload", scriptApi.UploadFile)
		scriptRouter.GET("download/:id", scriptApi.DownloadFile)
		scriptRouter.GET("preview/:id", scriptApi.PreviewFile)
		scriptRouter.POST("updateContent", scriptApi.UpdateContent)
	}
}
