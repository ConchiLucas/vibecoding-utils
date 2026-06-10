package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateFieldSnippetRouter struct{}

func (s *TbGenerateFieldSnippetRouter) InitTbGenerateFieldSnippetRouter(Router *gin.RouterGroup) {
	fieldSnippetRouter := Router.Group("tbgeneratefieldsnippet")
	fieldSnippetApi := api.ApiGroupApp.SystemApiGroup.TbGenerateFieldSnippetApi
	{
		fieldSnippetRouter.GET("latest", fieldSnippetApi.GetLatestGenerateFieldSnippet)
		fieldSnippetRouter.GET("history", fieldSnippetApi.GetGenerateFieldSnippetHistory)
		fieldSnippetRouter.POST("preview", fieldSnippetApi.PreviewGenerateFieldSnippet)
		fieldSnippetRouter.POST("save", fieldSnippetApi.SaveGenerateFieldSnippet)
	}
}
