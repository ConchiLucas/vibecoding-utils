package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbAgileRequestRouter struct{}

func (s *TbAgileRequestRouter) InitTbAgileRequestRouter(Router *gin.RouterGroup) {
	agileRouter := Router.Group("agileRequest")
	var agileApi = v1.ApiGroupApp.SystemApiGroup.TbAgileRequestApi
	{
		agileRouter.POST("send", agileApi.Send)
		agileRouter.GET("history", agileApi.GetList)
		agileRouter.GET("detail", agileApi.GetByID)
		agileRouter.DELETE("history", agileApi.DeleteByID)
		agileRouter.DELETE("history/clear", agileApi.Clear)
	}
}
