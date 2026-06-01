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

type TbTableApi struct{}

func (a *TbTableApi) CreateTbTable(c *gin.Context) {
	var table system.TbTable
	err := c.ShouldBindJSON(&table)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableService.CreateTbTable(table)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbTableApi) DeleteTbTable(c *gin.Context) {
	var table system.TbTable
	err := c.ShouldBindJSON(&table)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableService.DeleteTbTable(table)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbTableApi) DeleteTbTableByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableService.DeleteTbTableByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbTableApi) UpdateTbTable(c *gin.Context) {
	var table system.TbTable
	err := c.ShouldBindJSON(&table)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableService.UpdateTbTable(&table)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbTableApi) GetTbTable(c *gin.Context) {
	var table system.TbTable
	err := c.ShouldBindQuery(&table)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(table, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	table, err = tbTableService.GetTbTable(table.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"table": table}, "查询成功", c)
	}
}

func (a *TbTableApi) GetTbTableList(c *gin.Context) {
	var pageInfo systemReq.TableSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbTableService.GetTbTableInfoList(pageInfo)
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

func (a *TbTableApi) TableFuzzyQuery(c *gin.Context) {
	var table system.TbTable
	err := c.ShouldBindJSON(&table)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	list, err := tbTableService.TableFuzzyQuery(table, userName)
	if err != nil {
		global.GVA_LOG.Error("模糊查询失败!", zap.Error(err))
		response.FailWithMessage("模糊查询失败", c)
	} else {
		response.OkWithData(list, c)
	}
}
