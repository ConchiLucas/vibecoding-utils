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

type TbTableColumnApi struct{}

func (a *TbTableColumnApi) CreateTbTableColumn(c *gin.Context) {
	var tc system.TbTableColumn
	err := c.ShouldBindJSON(&tc)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableColumnService.CreateTbTableColumn(tc)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbTableColumnApi) DeleteTbTableColumn(c *gin.Context) {
	var tc system.TbTableColumn
	err := c.ShouldBindJSON(&tc)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableColumnService.DeleteTbTableColumn(tc)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbTableColumnApi) DeleteTbTableColumnByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableColumnService.DeleteTbTableColumnByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbTableColumnApi) UpdateTbTableColumn(c *gin.Context) {
	var tc system.TbTableColumn
	err := c.ShouldBindJSON(&tc)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableColumnService.UpdateTbTableColumn(&tc)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbTableColumnApi) GetTbTableColumn(c *gin.Context) {
	var tc system.TbTableColumn
	err := c.ShouldBindQuery(&tc)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(tc, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	tc, err = tbTableColumnService.GetTbTableColumn(tc.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"tableColumn": tc}, "查询成功", c)
	}
}

func (a *TbTableColumnApi) GetTbTableColumnList(c *gin.Context) {
	var pageInfo systemReq.TableColumnSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbTableColumnService.GetTbTableColumnInfoList(pageInfo)
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
