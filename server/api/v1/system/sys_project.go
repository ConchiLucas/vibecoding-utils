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

type ProjectApi struct{}

// GetProjectPage 分页获取项目列表
func (a *ProjectApi) GetProjectPage(c *gin.Context) {
	var req request.TbProjectSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := projectService.GetProjectPage(req)
	if err != nil {
		global.GVA_LOG.Error("获取项目列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total, "page": req.Page, "pageSize": req.PageSize}, "获取成功", c)
}

// GetProjectList 获取项目列表（不分页）
func (a *ProjectApi) GetProjectList(c *gin.Context) {
	var project system.TbProject
	if err := c.ShouldBindJSON(&project); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := projectService.GetProjectList(project)
	if err != nil {
		global.GVA_LOG.Error("获取项目列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetProjectById 根据ID获取项目
func (a *ProjectApi) GetProjectById(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	project, err := projectService.GetProjectById(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取项目信息失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(project, "获取成功", c)
}

// GetNextDeployPort 获取下一次部署建议端口。
func (a *ProjectApi) GetNextDeployPort(c *gin.Context) {
	portType := c.Query("type")
	if portType == "" {
		response.FailWithMessage("参数 type 不能为空", c)
		return
	}

	result, err := projectService.GetNextDeployPort(portType)
	if err != nil {
		global.GVA_LOG.Error("获取建议端口失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "获取成功", c)
}

// SaveOrUpdateProject 新增或更新项目
func (a *ProjectApi) SaveOrUpdateProject(c *gin.Context) {
	var project system.TbProject
	if err := c.ShouldBindJSON(&project); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if project.ID == 0 {
		project.UserId = utils.GetUserID(c)
	}
	if err := projectService.SaveOrUpdateProject(project); err != nil {
		global.GVA_LOG.Error("操作失败!", zap.Error(err))
		response.FailWithMessage("操作失败: "+err.Error(), c)
		return
	}
	if project.ID != 0 {
		response.OkWithMessage("修改成功", c)
	} else {
		response.OkWithMessage("新增成功", c)
	}
}

// DeleteProject 批量删除项目
func (a *ProjectApi) DeleteProject(c *gin.Context) {
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
	if err := projectService.DeleteProject(ids); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// ProcessDeploy 执行项目部署
func (a *ProjectApi) ProcessDeploy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "server"
	}
	if err := deployService.ProcessDeploy(uint(id), env); err != nil {
		global.GVA_LOG.Error("部署失败!", zap.Error(err))
		response.FailWithMessage("部署失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("执行部署成功", c)
}
