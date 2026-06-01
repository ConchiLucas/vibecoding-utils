package system

import (
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ServerApi struct{}

// GetServerPage 分页获取服务器列表
func (a *ServerApi) GetServerPage(c *gin.Context) {
	var req request.TbServerSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := serverService.GetServerPage(req)
	if err != nil {
		global.GVA_LOG.Error("获取服务器列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total, "page": req.Page, "pageSize": req.PageSize}, "获取成功", c)
}

// GetServerList 获取服务器列表（不分页）
func (a *ServerApi) GetServerList(c *gin.Context) {
	var server system.TbServer
	if err := c.ShouldBindJSON(&server); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := serverService.GetServerList(server)
	if err != nil {
		global.GVA_LOG.Error("获取服务器列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetServerById 根据ID获取服务器
func (a *ServerApi) GetServerById(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	server, err := serverService.GetServerById(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取服务器信息失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(server, "获取成功", c)
}

// SaveOrUpdateServer 新增或更新服务器
func (a *ServerApi) SaveOrUpdateServer(c *gin.Context) {
	var server system.TbServer
	if err := c.ShouldBindJSON(&server); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if server.ID == 0 {
		server.UserId = utils.GetUserID(c)
	}
	if err := serverService.SaveOrUpdateServer(server); err != nil {
		global.GVA_LOG.Error("操作失败!", zap.Error(err))
		response.FailWithMessage("操作失败: "+err.Error(), c)
		return
	}
	if server.ID != 0 {
		response.OkWithMessage("修改成功", c)
	} else {
		response.OkWithMessage("新增成功", c)
	}
}

// DeleteServer 批量删除服务器
func (a *ServerApi) DeleteServer(c *gin.Context) {
	idsStr := c.Param("ids")
	idStrList := strings.Split(idsStr, ",")
	var ids []int
	for _, idStr := range idStrList {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := serverService.DeleteServer(ids); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}
