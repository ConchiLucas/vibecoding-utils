package system

import (
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProjectGroupApi struct{}

// GetGroupList 获取项目组列表
func (a *ProjectGroupApi) GetGroupList(c *gin.Context) {
	userId := utils.GetUserID(c)
	list, err := projectGroupService.GetGroupList(userId)
	if err != nil {
		global.GVA_LOG.Error("获取项目组列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// SaveOrUpdateGroup 新增或更新项目组
func (a *ProjectGroupApi) SaveOrUpdateGroup(c *gin.Context) {
	var group system.TbProjectGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if group.ID == 0 {
		group.UserId = utils.GetUserID(c)
	}
	result, err := projectGroupService.SaveOrUpdateGroup(group)
	if err != nil {
		global.GVA_LOG.Error("操作失败!", zap.Error(err))
		response.FailWithMessage("操作失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// DeleteGroup 删除项目组
func (a *ProjectGroupApi) DeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := projectGroupService.DeleteGroup(id); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}
