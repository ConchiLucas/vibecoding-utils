package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGeneratePlaceHolderApi struct{}

func (a *TbGeneratePlaceHolderApi) CreateTbGeneratePlaceHolder(c *gin.Context) {
	var req system.TbGeneratePlaceHolder
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGeneratePlaceHolderService.CreateTbGeneratePlaceHolder(&req)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbGeneratePlaceHolderApi) DeleteTbGeneratePlaceHolder(c *gin.Context) {
	var req system.TbGeneratePlaceHolder
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGeneratePlaceHolderService.DeleteTbGeneratePlaceHolder(req)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGeneratePlaceHolderApi) UpdateTbGeneratePlaceHolder(c *gin.Context) {
	var req system.TbGeneratePlaceHolder
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGeneratePlaceHolderService.UpdateTbGeneratePlaceHolder(&req)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGeneratePlaceHolderApi) GetTbGeneratePlaceHolder(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGeneratePlaceHolderService.GetTbGeneratePlaceHolder(id)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGeneratePlaceHolderApi) GetTbGeneratePlaceHolderList(c *gin.Context) {
	res, err := tbGeneratePlaceHolderService.GetTbGeneratePlaceHolderList()
	if err != nil {
		global.GVA_LOG.Error("查询列表失败!", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}
