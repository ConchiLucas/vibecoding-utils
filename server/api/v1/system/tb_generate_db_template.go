package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGenerateDbTemplateTypeApi struct{}
type TbGenerateDbTemplateScriptApi struct{}

func (a *TbGenerateDbTemplateTypeApi) Create(c *gin.Context) {
	var req system.TbGenerateDbTemplateType
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ProjectId <= 0 {
		response.FailWithMessage("projectId 必填", c)
		return
	}
	if strings.TrimSpace(req.TypeName) == "" {
		response.FailWithMessage("业务类型名称必填", c)
		return
	}
	if err := tbGenerateDbTemplateTypeService.Create(&req); err != nil {
		global.GVA_LOG.Error("创建数据库模板业务类型失败", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithData(req, c)
	}
}

func (a *TbGenerateDbTemplateTypeApi) Delete(c *gin.Context) {
	var req system.TbGenerateDbTemplateType
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 {
		response.FailWithMessage("ID 必填", c)
		return
	}
	if err := tbGenerateDbTemplateTypeService.Delete(req); err != nil {
		global.GVA_LOG.Error("删除数据库模板业务类型失败", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateDbTemplateTypeApi) Update(c *gin.Context) {
	var req system.TbGenerateDbTemplateType
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 {
		response.FailWithMessage("ID 必填", c)
		return
	}
	if strings.TrimSpace(req.TypeName) == "" {
		response.FailWithMessage("业务类型名称必填", c)
		return
	}
	if err := tbGenerateDbTemplateTypeService.Update(&req); err != nil {
		global.GVA_LOG.Error("更新数据库模板业务类型失败", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithData(req, c)
	}
}

func (a *TbGenerateDbTemplateTypeApi) Get(c *gin.Context) {
	res, err := tbGenerateDbTemplateTypeService.Get(c.Query("id"))
	if err != nil {
		global.GVA_LOG.Error("查询数据库模板业务类型失败", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateDbTemplateTypeApi) List(c *gin.Context) {
	projectId, _ := strconv.Atoi(c.Query("projectId"))
	res, err := tbGenerateDbTemplateTypeService.List(projectId)
	if err != nil {
		global.GVA_LOG.Error("查询数据库模板业务类型列表失败", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateDbTemplateScriptApi) Create(c *gin.Context) {
	var req system.TbGenerateDbTemplateScript
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := validateDbTemplateScript(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateDbTemplateScriptService.Create(&req); err != nil {
		global.GVA_LOG.Error("创建数据库模板脚本失败", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithData(req, c)
	}
}

func (a *TbGenerateDbTemplateScriptApi) Delete(c *gin.Context) {
	var req system.TbGenerateDbTemplateScript
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 {
		response.FailWithMessage("ID 必填", c)
		return
	}
	if err := tbGenerateDbTemplateScriptService.Delete(req); err != nil {
		global.GVA_LOG.Error("删除数据库模板脚本失败", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateDbTemplateScriptApi) Update(c *gin.Context) {
	var req system.TbGenerateDbTemplateScript
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 {
		response.FailWithMessage("ID 必填", c)
		return
	}
	if err := validateDbTemplateScript(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateDbTemplateScriptService.Update(&req); err != nil {
		global.GVA_LOG.Error("更新数据库模板脚本失败", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithData(req, c)
	}
}

func (a *TbGenerateDbTemplateScriptApi) Get(c *gin.Context) {
	res, err := tbGenerateDbTemplateScriptService.Get(c.Query("id"))
	if err != nil {
		global.GVA_LOG.Error("查询数据库模板脚本失败", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateDbTemplateScriptApi) List(c *gin.Context) {
	projectId, _ := strconv.Atoi(c.Query("projectId"))
	typeId, _ := strconv.Atoi(c.Query("typeId"))
	res, err := tbGenerateDbTemplateScriptService.List(projectId, typeId)
	if err != nil {
		global.GVA_LOG.Error("查询数据库模板脚本列表失败", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func validateDbTemplateScript(req system.TbGenerateDbTemplateScript) error {
	if req.ProjectId <= 0 {
		return fmt.Errorf("projectId 必填")
	}
	if req.TypeId <= 0 {
		return fmt.Errorf("typeId 必填")
	}
	if strings.TrimSpace(req.ScriptName) == "" {
		return fmt.Errorf("脚本名称必填")
	}
	return nil
}
