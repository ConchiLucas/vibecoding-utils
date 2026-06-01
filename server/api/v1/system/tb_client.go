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

type TbClientApi struct{}

func (a *TbClientApi) CreateTbClient(c *gin.Context) {
	var client system.TbClient
	err := c.ShouldBindJSON(&client)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbClientService.CreateTbClient(client)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbClientApi) DeleteTbClient(c *gin.Context) {
	var client system.TbClient
	err := c.ShouldBindJSON(&client)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbClientService.DeleteTbClient(client)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbClientApi) DeleteTbClientByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbClientService.DeleteTbClientByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbClientApi) UpdateTbClient(c *gin.Context) {
	var client system.TbClient
	err := c.ShouldBindJSON(&client)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbClientService.UpdateTbClient(&client)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbClientApi) GetTbClient(c *gin.Context) {
	var client system.TbClient
	err := c.ShouldBindQuery(&client)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(client, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	client, err = tbClientService.GetTbClient(client.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"client": client}, "查询成功", c)
	}
}

func (a *TbClientApi) GetTbClientList(c *gin.Context) {
	var pageInfo systemReq.ClientSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbClientService.GetTbClientInfoList(pageInfo)
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
