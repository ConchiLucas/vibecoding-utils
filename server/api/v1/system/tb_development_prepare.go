package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbDevelopmentPrepareApi struct{}

func (a *TbDevelopmentPrepareApi) GetList(c *gin.Context) {
	var req systemReq.DevelopmentPrepareSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbDevelopmentPrepareService.GetList(req, utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("获取开发准备列表失败", zap.Error(err))
		response.FailWithMessage("获取开发准备列表失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

func (a *TbDevelopmentPrepareApi) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailWithMessage("参数 id 错误", c)
		return
	}
	item, err := tbDevelopmentPrepareService.GetByID(uint(id), utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("获取开发准备详情失败", zap.Error(err))
		response.FailWithMessage("获取开发准备详情失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(item, "获取成功", c)
}

func (a *TbDevelopmentPrepareApi) SaveOrUpdate(c *gin.Context) {
	var item modelSystem.TbDevelopmentPrepare
	if err := c.ShouldBindJSON(&item); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbDevelopmentPrepareService.SaveOrUpdate(item, utils.GetUserID(c)); err != nil {
		global.GVA_LOG.Error("保存开发准备失败", zap.Error(err))
		response.FailWithMessage("保存开发准备失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("保存成功", c)
}

func (a *TbDevelopmentPrepareApi) Delete(c *gin.Context) {
	ids, err := parseDevelopmentPrepareIDList(c.Param("ids"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbDevelopmentPrepareService.Delete(ids, utils.GetUserID(c)); err != nil {
		global.GVA_LOG.Error("删除开发准备失败", zap.Error(err))
		response.FailWithMessage("删除开发准备失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func parseDevelopmentPrepareIDList(idsStr string) ([]int, error) {
	parts := strings.Split(idsStr, ",")
	ids := make([]int, 0, len(parts))
	for _, item := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("参数 ids 错误")
	}
	return ids, nil
}
