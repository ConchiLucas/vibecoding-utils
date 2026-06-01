package system

import (
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbAgileRequestApi struct{}

func (a *TbAgileRequestApi) Send(c *gin.Context) {
	var req systemReq.AgileRequestSend
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := tbAgileRequestService.Send(req, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("敏捷请求执行失败", zap.Error(err))
		response.FailWithDetailed(result, err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "请求完成", c)
}

func (a *TbAgileRequestApi) GetList(c *gin.Context) {
	var pageInfo systemReq.AgileRequestSearch
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbAgileRequestService.GetList(pageInfo, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("获取敏捷请求历史失败", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

func (a *TbAgileRequestApi) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailWithMessage("id 参数不正确", c)
		return
	}
	result, err := tbAgileRequestService.GetByID(uint(id), utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("获取敏捷请求详情失败", zap.Error(err))
		response.FailWithMessage("查询失败", c)
		return
	}
	response.OkWithDetailed(result, "查询成功", c)
}

func (a *TbAgileRequestApi) DeleteByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailWithMessage("id 参数不正确", c)
		return
	}
	if err := tbAgileRequestService.DeleteByID(uint(id), utils.GetUserName(c)); err != nil {
		global.GVA_LOG.Error("删除敏捷请求历史失败", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *TbAgileRequestApi) Clear(c *gin.Context) {
	if err := tbAgileRequestService.Clear(utils.GetUserName(c)); err != nil {
		global.GVA_LOG.Error("清空敏捷请求历史失败", zap.Error(err))
		response.FailWithMessage("清空失败", c)
		return
	}
	response.OkWithMessage("清空成功", c)
}
