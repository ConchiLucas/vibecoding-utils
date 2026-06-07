package system

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	systemService "github.com/flipped-aurora/easy-deploy/server/service/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGenerateProjectApi struct{}

func (a *TbGenerateProjectApi) CreateTbGenerateProject(c *gin.Context) {
	var req system.TbGenerateProject
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectService.CreateTbGenerateProject(&req)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbGenerateProjectApi) DeleteTbGenerateProject(c *gin.Context) {
	var req system.TbGenerateProject
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectService.DeleteTbGenerateProject(req)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateProjectApi) UpdateTbGenerateProject(c *gin.Context) {
	var req system.TbGenerateProject
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateProjectService.UpdateTbGenerateProject(&req)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateProjectApi) UpdateSelectedProjectInstance(c *gin.Context) {
	var req systemReq.UpdateSelectedProjectInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectService.UpdateSelectedProjectInstance(req.TemplateProjectId, req.ProjectInstanceId); err != nil {
		global.GVA_LOG.Error("更新项目选中状态失败!", zap.Error(err))
		response.FailWithMessage("更新项目选中状态失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateProjectApi) GenerateCode(c *gin.Context) {
	var req systemReq.GenerateProjectCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	res, err := tbGenerateProjectService.GenerateCode(req)
	if err != nil {
		global.GVA_LOG.Error("生成代码失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
	} else {
		res.PromptUrl = buildGenerateProjectPromptURL(c, req, res)
		response.OkWithDetailed(res, "生成成功", c)
	}
}

func (a *TbGenerateProjectApi) PublicPrompt(c *gin.Context) {
	req, err := parsePublicGeneratePromptReq(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	res, err := tbGenerateProjectPathService.BuildPromptSummary(req)
	if err != nil {
		global.GVA_LOG.Error("生成公开提示词失败!", zap.Error(err))
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(res.Prompt))
}

func (a *TbGenerateProjectApi) GetTbGenerateProject(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGenerateProjectService.GetTbGenerateProject(id)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectApi) GetTbGenerateProjectList(c *gin.Context) {
	projectConfigId := 0
	if v := c.Query("projectConfigId"); v != "" {
		fmt.Sscanf(v, "%d", &projectConfigId)
	}
	res, err := tbGenerateProjectService.GetTbGenerateProjectList(projectConfigId)
	if err != nil {
		global.GVA_LOG.Error("查询列表失败!", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateProjectApi) CopyProject(c *gin.Context) {
	var req systemReq.CopyGenerateProjectReq
	if c.Request.Method == http.MethodGet {
		if parsed, err := strconv.Atoi(strings.TrimSpace(c.Query("id"))); err == nil {
			req.SourceProjectId = parsed
		}
	} else if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	res, err := tbGenerateProjectService.CopyProject(req)
	if err != nil {
		global.GVA_LOG.Error("克隆失败", zap.Error(err))
		response.FailWithMessage("克隆失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func parsePublicGeneratePromptReq(c *gin.Context) (systemReq.BuildGenerateProjectPromptSummaryReq, error) {
	projectInstanceId, err := strconv.Atoi(strings.TrimSpace(c.Query("projectInstanceId")))
	if err != nil || projectInstanceId <= 0 {
		return systemReq.BuildGenerateProjectPromptSummaryReq{}, fmt.Errorf("projectInstanceId 必填")
	}

	pathSet := 0
	if rawPathSet := strings.TrimSpace(c.Query("pathSet")); rawPathSet != "" {
		parsed, err := strconv.Atoi(rawPathSet)
		if err != nil {
			return systemReq.BuildGenerateProjectPromptSummaryReq{}, fmt.Errorf("pathSet 无效")
		}
		pathSet = parsed
	}

	pathIds := make([]uint, 0)
	for _, part := range strings.Split(c.Query("pathIds"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil || parsed == 0 {
			return systemReq.BuildGenerateProjectPromptSummaryReq{}, fmt.Errorf("pathIds 无效")
		}
		pathIds = append(pathIds, uint(parsed))
	}

	module := strings.TrimSpace(c.Query("module"))
	tableName := strings.TrimSpace(c.Query("tableName"))
	if module == "" {
		return systemReq.BuildGenerateProjectPromptSummaryReq{}, fmt.Errorf("module 必填")
	}
	if tableName == "" {
		return systemReq.BuildGenerateProjectPromptSummaryReq{}, fmt.Errorf("TableName 必填")
	}

	overwrite := false
	if rawOverwrite := strings.TrimSpace(c.Query("overwrite")); rawOverwrite != "" {
		parsed, err := strconv.ParseBool(rawOverwrite)
		if err != nil {
			return systemReq.BuildGenerateProjectPromptSummaryReq{}, fmt.Errorf("overwrite 无效")
		}
		overwrite = parsed
	}

	return systemReq.BuildGenerateProjectPromptSummaryReq{
		ProjectInstanceId: projectInstanceId,
		PathSet:           pathSet,
		PathIds:           pathIds,
		Module:            module,
		TableName:         tableName,
		Overwrite:         overwrite,
	}, nil
}

func buildGenerateProjectPromptURL(c *gin.Context, req systemReq.GenerateProjectCodeReq, res systemService.GenerateProjectCodeResult) string {
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}

	prefix := strings.TrimRight(global.GVA_CONFIG.System.RouterPrefix, "/")
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	query := url.Values{}
	query.Set("projectInstanceId", strconv.Itoa(res.ProjectInstanceId))
	query.Set("pathSet", strconv.Itoa(res.PathSet))
	query.Set("module", strings.TrimSpace(req.Module))
	query.Set("tableName", strings.TrimSpace(req.TableName))
	query.Set("overwrite", strconv.FormatBool(req.Overwrite))

	pathIds := make([]string, 0, len(res.Files))
	for _, file := range res.Files {
		if file.PathId > 0 {
			pathIds = append(pathIds, strconv.FormatUint(uint64(file.PathId), 10))
		}
	}
	if len(pathIds) > 0 {
		query.Set("pathIds", strings.Join(pathIds, ","))
	}

	return fmt.Sprintf("%s://%s%s/tbgenerateproject/publicPrompt?%s", scheme, host, prefix, query.Encode())
}
