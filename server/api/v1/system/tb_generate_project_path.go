package system

import (
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
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

func (a *TbGenerateProjectPathApi) DeletePathSet(c *gin.Context) {
	var req systemReq.DeleteGenerateProjectPathSetReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	deletedCount, err := tbGenerateProjectPathService.DeletePathSet(req)
	if err != nil {
		global.GVA_LOG.Error("删除路径配置失败!", zap.Error(err))
		response.FailWithMessage("删除路径配置失败", c)
	} else {
		response.OkWithData(gin.H{"deletedCount": deletedCount}, c)
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

func (a *TbGenerateProjectPathApi) RenamePathSet(c *gin.Context) {
	var req systemReq.RenameGenerateProjectPathSetReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	updatedCount, err := tbGenerateProjectPathService.RenamePathSet(req)
	if err != nil {
		global.GVA_LOG.Error("重命名路径配置失败!", zap.Error(err))
		response.FailWithMessage("重命名路径配置失败", c)
	} else {
		response.OkWithData(gin.H{"updatedCount": updatedCount}, c)
	}
}

func (a *TbGenerateProjectPathApi) CopyPathSet(c *gin.Context) {
	var req systemReq.CopyGenerateProjectPathSetReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	nextPathSet, err := tbGenerateProjectPathService.CopyPathSet(req)
	if err != nil {
		global.GVA_LOG.Error("复制路径配置失败!", zap.Error(err))
		response.FailWithMessage("复制路径配置失败", c)
	} else {
		response.OkWithData(gin.H{"pathSet": nextPathSet}, c)
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
	projectId := 0
	if v := c.Query("projectId"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			projectId = parsed
		}
	}
	projectInstanceId := 0
	if v := c.Query("projectInstanceId"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			projectInstanceId = parsed
		}
	}
	res, err := tbGenerateProjectPathService.GetTbGenerateProjectPathList(projectId, projectInstanceId)
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
