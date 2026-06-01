package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGenerateProjectPlaceHolderApi struct{}

func (a *TbGenerateProjectPlaceHolderApi) CreateTbGenerateProjectPlaceHolder(c *gin.Context) {
	var req system.TbGenerateProjectPlaceHolder
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectPlaceHolderService.CreateTbGenerateProjectPlaceHolder(&req)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbGenerateProjectPlaceHolderApi) DeleteTbGenerateProjectPlaceHolder(c *gin.Context) {
	var req system.TbGenerateProjectPlaceHolder
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectPlaceHolderService.DeleteTbGenerateProjectPlaceHolder(req)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateProjectPlaceHolderApi) UpdateTbGenerateProjectPlaceHolder(c *gin.Context) {
	var req system.TbGenerateProjectPlaceHolder
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectPlaceHolderService.UpdateTbGenerateProjectPlaceHolder(&req)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateProjectPlaceHolderApi) GetTbGenerateProjectPlaceHolder(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGenerateProjectPlaceHolderService.GetTbGenerateProjectPlaceHolder(id)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectPlaceHolderApi) GetTbGenerateProjectPlaceHolderList(c *gin.Context) {
	res, err := tbGenerateProjectPlaceHolderService.GetTbGenerateProjectPlaceHolderList()
	if err != nil {
		global.GVA_LOG.Error("查询列表失败!", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}
