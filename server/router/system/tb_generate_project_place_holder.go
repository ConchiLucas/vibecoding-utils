package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateProjectPlaceHolderRouter struct{}

func (s *TbGenerateProjectPlaceHolderRouter) InitTbGenerateProjectPlaceHolderRouter(Router *gin.RouterGroup) {
	tbGenerateProjectPlaceHolderRouter := Router.Group("tbgenerateprojectplaceholder")
	tbGenerateProjectPlaceHolderApi := api.ApiGroupApp.SystemApiGroup.TbGenerateProjectPlaceHolderApi
	{
		tbGenerateProjectPlaceHolderRouter.POST("createTbGenerateProjectPlaceHolder", tbGenerateProjectPlaceHolderApi.CreateTbGenerateProjectPlaceHolder)
		tbGenerateProjectPlaceHolderRouter.DELETE("deleteTbGenerateProjectPlaceHolder", tbGenerateProjectPlaceHolderApi.DeleteTbGenerateProjectPlaceHolder)
		tbGenerateProjectPlaceHolderRouter.PUT("updateTbGenerateProjectPlaceHolder", tbGenerateProjectPlaceHolderApi.UpdateTbGenerateProjectPlaceHolder)
		tbGenerateProjectPlaceHolderRouter.GET("getTbGenerateProjectPlaceHolder", tbGenerateProjectPlaceHolderApi.GetTbGenerateProjectPlaceHolder)
		tbGenerateProjectPlaceHolderRouter.GET("getTbGenerateProjectPlaceHolderList", tbGenerateProjectPlaceHolderApi.GetTbGenerateProjectPlaceHolderList)
	}
}
