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

type TbInterfaceParamsApi struct{}

func (a *TbInterfaceParamsApi) CreateTbInterfaceParams(c *gin.Context) {
	var params system.TbInterfaceParams
	err := c.ShouldBindJSON(&params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceParamsService.CreateTbInterfaceParams(params)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbInterfaceParamsApi) DeleteTbInterfaceParams(c *gin.Context) {
	var params system.TbInterfaceParams
	err := c.ShouldBindJSON(&params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceParamsService.DeleteTbInterfaceParams(params)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbInterfaceParamsApi) DeleteTbInterfaceParamsByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceParamsService.DeleteTbInterfaceParamsByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbInterfaceParamsApi) UpdateTbInterfaceParams(c *gin.Context) {
	var params system.TbInterfaceParams
	err := c.ShouldBindJSON(&params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceParamsService.UpdateTbInterfaceParams(&params)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbInterfaceParamsApi) GetTbInterfaceParams(c *gin.Context) {
	var params system.TbInterfaceParams
	err := c.ShouldBindQuery(&params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(params, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	params, err = tbInterfaceParamsService.GetTbInterfaceParams(params.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"interfaceParams": params}, "查询成功", c)
	}
}

func (a *TbInterfaceParamsApi) GetTbInterfaceParamsList(c *gin.Context) {
	var pageInfo systemReq.InterfaceParamsSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbInterfaceParamsService.GetTbInterfaceParamsInfoList(pageInfo)
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

// GetParamsEntity returns the last-used request/response params for an interface,
// or builds an empty JSON skeleton from its column schema if no history exists.
func (a *TbInterfaceParamsApi) GetParamsEntity(c *gin.Context) {
	var req struct {
		ID    uint   `json:"id"`
		Paths string `json:"paths"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	result, err := tbInterfaceParamsService.GetParamsEntity(req.ID, req.Paths, userName)
	if err != nil {
		global.GVA_LOG.Error("获取接口参数失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithData(result, c)
}

