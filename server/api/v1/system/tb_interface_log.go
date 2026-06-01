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

type TbInterfaceLogApi struct{}

func (a *TbInterfaceLogApi) CreateTbInterfaceLog(c *gin.Context) {
	var log system.TbInterfaceLog
	err := c.ShouldBindJSON(&log)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceLogService.CreateTbInterfaceLog(log)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbInterfaceLogApi) DeleteTbInterfaceLog(c *gin.Context) {
	var log system.TbInterfaceLog
	err := c.ShouldBindJSON(&log)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceLogService.DeleteTbInterfaceLog(log)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbInterfaceLogApi) DeleteTbInterfaceLogByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceLogService.DeleteTbInterfaceLogByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbInterfaceLogApi) UpdateTbInterfaceLog(c *gin.Context) {
	var log system.TbInterfaceLog
	err := c.ShouldBindJSON(&log)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceLogService.UpdateTbInterfaceLog(&log)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbInterfaceLogApi) GetTbInterfaceLog(c *gin.Context) {
	var log system.TbInterfaceLog
	err := c.ShouldBindQuery(&log)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(log, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	log, err = tbInterfaceLogService.GetTbInterfaceLog(log.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"interfaceLog": log}, "查询成功", c)
	}
}

func (a *TbInterfaceLogApi) GetTbInterfaceLogList(c *gin.Context) {
	var pageInfo systemReq.InterfaceLogSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbInterfaceLogService.GetTbInterfaceLogInfoList(pageInfo)
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

// GetParamsPreview returns the req/res params of a specific log entry
// for display in the detail popup (mirrors Python log/getParamsPreview/{id}).
func (a *TbInterfaceLogApi) GetParamsPreview(c *gin.Context) {
	var log system.TbInterfaceLog
	if err := c.ShouldBindQuery(&log); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := utils.Verify(log, utils.IdVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	detail, err := tbInterfaceLogService.GetTbInterfaceLog(log.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
		return
	}
	type ParamsItem struct {
		ParamsName string `json:"paramsName"`
		Content    string `json:"content"`
	}
	var result []ParamsItem
	if detail.ReqParams != "" {
		result = append(result, ParamsItem{ParamsName: "请求参数", Content: detail.ReqParams})
	}
	if detail.ResParams != "" {
		result = append(result, ParamsItem{ParamsName: "返回参数", Content: detail.ResParams})
	}
	response.OkWithData(result, c)
}

