package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	systemSvc "github.com/flipped-aurora/easy-deploy/server/service/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbInterfaceApi struct{}

func (a *TbInterfaceApi) CreateTbInterface(c *gin.Context) {
	var iface system.TbInterface
	err := c.ShouldBindJSON(&iface)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceService.CreateTbInterface(iface)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbInterfaceApi) DeleteTbInterface(c *gin.Context) {
	var iface system.TbInterface
	err := c.ShouldBindJSON(&iface)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceService.DeleteTbInterface(iface)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbInterfaceApi) DeleteTbInterfaceByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceService.DeleteTbInterfaceByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbInterfaceApi) UpdateTbInterface(c *gin.Context) {
	var iface system.TbInterface
	err := c.ShouldBindJSON(&iface)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceService.UpdateTbInterface(&iface)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbInterfaceApi) GetTbInterface(c *gin.Context) {
	var iface system.TbInterface
	err := c.ShouldBindQuery(&iface)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(iface, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	iface, err = tbInterfaceService.GetTbInterface(iface.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"interface": iface}, "查询成功", c)
	}
}

func (a *TbInterfaceApi) GetTbInterfaceList(c *gin.Context) {
	var pageInfo systemReq.InterfaceSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbInterfaceService.GetTbInterfaceInfoList(pageInfo)
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

// ForwardInterface executes a proxied HTTP call to the target service.
// Auth is provided via the selected user's requestHeader (no auto-login).
func (a *TbInterfaceApi) ForwardInterface(c *gin.Context) {
	var req systemSvc.ForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	req.UserName = utils.GetUserName(c)
	result, err := tbInterfaceService.ForwardInterface(req)
	if err != nil {
		global.GVA_LOG.Error("转发请求失败!", zap.Error(err))
		response.FailWithMessage("转发失败: "+err.Error(), c)
		return
	}
	response.OkWithData(result, c)
}

