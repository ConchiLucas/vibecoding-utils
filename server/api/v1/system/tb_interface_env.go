package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbInterfaceEnvApi struct{}

func (api *TbInterfaceEnvApi) CreateTbInterfaceEnv(c *gin.Context) {
	var data system.TbInterfaceEnv
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceEnvService.CreateTbInterfaceEnv(data); err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

func (api *TbInterfaceEnvApi) DeleteTbInterfaceEnv(c *gin.Context) {
	var data system.TbInterfaceEnv
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceEnvService.DeleteTbInterfaceEnv(data.ID); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (api *TbInterfaceEnvApi) UpdateTbInterfaceEnv(c *gin.Context) {
	var data system.TbInterfaceEnv
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceEnvService.UpdateTbInterfaceEnv(data); err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

func (api *TbInterfaceEnvApi) GetTbInterfaceEnvList(c *gin.Context) {
	var pageInfo systemReq.InterfaceEnvSearch
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbInterfaceEnvService.GetTbInterfaceEnvList(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}
