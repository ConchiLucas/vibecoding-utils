package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateProjectInstanceRouter struct{}

func (s *TbGenerateProjectInstanceRouter) InitTbGenerateProjectInstanceRouter(Router *gin.RouterGroup) {
	tbGenerateProjectInstanceRouter := Router.Group("tbgenerateprojectinstance")
	tbGenerateProjectInstanceApi := api.ApiGroupApp.SystemApiGroup.TbGenerateProjectInstanceApi
	{
		tbGenerateProjectInstanceRouter.POST("createTbGenerateProjectInstance", tbGenerateProjectInstanceApi.CreateTbGenerateProjectInstance)
		tbGenerateProjectInstanceRouter.DELETE("deleteTbGenerateProjectInstance", tbGenerateProjectInstanceApi.DeleteTbGenerateProjectInstance)
		tbGenerateProjectInstanceRouter.PUT("updateTbGenerateProjectInstance", tbGenerateProjectInstanceApi.UpdateTbGenerateProjectInstance)
		tbGenerateProjectInstanceRouter.PUT("updateSelectedPathSet", tbGenerateProjectInstanceApi.UpdateSelectedPathSet)
		tbGenerateProjectInstanceRouter.GET("getTbGenerateProjectInstance", tbGenerateProjectInstanceApi.GetTbGenerateProjectInstance)
		tbGenerateProjectInstanceRouter.GET("getTbGenerateProjectInstanceList", tbGenerateProjectInstanceApi.GetTbGenerateProjectInstanceList)
	}
}
