package system

import (
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TbGenerateFieldSnippetApi struct{}

func (a *TbGenerateFieldSnippetApi) GetLatestGenerateFieldSnippet(c *gin.Context) {
	businessType := strings.TrimSpace(c.Query("businessType"))
	res, err := tbGenerateFieldSnippetService.GetLatestGenerateFieldSnippet(businessType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.OkWithData(nil, c)
			return
		}
		global.GVA_LOG.Error("查询最新字段片段失败!", zap.Error(err))
		response.FailWithMessage("查询最新字段片段失败", c)
		return
	}
	response.OkWithData(res, c)
}

func (a *TbGenerateFieldSnippetApi) GetGenerateFieldSnippetHistory(c *gin.Context) {
	businessType := strings.TrimSpace(c.Query("businessType"))
	res, err := tbGenerateFieldSnippetService.GetGenerateFieldSnippetHistory(businessType)
	if err != nil {
		global.GVA_LOG.Error("查询字段片段历史失败!", zap.Error(err))
		response.FailWithMessage("查询字段片段历史失败", c)
		return
	}
	response.OkWithData(res, c)
}

func (a *TbGenerateFieldSnippetApi) PreviewGenerateFieldSnippet(c *gin.Context) {
	var req systemReq.PreviewGenerateFieldSnippetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	res, err := tbGenerateFieldSnippetService.PreviewGenerateFieldSnippet(req)
	if err != nil {
		global.GVA_LOG.Error("预览字段片段失败!", zap.Error(err))
		response.FailWithMessage("预览字段片段失败", c)
		return
	}
	response.OkWithData(res, c)
}

func (a *TbGenerateFieldSnippetApi) SaveGenerateFieldSnippet(c *gin.Context) {
	var req systemReq.SaveGenerateFieldSnippetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	res, err := tbGenerateFieldSnippetService.SaveGenerateFieldSnippet(req)
	if err != nil {
		global.GVA_LOG.Error("保存字段片段失败!", zap.Error(err))
		response.FailWithMessage("保存字段片段失败", c)
		return
	}
	response.OkWithDetailed(res, "保存成功", c)
}
