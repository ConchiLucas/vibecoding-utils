package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbAgileTableSampleApi struct{}

func (a *TbAgileTableSampleApi) List(c *gin.Context) {
	var scope systemReq.AgileTableSampleScope
	if err := c.ShouldBindQuery(&scope); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := tbAgileTableSampleService.List(scope, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("获取敏捷表样本配置失败", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithData(list, c)
}

func (a *TbAgileTableSampleApi) Save(c *gin.Context) {
	var req systemReq.AgileTableSampleSave
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := tbAgileTableSampleService.Save(req, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("保存敏捷表样本配置失败", zap.Error(err))
		response.FailWithMessage("保存失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(list, "保存成功", c)
}

func (a *TbAgileTableSampleApi) History(c *gin.Context) {
	var scope systemReq.AgileTableSampleScope
	if err := c.ShouldBindQuery(&scope); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := tbAgileTableSampleService.History(scope, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("获取敏捷表样本业务方案失败", zap.Error(err))
		response.FailWithMessage("获取业务方案失败", c)
		return
	}
	response.OkWithData(list, c)
}
