package system

import (
	api "github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbGenerateProjectRouter struct{}

func (s *TbGenerateProjectRouter) InitTbGenerateProjectRouter(Router *gin.RouterGroup) {
	tbGenerateProjectRouter := Router.Group("tbgenerateproject")
	tbGenerateProjectApi := api.ApiGroupApp.SystemApiGroup.TbGenerateProjectApi
	{
		tbGenerateProjectRouter.POST("createTbGenerateProject", tbGenerateProjectApi.CreateTbGenerateProject)   // 新建
		tbGenerateProjectRouter.DELETE("deleteTbGenerateProject", tbGenerateProjectApi.DeleteTbGenerateProject) // 删除
		tbGenerateProjectRouter.PUT("updateTbGenerateProject", tbGenerateProjectApi.UpdateTbGenerateProject)    // 更新
		tbGenerateProjectRouter.GET("getTbGenerateProject", tbGenerateProjectApi.GetTbGenerateProject)          // 根据ID获取
		tbGenerateProjectRouter.GET("getTbGenerateProjectList", tbGenerateProjectApi.GetTbGenerateProjectList)  // 获取列表
		tbGenerateProjectRouter.GET("copy", tbGenerateProjectApi.CopyProject)                                   // 克隆
	}
}
