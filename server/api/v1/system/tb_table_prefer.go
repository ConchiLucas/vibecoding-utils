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

type TbTablePreferApi struct{}

func (a *TbTablePreferApi) CreateTbTablePrefer(c *gin.Context) {
	var prefer system.TbTablePrefer
	err := c.ShouldBindJSON(&prefer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTablePreferService.CreateTbTablePrefer(prefer)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbTablePreferApi) DeleteTbTablePrefer(c *gin.Context) {
	var prefer system.TbTablePrefer
	err := c.ShouldBindJSON(&prefer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTablePreferService.DeleteTbTablePrefer(prefer)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbTablePreferApi) DeleteTbTablePreferByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTablePreferService.DeleteTbTablePreferByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbTablePreferApi) UpdateTbTablePrefer(c *gin.Context) {
	var prefer system.TbTablePrefer
	err := c.ShouldBindJSON(&prefer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTablePreferService.UpdateTbTablePrefer(&prefer)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbTablePreferApi) GetTbTablePrefer(c *gin.Context) {
	var prefer system.TbTablePrefer
	err := c.ShouldBindQuery(&prefer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(prefer, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	prefer, err = tbTablePreferService.GetTbTablePrefer(prefer.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"tablePrefer": prefer}, "查询成功", c)
	}
}

func (a *TbTablePreferApi) GetTbTablePreferList(c *gin.Context) {
	var pageInfo systemReq.TablePreferSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbTablePreferService.GetTbTablePreferInfoList(pageInfo)
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

type TablePreferByParamsModel struct {
	DatabaseStr     string `json:"databaseStr"`
	ProjectConfigID uint   `json:"projectConfigId"`
	ConnectionID    uint   `json:"connectionId"`
}

func (a *TbTablePreferApi) GetPreferVOByParams(c *gin.Context) {
	var params TablePreferByParamsModel
	err := c.ShouldBindJSON(&params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	prefer, err := tbTablePreferService.GetPreferVOByParams(params.DatabaseStr, userName, params.ProjectConfigID, params.ConnectionID)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
	} else {
		response.OkWithData(prefer, c)
	}
}

func (a *TbTablePreferApi) GetPreferColumnValueList(c *gin.Context) {
	var params TablePreferByParamsModel
	err := c.ShouldBindJSON(&params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	list, err := tbTablePreferService.GetPreferColumnValueList(params.DatabaseStr, userName, params.ProjectConfigID, params.ConnectionID)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
	} else {
		response.OkWithData(list, c)
	}
}

func (a *TbTablePreferApi) GetHistoryTableNames(c *gin.Context) {
	userName := utils.GetUserName(c)
	var params struct {
		ProjectConfigID uint `form:"projectConfigId"`
		ConnectionID    uint `form:"connectionId"`
	}
	if err := c.ShouldBindQuery(&params); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := tbTablePreferService.GetHistoryTableNames(userName, params.ProjectConfigID, params.ConnectionID)
	if err != nil {
		global.GVA_LOG.Error("获取历史表名失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
	} else {
		response.OkWithData(list, c)
	}
}
