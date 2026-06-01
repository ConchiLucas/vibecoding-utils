package system

import (
	"fmt"
	"io"
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProcessDeployStream SSE 流式部署日志推送
func (a *ProjectApi) ProcessDeployStream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "server"
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁止 nginx 缓冲

	// 创建日志 channel（带缓冲，防止阻塞命令执行）
	logCh := make(chan string, 200)

	// 异步执行部署
	doneCh := make(chan error, 1)
	go func() {
		defer close(logCh)
		err := deployService.ProcessDeployWithLog(uint(id), env, logCh)
		doneCh <- err
	}()

	// 流式推送日志
	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-logCh:
			if !ok {
				// channel 关闭，部署结束
				deployErr := <-doneCh
				if deployErr != nil {
					global.GVA_LOG.Error("部署失败!", zap.Error(deployErr))
					c.SSEvent("error", deployErr.Error())
				} else {
					c.SSEvent("done", "部署完成")
				}
				return false
			}
			c.SSEvent("log", msg)
			return true
		case <-c.Request.Context().Done():
			// 客户端断开连接
			global.GVA_LOG.Warn(fmt.Sprintf("客户端断开SSE连接, 项目ID=%d", id))
			return false
		}
	})
}

// ProcessStopStream SSE 流式关闭日志推送
func (a *ProjectApi) ProcessStopStream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "server"
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	logCh := make(chan string, 200)

	doneCh := make(chan error, 1)
	go func() {
		defer close(logCh)
		err := deployService.ProcessStopWithLog(uint(id), env, logCh)
		doneCh <- err
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-logCh:
			if !ok {
				stopErr := <-doneCh
				if stopErr != nil {
					global.GVA_LOG.Error("关闭失败!", zap.Error(stopErr))
					c.SSEvent("error", stopErr.Error())
				} else {
					c.SSEvent("done", "关闭完成")
				}
				return false
			}
			c.SSEvent("log", msg)
			return true
		case <-c.Request.Context().Done():
			global.GVA_LOG.Warn(fmt.Sprintf("客户端断开SSE连接(stop), 项目ID=%d", id))
			return false
		}
	})
}

// ProcessDockerLogStream SSE 推送 Docker 实时日志。
func (a *ProjectApi) ProcessDockerLogStream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	env := c.Query("env")
	if env == "" {
		env = "server"
	}
	serviceName := c.Query("service")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	logCh := make(chan string, 300)
	doneCh := make(chan error, 1)
	go func() {
		defer close(logCh)
		doneCh <- deployService.StreamDockerLogs(c.Request.Context(), uint(id), env, serviceName, logCh)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-logCh:
			if !ok {
				logErr := <-doneCh
				if logErr != nil {
					global.GVA_LOG.Error("Docker日志读取失败!", zap.Error(logErr))
					c.SSEvent("error", logErr.Error())
				} else {
					c.SSEvent("done", "日志连接已关闭")
				}
				return false
			}
			c.SSEvent("log", msg)
			return true
		case <-c.Request.Context().Done():
			global.GVA_LOG.Warn(fmt.Sprintf("客户端断开Docker日志SSE连接, 项目ID=%d", id))
			return false
		}
	})
}
