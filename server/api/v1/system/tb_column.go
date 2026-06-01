package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbColumnApi struct{}

func (a *TbColumnApi) CreateTbColumn(c *gin.Context) {
	var col system.TbColumn
	err := c.ShouldBindJSON(&col)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbColumnService.CreateTbColumn(col)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbColumnApi) DeleteTbColumn(c *gin.Context) {
	var col system.TbColumn
	err := c.ShouldBindJSON(&col)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbColumnService.DeleteTbColumn(col)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbColumnApi) DeleteTbColumnByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbColumnService.DeleteTbColumnByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbColumnApi) UpdateTbColumn(c *gin.Context) {
	var col system.TbColumn
	err := c.ShouldBindJSON(&col)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbColumnService.UpdateTbColumn(&col)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbColumnApi) GetTbColumn(c *gin.Context) {
	var col system.TbColumn
	err := c.ShouldBindQuery(&col)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(col, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	col, err = tbColumnService.GetTbColumn(col.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"column": col}, "查询成功", c)
	}
}

func (a *TbColumnApi) GetTbColumnList(c *gin.Context) {
	var pageInfo systemReq.ColumnSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbColumnService.GetTbColumnInfoList(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
	} else {
		response.OkWithDetailed(response.PageResult{
			List:     list,
			Total:    total,
			Page:     pageInfo.Page,
			PageSize: pageInfo.PageSize,
		}, "获取成功", c)
	}
}

func (a *TbColumnApi) GetColumnTree(c *gin.Context) {
	var qo systemReq.InterfaceTreeQO
	if err := c.ShouldBindJSON(&qo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	treeList, err := tbColumnService.GetColumnTree(qo)
	if err != nil {
		global.GVA_LOG.Error("获取参数树失败!", zap.Error(err))
		response.FailWithMessage("获取参数树失败", c)
		return
	}

	response.OkWithDetailed(treeList, "获取成功", c)
}
