package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGeneratePlaceHolderRouter struct{}

func (s *TbGeneratePlaceHolderRouter) InitTbGeneratePlaceHolderRouter(Router *gin.RouterGroup) {
	tbGeneratePlaceHolderRouter := Router.Group("tbgenerateplaceholder")
	tbGeneratePlaceHolderApi := api.ApiGroupApp.SystemApiGroup.TbGeneratePlaceHolderApi
	{
		tbGeneratePlaceHolderRouter.POST("createTbGeneratePlaceHolder", tbGeneratePlaceHolderApi.CreateTbGeneratePlaceHolder)
		tbGeneratePlaceHolderRouter.DELETE("deleteTbGeneratePlaceHolder", tbGeneratePlaceHolderApi.DeleteTbGeneratePlaceHolder)
		tbGeneratePlaceHolderRouter.PUT("updateTbGeneratePlaceHolder", tbGeneratePlaceHolderApi.UpdateTbGeneratePlaceHolder)
		tbGeneratePlaceHolderRouter.GET("getTbGeneratePlaceHolder", tbGeneratePlaceHolderApi.GetTbGeneratePlaceHolder)
		tbGeneratePlaceHolderRouter.GET("getTbGeneratePlaceHolderList", tbGeneratePlaceHolderApi.GetTbGeneratePlaceHolderList)
	}
}
