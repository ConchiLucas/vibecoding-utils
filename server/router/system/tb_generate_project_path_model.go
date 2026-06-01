package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateProjectPathModelRouter struct{}

func (s *TbGenerateProjectPathModelRouter) InitTbGenerateProjectPathModelRouter(Router *gin.RouterGroup) {
	tbGenerateProjectPathModelRouter := Router.Group("tbgenerateprojectpathmodel")
	tbGenerateProjectPathModelApi := api.ApiGroupApp.SystemApiGroup.TbGenerateProjectPathModelApi
	{
		tbGenerateProjectPathModelRouter.POST("createTbGenerateProjectPathModel", tbGenerateProjectPathModelApi.CreateTbGenerateProjectPathModel)
		tbGenerateProjectPathModelRouter.DELETE("deleteTbGenerateProjectPathModel", tbGenerateProjectPathModelApi.DeleteTbGenerateProjectPathModel)
		tbGenerateProjectPathModelRouter.PUT("updateTbGenerateProjectPathModel", tbGenerateProjectPathModelApi.UpdateTbGenerateProjectPathModel)
		tbGenerateProjectPathModelRouter.GET("getTbGenerateProjectPathModel", tbGenerateProjectPathModelApi.GetTbGenerateProjectPathModel)
		tbGenerateProjectPathModelRouter.GET("getTbGenerateProjectPathModelList", tbGenerateProjectPathModelApi.GetTbGenerateProjectPathModelList)
	}
}
