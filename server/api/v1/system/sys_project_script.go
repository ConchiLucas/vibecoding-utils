package system

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProjectScriptApi struct{}

// GetProjectScriptPage 分页获取项目脚本列表
func (a *ProjectScriptApi) GetProjectScriptPage(c *gin.Context) {
	var req request.TbProjectScriptSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := projectScriptService.GetProjectScriptPage(req)
	if err != nil {
		global.GVA_LOG.Error("获取脚本列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(gin.H{"list": list, "total": total, "page": req.Page, "pageSize": req.PageSize}, "获取成功", c)
}

// GetProjectScriptList 获取项目脚本列表（不分页）
func (a *ProjectScriptApi) GetProjectScriptList(c *gin.Context) {
	var script system.TbProjectScript
	if err := c.ShouldBindJSON(&script); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := projectScriptService.GetProjectScriptList(script.ProjectId, script.RouteId)
	if err != nil {
		global.GVA_LOG.Error("获取脚本列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}

// GetProjectScriptById 根据ID获取项目脚本
func (a *ProjectScriptApi) GetProjectScriptById(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	script, err := projectScriptService.GetProjectScriptById(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取脚本信息失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(script, "获取成功", c)
}

// SaveOrUpdateProjectScript 新增或更新项目脚本
func (a *ProjectScriptApi) SaveOrUpdateProjectScript(c *gin.Context) {
	var script system.TbProjectScript
	if err := c.ShouldBindJSON(&script); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := projectScriptService.SaveOrUpdateProjectScript(script); err != nil {
		global.GVA_LOG.Error("操作失败!", zap.Error(err))
		response.FailWithMessage("操作失败: "+err.Error(), c)
		return
	}
	if script.ID != 0 {
		response.OkWithMessage("修改成功", c)
	} else {
		response.OkWithMessage("新增成功", c)
	}
}

// DeleteProjectScript 批量删除项目脚本
func (a *ProjectScriptApi) DeleteProjectScript(c *gin.Context) {
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

	if err := projectScriptService.DeleteProjectScript(ids); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// UploadFile 上传文件
func (a *ProjectScriptApi) UploadFile(c *gin.Context) {
	projectIdStr := c.PostForm("projectId")
	fileNickName := c.PostForm("fileNickName")
	projectId, err := strconv.Atoi(projectIdStr)
	if err != nil {
		response.FailWithMessage("项目ID参数错误", c)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.FailWithMessage("获取上传文件失败", c)
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		response.FailWithMessage("配置及脚本文件大小不能超过10MB", c)
		return
	}

	buf := make([]byte, header.Size)
	_, err = file.Read(buf)
	if err != nil {
		response.FailWithMessage("读取文件字节失败", c)
		return
	}
	contentStr := string(buf)

	// 保存脚本记录
	script := system.TbProjectScript{
		ProjectId:    projectId,
		FileNickName: fileNickName,
		FileName:     header.Filename,
		Content:      contentStr,
	}
	if err := projectScriptService.SaveOrUpdateProjectScript(script); err != nil {
		global.GVA_LOG.Error("保存脚本记录失败!", zap.Error(err))
		response.FailWithMessage("保存记录失败", c)
		return
	}

	response.OkWithMessage("上传文件成功", c)
}

// DownloadFile 下载文件
func (a *ProjectScriptApi) DownloadFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	script, err := projectScriptService.GetProjectScriptById(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取脚本信息失败!", zap.Error(err))
		response.FailWithMessage("获取脚本信息失败", c)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, script.FileName))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", []byte(script.Content))
}

// PreviewFile 预览文件内容
func (a *ProjectScriptApi) PreviewFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	script, err := projectScriptService.GetProjectScriptById(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取脚本信息失败!", zap.Error(err))
		response.FailWithMessage("获取脚本信息失败", c)
		return
	}

	response.OkWithDetailed(script.Content, "获取成功", c)
}

// UpdateContent 更新文件内容
func (a *ProjectScriptApi) UpdateContent(c *gin.Context) {
	var req request.ScriptUpdateContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := projectScriptService.UpdateScriptContent(req.ID, req.Content); err != nil {
		global.GVA_LOG.Error("更新文件内容失败!", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// tryDecodeContent 尝试解码文件内容
func tryDecodeContent(content []byte) string {
	// 尝试直接作为UTF-8
	contentStr := string(content)
	// 简单检查是否包含无效UTF-8字符
	if isValidUTF8(contentStr) {
		return contentStr
	}
	// 如果不是有效UTF-8，尝试替换无效字符
	return strings.ToValidUTF8(contentStr, "�")
}

// isValidUTF8 检查字符串是否是有效的UTF-8
func isValidUTF8(s string) bool {
	return utf8.ValidString(s)
}
