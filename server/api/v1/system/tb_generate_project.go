package system

import (
	"fmt"
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
	id := c.Query("id")
	if err := tbGenerateProjectService.CopyProject(id); err != nil {
		global.GVA_LOG.Error("克隆失败", zap.Error(err))
		response.FailWithMessage("克隆失败", c)
	} else {
		response.OkWithMessage("克隆成功", c)
	}
}

func (a *TbGenerateProjectApi) GlobalReplace(c *gin.Context) {
	var req request.GlobalReplace
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbGenerateProjectService.ProjectGlobalReplace(req.Id, req.FormerStr, req.ReplaceStr); err != nil {
		global.GVA_LOG.Error("全局替换失败", zap.Error(err))
		response.FailWithMessage("替换失败", c)
	} else {
		response.OkWithMessage("替换成功", c)
	}
}

func (a *TbGenerateProjectApi) GenerateCode(c *gin.Context) {
	var req request.GenerateCodeModel
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userId := getGenerateRecordUserKey(c)

	genProj := utils.GenerateProjectInfo{
		ID:             req.Id,
		ModuleName:     req.ModuleName,
		ModuleComment:  req.ModuleComment,
		TableStructure: req.TableStructure,
		DbType:         req.DbType,
	}

	var generatedProjectIds []int
	for _, idStr := range strings.Split(req.Id, ",") {
		projectId, _ := strconv.Atoi(strings.TrimSpace(idStr))
		var paths []system.TbGenerateProjectPath
		global.GVA_DB.Where("project_id = ? AND enabled = 1", idStr).Find(&paths)

		var pathIds []int
		for _, p := range paths {
			pathIds = append(pathIds, int(p.ID))
		}

		var models []system.TbGenerateProjectPathModel
		if len(pathIds) > 0 {
			global.GVA_DB.Where("path_id IN ?", pathIds).Find(&models)
		}

		var holders []system.TbGenerateProjectPlaceHolder
		global.GVA_DB.Where("project_id = ?", idStr).Find(&holders)

		var baseHolders []system.TbGeneratePlaceHolder
		global.GVA_DB.Where("user_name = ?", userId).Find(&baseHolders)

		var proj system.TbGenerateProject
		global.GVA_DB.Where("id = ?", idStr).First(&proj)

		err := utils.GenerateCode(genProj, paths, models, holders, baseHolders, proj.DiskPath)
		if err != nil {
			response.FailWithMessage(fmt.Sprintf("生成失败: %v", err), c)
			return
		}
		if projectId > 0 {
			generatedProjectIds = append(generatedProjectIds, projectId)
		}
	}

	// Add record
	if len(generatedProjectIds) == 0 {
		generatedProjectIds = append(generatedProjectIds, 0)
	}
	for _, projectId := range generatedProjectIds {
		record := system.TbGenerateRecord{
			ProjectId:      projectId,
			UserName:       userId,
			ModuleName:     req.ModuleName,
			ModuleComment:  req.ModuleComment,
			TableStructure: req.TableStructure,
			DbType:         req.DbType,
		}

		var exist system.TbGenerateRecord
		if err := global.GVA_DB.Where("user_name = ? AND project_id = ?", userId, projectId).First(&exist).Error; err == nil {
			record.ID = exist.ID
			global.GVA_DB.Updates(&record)
		} else {
			global.GVA_DB.Create(&record)
		}
	}

	response.OkWithMessage("生成成功", c)
}
