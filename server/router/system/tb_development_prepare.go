package system

import "github.com/gin-gonic/gin"

type TbDevelopmentPrepareRouter struct{}

func (r *TbDevelopmentPrepareRouter) InitTbDevelopmentPrepareRouter(Router *gin.RouterGroup) {
	prepareRouter := Router.Group("developmentPrepare")
	{
		prepareRouter.POST("page", tbDevelopmentPrepareApi.GetList)
		prepareRouter.GET("detail/:id", tbDevelopmentPrepareApi.GetByID)
		prepareRouter.POST("saveOrUpdate", tbDevelopmentPrepareApi.SaveOrUpdate)
		prepareRouter.DELETE("delete/:ids", tbDevelopmentPrepareApi.Delete)
	}
}
