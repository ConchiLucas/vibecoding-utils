package system

import (
	"fmt"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGenerateProjectApi struct{}

func (a *TbGenerateProjectApi) CreateTbGenerateProject(c *gin.Context) {
	var req system.TbGenerateProject
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectService.CreateTbGenerateProject(&req)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbGenerateProjectApi) DeleteTbGenerateProject(c *gin.Context) {
	var req system.TbGenerateProject
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectService.DeleteTbGenerateProject(req)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateProjectApi) UpdateTbGenerateProject(c *gin.Context) {
	var req system.TbGenerateProject
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectService.UpdateTbGenerateProject(&req)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateProjectApi) GetTbGenerateProject(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGenerateProjectService.GetTbGenerateProject(id)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectApi) GetTbGenerateProjectList(c *gin.Context) {
	projectConfigId := 0
	if v := c.Query("projectConfigId"); v != "" {
		fmt.Sscanf(v, "%d", &projectConfigId)
	}
	res, err := tbGenerateProjectService.GetTbGenerateProjectList(projectConfigId)
	if err != nil {
		global.GVA_LOG.Error("查询列表失败!", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectApi) CopyProject(c *gin.Context) {
	id := c.Query("id")
	if err := tbGenerateProjectService.CopyProject(id); err != nil {
		global.GVA_LOG.Error("克隆失败", zap.Error(err))
		response.FailWithMessage("克隆失败", c)
	} else {
		response.OkWithMessage("克隆成功", c)
	}
}
