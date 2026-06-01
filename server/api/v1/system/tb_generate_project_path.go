package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGenerateProjectPathApi struct{}

func (a *TbGenerateProjectPathApi) CreateTbGenerateProjectPath(c *gin.Context) {
	var req system.TbGenerateProjectPath
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectPathService.CreateTbGenerateProjectPath(&req)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbGenerateProjectPathApi) DeleteTbGenerateProjectPath(c *gin.Context) {
	var req system.TbGenerateProjectPath
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectPathService.DeleteTbGenerateProjectPath(req)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateProjectPathApi) UpdateTbGenerateProjectPath(c *gin.Context) {
	var req system.TbGenerateProjectPath
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectPathService.UpdateTbGenerateProjectPath(&req)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateProjectPathApi) GetTbGenerateProjectPath(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGenerateProjectPathService.GetTbGenerateProjectPath(id)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectPathApi) GetTbGenerateProjectPathList(c *gin.Context) {
	res, err := tbGenerateProjectPathService.GetTbGenerateProjectPathList()
	if err != nil {
		global.GVA_LOG.Error("查询列表失败!", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectPathApi) UpdateEnabled(c *gin.Context) {
	var req system.TbGenerateProjectPath
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectPathService.UpdateEnabled(req.ID, req.Enabled); err != nil {
		global.GVA_LOG.Error("更新启用状态失败!", zap.Error(err))
		response.FailWithMessage("更新启用状态失败", c)
	} else {
		response.OkWithMessage("更改状态成功", c)
	}
}
