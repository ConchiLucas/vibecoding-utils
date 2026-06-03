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

type TbTableRelateApi struct{}

func (a *TbTableRelateApi) CreateTbTableRelate(c *gin.Context) {
	var tr system.TbTableRelate
	err := c.ShouldBindJSON(&tr)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableRelateService.CreateTbTableRelate(tr)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbTableRelateApi) DeleteTbTableRelate(c *gin.Context) {
	var tr system.TbTableRelate
	err := c.ShouldBindJSON(&tr)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableRelateService.DeleteTbTableRelate(tr)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbTableRelateApi) DeleteTbTableRelateByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableRelateService.DeleteTbTableRelateByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbTableRelateApi) UpdateTbTableRelate(c *gin.Context) {
	var tr system.TbTableRelate
	err := c.ShouldBindJSON(&tr)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbTableRelateService.UpdateTbTableRelate(&tr)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbTableRelateApi) GetTbTableRelate(c *gin.Context) {
	var tr system.TbTableRelate
	err := c.ShouldBindQuery(&tr)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(tr, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	tr, err = tbTableRelateService.GetTbTableRelate(tr.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"tableRelate": tr}, "查询成功", c)
	}
}

func (a *TbTableRelateApi) GetTbTableRelateList(c *gin.Context) {
	var pageInfo systemReq.TableRelateSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbTableRelateService.GetTbTableRelateInfoList(pageInfo)
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

func (a *TbTableRelateApi) ImportTableRelations(c *gin.Context) {
	var req systemReq.ImportTableRelationsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	result, err := tbTableRelateService.ImportTableRelations(req, userName)
	if err != nil {
		global.GVA_LOG.Error("AI导入表字段关联关系失败!", zap.Error(err))
		response.FailWithMessage("AI导入表字段关联关系失败: "+err.Error(), c)
	} else {
		response.OkWithDetailed(result, "导入成功", c)
	}
}

func (a *TbTableRelateApi) GetClientData(c *gin.Context) {
	var query systemReq.ClientQueryModel
	err := c.ShouldBindJSON(&query)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	query.UserName = userName
	list, err := tbTableRelateService.GetClientData(query)
	if err != nil {
		global.GVA_LOG.Error("客户端数据查询失败!", zap.Error(err))
		response.FailWithMessage("客户端数据查询失败", c)
	} else {
		response.OkWithData(list, c)
	}
}

func (a *TbTableRelateApi) GetRemoteColumns(c *gin.Context) {
	var query systemReq.ClientQueryModel
	err := c.ShouldBindJSON(&query)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	list, err := tbTableRelateService.GetRemoteTableColumns(query)
	if err != nil {
		global.GVA_LOG.Error("获取外部表级字段失败!", zap.Error(err))
		response.FailWithMessage("获取外部表级字段失败", c)
	} else {
		response.OkWithData(list, c)
	}
}

// GetTableComments returns table annotations/comments for a list of "db:table" pairs.
func (a *TbTableRelateApi) GetTableComments(c *gin.Context) {
	var req struct {
		ProjectConfigID uint     `json:"projectConfigId"`
		Environment     string   `json:"environment"`
		ConnectionID    uint     `json:"connectionId"`
		Tables          []string `json:"tables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.Environment == "" || len(req.Tables) == 0 {
		response.OkWithData(map[string]string{}, c)
		return
	}
	comments := tbTableRelateService.GetTableComments(req.ProjectConfigID, req.Environment, req.ConnectionID, req.Tables)
	response.OkWithData(comments, c)
}
