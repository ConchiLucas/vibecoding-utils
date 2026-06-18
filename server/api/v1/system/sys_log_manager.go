package system

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LogManagerApi struct{}

func (a *LogManagerApi) GetLogProjectPage(c *gin.Context) {
	var req request.TbLogProjectSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := logManagerService.GetLogProjectPage(req)
	if err != nil {
		global.GVA_LOG.Error("获取日志项目列表失败", zap.Error(err))
		response.FailWithMessage("获取日志项目列表失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total, "page": req.Page, "pageSize": req.PageSize}, "获取成功", c)
}

func (a *LogManagerApi) GetLogProjectGroups(c *gin.Context) {
	list, err := logManagerService.GetLogProjectGroups()
	if err != nil {
		global.GVA_LOG.Error("获取日志项目组失败", zap.Error(err))
		response.FailWithMessage("获取日志项目组失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

func (a *LogManagerApi) SaveOrUpdateLogProject(c *gin.Context) {
	var project system.TbLogProject
	if err := c.ShouldBindJSON(&project); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if project.ID == 0 {
		project.UserId = utils.GetUserID(c)
	}
	if err := logManagerService.SaveOrUpdateLogProject(project); err != nil {
		global.GVA_LOG.Error("保存日志项目失败", zap.Error(err))
		response.FailWithMessage("保存日志项目失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("保存成功", c)
}

func (a *LogManagerApi) DeleteLogProject(c *gin.Context) {
	ids, err := parseIDList(c.Param("ids"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := logManagerService.DeleteLogProject(ids); err != nil {
		global.GVA_LOG.Error("删除日志项目失败", zap.Error(err))
		response.FailWithMessage("删除日志项目失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *LogManagerApi) SaveOrUpdateLogRoute(c *gin.Context) {
	var route system.TbLogProjectRoute
	if err := c.ShouldBindJSON(&route); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := logManagerService.SaveOrUpdateLogRoute(route); err != nil {
		global.GVA_LOG.Error("保存日志服务路线失败", zap.Error(err))
		response.FailWithMessage("保存日志服务路线失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("保存成功", c)
}

func (a *LogManagerApi) DeleteLogRoute(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.FailWithMessage("参数 id 错误", c)
		return
	}
	if err := logManagerService.DeleteLogRoute(id); err != nil {
		global.GVA_LOG.Error("删除日志服务路线失败", zap.Error(err))
		response.FailWithMessage("删除日志服务路线失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *LogManagerApi) ListDockerServices(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	scope := c.DefaultQuery("scope", "")
	list, err := logManagerService.ListDockerServices(uint(id), scope)
	if err != nil {
		global.GVA_LOG.Error("获取 Docker 服务列表失败", zap.Error(err))
		response.FailWithMessage("获取 Docker 服务列表失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

func (a *LogManagerApi) ServiceStatusStream(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	scope := c.DefaultQuery("scope", "")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	sendStatus := func() bool {
		list, err := logManagerService.ListDockerServices(uint(id), scope)
		if err != nil {
			global.GVA_LOG.Error("日志管理服务状态读取失败", zap.Error(err))
			c.SSEvent("error", err.Error())
			return false
		}
		payload, err := json.Marshal(list)
		if err != nil {
			global.GVA_LOG.Error("日志管理服务状态序列化失败", zap.Error(err))
			c.SSEvent("error", err.Error())
			return false
		}
		c.SSEvent("status", string(payload))
		return true
	}

	first := true
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			global.GVA_LOG.Warn("客户端断开日志管理状态SSE连接")
			return false
		default:
		}
		if first {
			first = false
			return sendStatus()
		}
		select {
		case <-ticker.C:
		case <-c.Request.Context().Done():
			global.GVA_LOG.Warn("客户端断开日志管理状态SSE连接")
			return false
		}
		if !sendStatus() {
			return false
		}
		return true
	})
}

func (a *LogManagerApi) ServiceGroupStream(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	action := c.DefaultQuery("action", "start")
	scope := c.DefaultQuery("scope", "")
	streamLogManager(c, "服务组执行完成", func(logCh chan string) error {
		return logManagerService.StreamProjectServiceGroup(uint(id), action, scope, logCh)
	}, fmt.Sprintf("日志管理服务组执行失败, 项目ID=%d", id))
}

func (a *LogManagerApi) DeployStream(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "local"
	}
	streamLogManager(c, "服务启动执行完成", func(logCh chan string) error {
		return logManagerService.StreamProjectRoute(uint(id), env, "start", logCh)
	}, fmt.Sprintf("日志管理服务启动失败, 项目ID=%d", id))
}

func (a *LogManagerApi) StopStream(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "local"
	}
	streamLogManager(c, "服务关闭执行完成", func(logCh chan string) error {
		return logManagerService.StreamProjectRoute(uint(id), env, "stop", logCh)
	}, fmt.Sprintf("日志管理服务关闭失败, 项目ID=%d", id))
}

func (a *LogManagerApi) RestartStream(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "local"
	}
	serviceName := c.Query("service")
	streamLogManager(c, "服务重启执行完成", func(logCh chan string) error {
		if strings.TrimSpace(serviceName) != "" {
			return logManagerService.StreamDockerComposeServiceRestart(uint(id), env, serviceName, logCh)
		}
		return logManagerService.StreamProjectRoute(uint(id), env, "restart", logCh)
	}, fmt.Sprintf("日志管理服务重启失败, 项目ID=%d", id))
}

func (a *LogManagerApi) DockerLogStream(c *gin.Context) {
	id, err := parseLogManagerProjectID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "local"
	}
	serviceName := c.Query("service")
	streamLogManager(c, "Docker 日志读取结束", func(logCh chan string) error {
		return logManagerService.StreamDockerLogs(c.Request.Context(), uint(id), env, serviceName, logCh)
	}, fmt.Sprintf("日志管理 Docker 日志读取失败, 项目ID=%d", id))
}

func streamLogManager(c *gin.Context, doneMessage string, run func(chan string) error, errorLogMessage string) {
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
					global.GVA_LOG.Error(errorLogMessage, zap.Error(runErr))
					c.SSEvent("error", runErr.Error())
				} else {
					c.SSEvent("done", doneMessage)
				}
				return false
			}
			c.SSEvent("log", msg)
			return true
		case <-c.Request.Context().Done():
			global.GVA_LOG.Warn("客户端断开日志管理SSE连接")
			return false
		}
	})
}

func parseLogManagerProjectID(c *gin.Context) (int, error) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("projectId")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("参数 projectId 错误")
	}
	return id, nil
}

func parseIDList(idsStr string) ([]int, error) {
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
