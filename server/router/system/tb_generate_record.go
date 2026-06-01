package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateRecordRouter struct{}

func (s *TbGenerateRecordRouter) InitTbGenerateRecordRouter(Router *gin.RouterGroup) {
	tbGenerateRecordRouter := Router.Group("tbgeneraterecord")
	tbGenerateRecordApi := api.ApiGroupApp.SystemApiGroup.TbGenerateRecordApi
	{
		tbGenerateRecordRouter.POST("createTbGenerateRecord", tbGenerateRecordApi.CreateTbGenerateRecord)
		tbGenerateRecordRouter.DELETE("deleteTbGenerateRecord", tbGenerateRecordApi.DeleteTbGenerateRecord)
		tbGenerateRecordRouter.PUT("updateTbGenerateRecord", tbGenerateRecordApi.UpdateTbGenerateRecord)
		tbGenerateRecordRouter.GET("getTbGenerateRecord", tbGenerateRecordApi.GetTbGenerateRecord)
		tbGenerateRecordRouter.GET("getTbGenerateRecordList", tbGenerateRecordApi.GetTbGenerateRecordList)
		tbGenerateRecordRouter.GET("getGenerateRecordByUser", tbGenerateRecordApi.GetGenerateRecordByUser)
	}
}
