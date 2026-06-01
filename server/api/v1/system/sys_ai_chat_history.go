package system

import (
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/service/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AIChatHistoryApi struct{}

func (a *AIChatHistoryApi) SaveOrUpdate(c *gin.Context) {
	var req system.SaveAIChatHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	item, err := aiChatHistoryService.SaveOrUpdateChatHistory(req, utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("保存 AI 对话历史失败", zap.Error(err))
		response.FailWithMessage("保存失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(item, "保存成功", c)
}

func (a *AIChatHistoryApi) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	list, total, err := aiChatHistoryService.GetChatHistoryList(utils.GetUserID(c), limit)
	if err != nil {
		global.GVA_LOG.Error("获取 AI 对话历史失败", zap.Error(err))
		response.FailWithMessage("获取失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total}, "获取成功", c)
}

func (a *AIChatHistoryApi) Detail(c *gin.Context) {
	item, err := aiChatHistoryService.GetChatHistoryByChatID(utils.GetUserID(c), c.Param("chatId"))
	if err != nil {
		global.GVA_LOG.Error("获取 AI 对话历史详情失败", zap.Error(err))
		response.FailWithMessage("获取失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(item, "获取成功", c)
}
