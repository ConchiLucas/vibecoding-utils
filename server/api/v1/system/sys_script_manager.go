package system

import (
	"fmt"
	"io"
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ScriptManagerApi struct{}

func (a *ScriptManagerApi) ListCategories(c *gin.Context) {
	list, err := scriptManagerService.ListCategories(utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("获取脚本分类失败", zap.Error(err))
		response.FailWithMessage("获取脚本分类失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

func (a *ScriptManagerApi) CreateCategory(c *gin.Context) {
	var category system.TbScriptCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := scriptManagerService.SaveCategory(category, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) UpdateCategory(c *gin.Context) {
	var category system.TbScriptCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	category.ID = id
	result, err := scriptManagerService.SaveCategory(category, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) DeleteCategory(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := scriptManagerService.DeleteCategory(id, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *ScriptManagerApi) ListWorkflows(c *gin.Context) {
	var req systemReq.ScriptWorkflowSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := scriptManagerService.ListWorkflows(req, utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("获取脚本流程失败", zap.Error(err))
		response.FailWithMessage("获取脚本流程失败", c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total, "page": req.Page, "pageSize": req.PageSize}, "获取成功", c)
}

func (a *ScriptManagerApi) GetWorkflow(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	workflow, err := scriptManagerService.GetWorkflow(id, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage("脚本流程不存在", c)
		return
	}
	response.OkWithDetailed(workflow, "获取成功", c)
}

func (a *ScriptManagerApi) CreateWorkflow(c *gin.Context) {
	var workflow system.TbScriptWorkflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := scriptManagerService.SaveWorkflow(workflow, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) UpdateWorkflow(c *gin.Context) {
	var workflow system.TbScriptWorkflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	workflow.ID = id
	result, err := scriptManagerService.SaveWorkflow(workflow, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) DeleteWorkflow(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := scriptManagerService.DeleteWorkflow(id, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *ScriptManagerApi) CreateStep(c *gin.Context) {
	var step system.TbScriptStep
	if err := c.ShouldBindJSON(&step); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := scriptManagerService.SaveStep(step, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) UpdateStep(c *gin.Context) {
	var step system.TbScriptStep
	if err := c.ShouldBindJSON(&step); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	step.ID = id
	result, err := scriptManagerService.SaveStep(step, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) DeleteStep(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := scriptManagerService.DeleteStep(id, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *ScriptManagerApi) ListResourceCategories(c *gin.Context) {
	list, err := scriptManagerService.ListResourceCategories(utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("获取资源配置失败", zap.Error(err))
		response.FailWithMessage("获取资源配置失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

func (a *ScriptManagerApi) CreateResourceCategory(c *gin.Context) {
	var category system.TbScriptResourceCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := scriptManagerService.SaveResourceCategory(category, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) UpdateResourceCategory(c *gin.Context) {
	var category system.TbScriptResourceCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	category.ID = id
	result, err := scriptManagerService.SaveResourceCategory(category, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) DeleteResourceCategory(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := scriptManagerService.DeleteResourceCategory(id, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *ScriptManagerApi) CreateResourceConfig(c *gin.Context) {
	var config system.TbScriptResourceConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	result, err := scriptManagerService.SaveResourceConfig(config, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) UpdateResourceConfig(c *gin.Context) {
	var config system.TbScriptResourceConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	config.ID = id
	result, err := scriptManagerService.SaveResourceConfig(config, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "保存成功", c)
}

func (a *ScriptManagerApi) DeleteResourceConfig(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if err := scriptManagerService.DeleteResourceConfig(id, utils.GetUserID(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *ScriptManagerApi) ListExecutions(c *gin.Context) {
	var req systemReq.ScriptExecutionSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := scriptManagerService.ListExecutions(req, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage("获取执行日志失败", c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total, "page": req.Page, "pageSize": req.PageSize}, "获取成功", c)
}

func (a *ScriptManagerApi) GetExecutionLog(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	execution, err := scriptManagerService.GetExecutionLog(id, utils.GetUserID(c))
	if err != nil {
		response.FailWithMessage("执行记录不存在", c)
		return
	}
	response.OkWithDetailed(execution, "获取成功", c)
}

func (a *ScriptManagerApi) ExecuteWorkflowStream(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	a.streamScriptExecution(c, func(logCh chan string) error {
		return scriptManagerService.ExecuteWorkflowWithLog(c.Request.Context(), id, utils.GetUserID(c), logCh)
	})
}

func (a *ScriptManagerApi) ExecuteStepStream(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	a.streamScriptExecution(c, func(logCh chan string) error {
		return scriptManagerService.ExecuteStepWithLog(c.Request.Context(), id, utils.GetUserID(c), logCh)
	})
}

func (a *ScriptManagerApi) streamScriptExecution(c *gin.Context, run func(chan string) error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	logCh := make(chan string, 300)
	doneCh := make(chan error, 1)
	go func() {
		defer close(logCh)
		doneCh <- run(logCh)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-logCh:
			if !ok {
				runErr := <-doneCh
				if runErr != nil {
					c.SSEvent("error", runErr.Error())
				} else {
					c.SSEvent("done", "执行完成")
				}
				return false
			}
			c.SSEvent("log", msg)
			return true
		case <-c.Request.Context().Done():
			global.GVA_LOG.Warn("客户端断开脚本库SSE连接")
			return false
		}
	})
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return uint(id), nil
}
