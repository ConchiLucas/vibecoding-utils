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

type TbGenerateProjectInstanceApi struct{}

func (a *TbGenerateProjectInstanceApi) CreateTbGenerateProjectInstance(c *gin.Context) {
	var req system.TbGenerateProjectInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectInstanceService.CreateTbGenerateProjectInstance(&req); err != nil {
		global.GVA_LOG.Error("创建项目失败!", zap.Error(err))
		response.FailWithMessage("创建项目失败", c)
	} else {
		response.OkWithData(req, c)
	}
}

func (a *TbGenerateProjectInstanceApi) DeleteTbGenerateProjectInstance(c *gin.Context) {
	var req system.TbGenerateProjectInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectInstanceService.DeleteTbGenerateProjectInstance(req); err != nil {
		global.GVA_LOG.Error("删除项目失败!", zap.Error(err))
		response.FailWithMessage("删除项目失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateProjectInstanceApi) UpdateTbGenerateProjectInstance(c *gin.Context) {
	var req system.TbGenerateProjectInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectInstanceService.UpdateTbGenerateProjectInstance(&req); err != nil {
		global.GVA_LOG.Error("更新项目失败!", zap.Error(err))
		response.FailWithMessage("更新项目失败", c)
	} else {
		response.OkWithData(req, c)
	}
}

func (a *TbGenerateProjectInstanceApi) UpdateSelectedPathSet(c *gin.Context) {
	var req systemReq.UpdateSelectedPathSetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectInstanceService.UpdateSelectedPathSet(req.ProjectInstanceId, req.SelectedPathSetIdentity); err != nil {
		global.GVA_LOG.Error("更新路径配置选中状态失败!", zap.Error(err))
		response.FailWithMessage("更新路径配置选中状态失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateProjectInstanceApi) GetTbGenerateProjectInstance(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGenerateProjectInstanceService.GetTbGenerateProjectInstance(id)
	if err != nil {
		global.GVA_LOG.Error("查询项目失败!", zap.Error(err))
		response.FailWithMessage("查询项目失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectInstanceApi) GetTbGenerateProjectInstanceList(c *gin.Context) {
	templateProjectId := 0
	if v := c.Query("templateProjectId"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			templateProjectId = parsed
		}
	}
	ensureDefault := c.Query("ensureDefault") == "1" || c.Query("ensureDefault") == "true"
	res, err := tbGenerateProjectInstanceService.GetTbGenerateProjectInstanceList(templateProjectId, ensureDefault)
	if err != nil {
		global.GVA_LOG.Error("查询项目列表失败!", zap.Error(err))
		response.FailWithMessage("查询项目列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}
