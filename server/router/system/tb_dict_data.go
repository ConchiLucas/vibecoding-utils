package system

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1"
	"github.com/gin-gonic/gin"
)

type TbDictDataRouter struct{}

// InitTbDictDataRouter 初始化 字典数据 路由信息
func (s *TbDictDataRouter) InitTbDictDataRouter(Router *gin.RouterGroup) {
	dictDataRouter := Router.Group("dict")
	dictDataRouterWithoutRecord := Router.Group("dict")
	var dictDataApi = v1.ApiGroupApp.SystemApiGroup.TbDictDataApi
	{
		dictDataRouter.POST("createDictData", dictDataApi.CreateDictData)             // 新建字典数据
		dictDataRouter.DELETE("deleteDictData", dictDataApi.DeleteDictData)           // 删除字典数据
		dictDataRouter.DELETE("deleteDictDataByIds", dictDataApi.DeleteDictDataByIds) // 批量删除字典数据
		dictDataRouter.PUT("updateDictData", dictDataApi.UpdateDictData)              // 更新字典数据
	}
	{
		dictDataRouterWithoutRecord.GET("getDictData", dictDataApi.GetDictData)         // 根据ID获取字典数据
		dictDataRouterWithoutRecord.GET("getDictDataList", dictDataApi.GetDictDataList) // 获取字典数据列表
	}
}
