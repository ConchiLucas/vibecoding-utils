package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	systemRes "github.com/flipped-aurora/easy-deploy/server/model/system/response"
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
	results, err := executeGenerateCode(req, userId)
	if err != nil {
		global.GVA_LOG.Error("生成失败", zap.Error(err))
		response.FailWithMessage(fmt.Sprintf("生成失败: %v", err), c)
		return
	}

	response.OkWithDetailed(results, "生成成功", c)
}

func (a *TbGenerateProjectApi) GenerateCodePublic(c *gin.Context) {
	var req request.PublicGenerateCodeModel
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ProjectId <= 0 {
		response.FailWithMessage("projectId 必填", c)
		return
	}
	if strings.TrimSpace(req.ModuleName) == "" {
		response.FailWithMessage("moduleName 必填", c)
		return
	}
	if strings.TrimSpace(req.ModuleComment) == "" {
		response.FailWithMessage("moduleComment 必填", c)
		return
	}
	if strings.TrimSpace(req.TableStructure) == "" {
		response.FailWithMessage("tableStructure 必填", c)
		return
	}

	results, err := executeGenerateCode(request.GenerateCodeModel{
		Id:             strconv.Itoa(req.ProjectId),
		ModuleName:     req.ModuleName,
		ModuleComment:  req.ModuleComment,
		TableStructure: req.TableStructure,
		DbType:         req.DbType,
	}, "")
	if err != nil {
		global.GVA_LOG.Error("公开生成失败", zap.Error(err))
		response.FailWithMessage(fmt.Sprintf("生成失败: %v", err), c)
		return
	}
	if len(results) == 0 {
		response.FailWithMessage("生成失败: 未找到生成结果", c)
		return
	}

	response.OkWithDetailed(results[0], "生成成功", c)
}

func executeGenerateCode(req request.GenerateCodeModel, userId string) ([]systemRes.GenerateCodeProjectResult, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, fmt.Errorf("项目ID不能为空")
	}

	var results []systemRes.GenerateCodeProjectResult
	for _, idStr := range strings.Split(req.Id, ",") {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		projectId, err := strconv.Atoi(idStr)
		if err != nil || projectId <= 0 {
			return nil, fmt.Errorf("项目ID无效: %s", idStr)
		}

		var proj system.TbGenerateProject
		if err := global.GVA_DB.Where("id = ?", projectId).First(&proj).Error; err != nil {
			return nil, fmt.Errorf("项目不存在: %d", projectId)
		}
		if strings.TrimSpace(proj.DiskPath) == "" {
			return nil, fmt.Errorf("项目未配置磁盘输出路径: %d", projectId)
		}

		var paths []system.TbGenerateProjectPath
		if err := global.GVA_DB.Where("project_id = ? AND enabled = 1", projectId).Find(&paths).Error; err != nil {
			return nil, err
		}

		var pathIds []int
		for _, p := range paths {
			pathIds = append(pathIds, int(p.ID))
		}

		var models []system.TbGenerateProjectPathModel
		if len(pathIds) > 0 {
			if err := global.GVA_DB.Where("path_id IN ?", pathIds).Find(&models).Error; err != nil {
				return nil, err
			}
		}

		var holders []system.TbGenerateProjectPlaceHolder
		if err := global.GVA_DB.Where("project_id = ?", projectId).Find(&holders).Error; err != nil {
			return nil, err
		}

		recordUser := strings.TrimSpace(userId)
		if recordUser == "" || recordUser == "anonymous" || recordUser == "codex" {
			recordUser = strings.TrimSpace(proj.UserName)
		}
		if recordUser == "" {
			recordUser = "codex"
		}

		var baseHolders []system.TbGeneratePlaceHolder
		if err := global.GVA_DB.Where("user_name = ?", recordUser).Find(&baseHolders).Error; err != nil {
			return nil, err
		}

		genProj := utils.GenerateProjectInfo{
			ID:             idStr,
			ModuleName:     req.ModuleName,
			ModuleComment:  req.ModuleComment,
			TableStructure: req.TableStructure,
			DbType:         req.DbType,
		}

		generatedFiles, err := utils.GenerateCode(genProj, paths, models, holders, baseHolders, proj.DiskPath)
		if err != nil {
			return nil, err
		}

		record := system.TbGenerateRecord{
			ProjectId:      projectId,
			UserName:       recordUser,
			ModuleName:     req.ModuleName,
			ModuleComment:  req.ModuleComment,
			TableStructure: req.TableStructure,
			DbType:         req.DbType,
		}

		var exist system.TbGenerateRecord
		if err := global.GVA_DB.Where("user_name = ? AND project_id = ?", recordUser, projectId).First(&exist).Error; err == nil {
			record.ID = exist.ID
			if err := global.GVA_DB.Updates(&record).Error; err != nil {
				return nil, err
			}
		} else {
			if err := global.GVA_DB.Create(&record).Error; err != nil {
				return nil, err
			}
		}

		results = append(results, systemRes.GenerateCodeProjectResult{
			ProjectId:      projectId,
			ProjectName:    proj.ProjectName,
			DiskPath:       proj.DiskPath,
			GeneratedFiles: generatedFiles,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("项目ID不能为空")
	}
	return results, nil
}
