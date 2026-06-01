package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbAgileTableSampleRouter struct{}

func (s *TbAgileTableSampleRouter) InitTbAgileTableSampleRouter(Router *gin.RouterGroup) {
	sampleRouter := Router.Group("agileTableSample")
	var sampleApi = v1.ApiGroupApp.SystemApiGroup.TbAgileTableSampleApi
	{
		sampleRouter.GET("list", sampleApi.List)
		sampleRouter.GET("history", sampleApi.History)
		sampleRouter.POST("save", sampleApi.Save)
	}
}
