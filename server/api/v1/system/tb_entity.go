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

type TbEntityApi struct{}

func (a *TbEntityApi) CreateTbEntity(c *gin.Context) {
	var entity system.TbEntity
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbEntityService.CreateTbEntity(entity)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbEntityApi) DeleteTbEntity(c *gin.Context) {
	var entity system.TbEntity
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbEntityService.DeleteTbEntity(entity)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbEntityApi) DeleteTbEntityByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbEntityService.DeleteTbEntityByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbEntityApi) UpdateTbEntity(c *gin.Context) {
	var entity system.TbEntity
	err := c.ShouldBindJSON(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbEntityService.UpdateTbEntity(&entity)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbEntityApi) GetTbEntity(c *gin.Context) {
	var entity system.TbEntity
	err := c.ShouldBindQuery(&entity)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(entity, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	entity, err = tbEntityService.GetTbEntity(entity.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"entity": entity}, "查询成功", c)
	}
}

func (a *TbEntityApi) GetTbEntityList(c *gin.Context) {
	var pageInfo systemReq.EntitySearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbEntityService.GetTbEntityInfoList(pageInfo)
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
