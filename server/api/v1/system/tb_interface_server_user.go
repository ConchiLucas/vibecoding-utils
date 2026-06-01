package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbInterfaceServerUserApi struct{}

func (api *TbInterfaceServerUserApi) CreateTbInterfaceServerUser(c *gin.Context) {
	var data system.TbInterfaceServerUser
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceServerUserService.CreateTbInterfaceServerUser(data); err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

func (api *TbInterfaceServerUserApi) DeleteTbInterfaceServerUser(c *gin.Context) {
	var data system.TbInterfaceServerUser
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceServerUserService.DeleteTbInterfaceServerUser(data.ID); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (api *TbInterfaceServerUserApi) UpdateTbInterfaceServerUser(c *gin.Context) {
	var data system.TbInterfaceServerUser
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceServerUserService.UpdateTbInterfaceServerUser(data); err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

func (api *TbInterfaceServerUserApi) GetTbInterfaceServerUserList(c *gin.Context) {
	var pageInfo systemReq.ServerUserSearch
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbInterfaceServerUserService.GetTbInterfaceServerUserList(pageInfo)
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

// UpdateClientStatus toggles the enable_flag of a test user (1=enabled, 0=disabled).
func (api *TbInterfaceServerUserApi) UpdateClientStatus(c *gin.Context) {
	var data struct {
		ID         uint `json:"ID"`
		EnableFlag int  `json:"enableFlag"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbInterfaceServerUserService.UpdateClientStatus(data.ID, data.EnableFlag); err != nil {
		global.GVA_LOG.Error("更改状态失败!", zap.Error(err))
		response.FailWithMessage("更改状态失败", c)
		return
	}
	response.OkWithMessage("更改状态成功", c)
}

