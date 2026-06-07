package system

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type ScriptManagerService struct{}

const (
	ScriptStepTypeLocalExec      = "local_exec"
	ScriptStepTypeLocalUpload    = "local_upload"
	ScriptStepTypeTargetDownload = "target_download"
	ScriptStepTypeTargetExec     = "target_exec"

	ScriptResourceCategoryTypeFixed    = "fixed"
	ScriptResourceCategoryTypeDynamic  = "dynamic"
	ScriptResourceCategoryTypeConstant = "constant"

	scriptExecutionScopeWorkflow = "workflow"
	scriptExecutionScopeStep     = "step"
	scriptExecutionStatusRunning = "running"
	scriptExecutionStatusSuccess = "success"
	scriptExecutionStatusFailed  = "failed"
)

func (s *ScriptManagerService) ListCategories(userID uint) ([]modelSystem.TbScriptCategory, error) {
	var list []modelSystem.TbScriptCategory
	err := global.GVA_DB.Where("user_id = ?", userID).Order("id ASC").Find(&list).Error
	return list, err
}

func (s *ScriptManagerService) SaveCategory(category modelSystem.TbScriptCategory, userID uint) (modelSystem.TbScriptCategory, error) {
	category.CategoryName = strings.TrimSpace(category.CategoryName)
	if category.CategoryName == "" {
		return category, fmt.Errorf("分类名称不能为空")
	}
	if category.ID == 0 {
		category.UserId = userID
		return category, global.GVA_DB.Create(&category).Error
	}
	err := global.GVA_DB.Model(&modelSystem.TbScriptCategory{}).
		Where("id = ? AND user_id = ?", category.ID, userID).
		Updates(map[string]interface{}{
			"category_name": category.CategoryName,
			"description":   category.Description,
		}).Error
	if err != nil {
		return category, err
	}
	return s.GetCategory(category.ID, userID)
}

func (s *ScriptManagerService) GetCategory(id uint, userID uint) (modelSystem.TbScriptCategory, error) {
	var category modelSystem.TbScriptCategory
	err := global.GVA_DB.Where("id = ? AND user_id = ?", id, userID).First(&category).Error
	return category, err
}

func (s *ScriptManagerService) DeleteCategory(id uint, userID uint) error {
	var count int64
	if err := global.GVA_DB.Model(&modelSystem.TbScriptWorkflow{}).
		Where("category_id = ? AND user_id = ?", id, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该分类下还有脚本流程，请先删除或移动流程")
	}
	return global.GVA_DB.Where("id = ? AND user_id = ?", id, userID).Delete(&modelSystem.TbScriptCategory{}).Error
}

func (s *ScriptManagerService) ListWorkflows(req systemReq.ScriptWorkflowSearch, userID uint) ([]modelSystem.TbScriptWorkflow, int64, error) {
	db := global.GVA_DB.Model(&modelSystem.TbScriptWorkflow{}).Where("user_id = ?", userID)
	if req.CategoryId != 0 {
		db = db.Where("category_id = ?", req.CategoryId)
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("workflow_name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []modelSystem.TbScriptWorkflow
	err := db.Scopes(req.Paginate()).
		Preload("Steps", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("id ASC")
		}).
		Order("id ASC").
		Find(&list).Error
	return list, total, err
}

func (s *ScriptManagerService) GetWorkflow(id uint, userID uint) (modelSystem.TbScriptWorkflow, error) {
	var workflow modelSystem.TbScriptWorkflow
	err := global.GVA_DB.
		Preload("Steps", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("id ASC")
		}).
		Where("id = ? AND user_id = ?", id, userID).
		First(&workflow).Error
	return workflow, err
}

func (s *ScriptManagerService) SaveWorkflow(workflow modelSystem.TbScriptWorkflow, userID uint) (modelSystem.TbScriptWorkflow, error) {
	workflow.WorkflowName = strings.TrimSpace(workflow.WorkflowName)
	if workflow.WorkflowName == "" {
		return workflow, fmt.Errorf("流程名称不能为空")
	}
	if workflow.ID == 0 {
		workflow.UserId = userID
		return workflow, global.GVA_DB.Create(&workflow).Error
	}
	err := global.GVA_DB.Model(&modelSystem.TbScriptWorkflow{}).
		Where("id = ? AND user_id = ?", workflow.ID, userID).
		Updates(map[string]interface{}{
			"category_id":   workflow.CategoryId,
			"workflow_name": workflow.WorkflowName,
			"description":   workflow.Description,
		}).Error
	if err != nil {
		return workflow, err
	}
	return s.GetWorkflow(workflow.ID, userID)
}

func (s *ScriptManagerService) DeleteWorkflow(id uint, userID uint) error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var workflow modelSystem.TbScriptWorkflow
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&workflow).Error; err != nil {
			return err
		}
		if err := tx.Where("workflow_id = ?", id).Delete(&modelSystem.TbScriptStep{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workflow_id = ?", id).Delete(&modelSystem.TbScriptExecution{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workflow_id = ? AND user_id = ?", id, userID).Delete(&modelSystem.TbScriptResourceConfig{}).Error; err != nil {
			return err
		}
		return tx.Delete(&workflow).Error
	})
}

func (s *ScriptManagerService) SaveStep(step modelSystem.TbScriptStep, userID uint) (modelSystem.TbScriptStep, error) {
	if _, err := s.GetWorkflow(step.WorkflowId, userID); err != nil {
		return step, fmt.Errorf("脚本流程不存在或无权限")
	}
	step.StepName = strings.TrimSpace(step.StepName)
	if step.StepName == "" {
		return step, fmt.Errorf("步骤名称不能为空")
	}
	if step.StepType == "" {
		step.StepType = ScriptStepTypeLocalExec
	}
	if !isValidScriptStepType(step.StepType) {
		return step, fmt.Errorf("不支持的步骤类型: %s", step.StepType)
	}
	if step.ID == 0 {
		return step, global.GVA_DB.Create(&step).Error
	}
	err := global.GVA_DB.Model(&modelSystem.TbScriptStep{}).
		Where("id = ? AND workflow_id = ?", step.ID, step.WorkflowId).
		Updates(map[string]interface{}{
			"step_name":      step.StepName,
			"step_type":      step.StepType,
			"script_content": step.ScriptContent,
			"placeholders":   step.Placeholders,
		}).Error
	if err != nil {
		return step, err
	}
	return s.GetStep(step.ID, userID)
}

func (s *ScriptManagerService) GetStep(id uint, userID uint) (modelSystem.TbScriptStep, error) {
	var step modelSystem.TbScriptStep
	if err := global.GVA_DB.Where("id = ?", id).First(&step).Error; err != nil {
		return step, err
	}
	if _, err := s.GetWorkflow(step.WorkflowId, userID); err != nil {
		return step, err
	}
	return step, nil
}

func (s *ScriptManagerService) DeleteStep(id uint, userID uint) error {
	step, err := s.GetStep(id, userID)
	if err != nil {
		return err
	}
	return global.GVA_DB.Delete(&step).Error
}

func (s *ScriptManagerService) ListResourceCategories(userID uint) ([]modelSystem.TbScriptResourceCategory, error) {
	var list []modelSystem.TbScriptResourceCategory
	err := global.GVA_DB.
		Preload("Configs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("id ASC")
		}).
		Where("user_id = ?", userID).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (s *ScriptManagerService) SaveResourceCategory(category modelSystem.TbScriptResourceCategory, userID uint) (modelSystem.TbScriptResourceCategory, error) {
	category.CategoryName = strings.TrimSpace(category.CategoryName)
	if category.CategoryName == "" {
		return category, fmt.Errorf("资源分类名称不能为空")
	}
	category.CategoryType = strings.TrimSpace(category.CategoryType)
	if category.CategoryType == "" {
		category.CategoryType = ScriptResourceCategoryTypeFixed
	}
	if !isValidScriptResourceCategoryType(category.CategoryType) {
		return category, fmt.Errorf("不支持的资源分类类型: %s", category.CategoryType)
	}
	if category.ID == 0 {
		category.UserId = userID
		return category, global.GVA_DB.Create(&category).Error
	}
	err := global.GVA_DB.Model(&modelSystem.TbScriptResourceCategory{}).
		Where("id = ? AND user_id = ?", category.ID, userID).
		Updates(map[string]interface{}{
			"category_name": category.CategoryName,
			"category_type": category.CategoryType,
		}).Error
	if err != nil {
		return category, err
	}
	return s.GetResourceCategory(category.ID, userID)
}

func (s *ScriptManagerService) GetResourceCategory(id uint, userID uint) (modelSystem.TbScriptResourceCategory, error) {
	var category modelSystem.TbScriptResourceCategory
	err := global.GVA_DB.
		Preload("Configs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("id ASC")
		}).
		Where("id = ? AND user_id = ?", id, userID).
		First(&category).Error
	return category, err
}

func (s *ScriptManagerService) DeleteResourceCategory(id uint, userID uint) error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var category modelSystem.TbScriptResourceCategory
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&category).Error; err != nil {
			return err
		}
		if err := tx.Where("category_id = ? AND user_id = ?", id, userID).Delete(&modelSystem.TbScriptResourceConfig{}).Error; err != nil {
			return err
		}
		return tx.Delete(&category).Error
	})
}

func (s *ScriptManagerService) SaveResourceConfig(config modelSystem.TbScriptResourceConfig, userID uint) (modelSystem.TbScriptResourceConfig, error) {
	category, err := s.GetResourceCategory(config.CategoryId, userID)
	if err != nil {
		return config, fmt.Errorf("资源分类不存在或无权限")
	}
	if category.CategoryType == ScriptResourceCategoryTypeDynamic {
		if config.WorkflowId == 0 {
			return config, fmt.Errorf("动态配置必须选择脚本流程")
		}
		if _, err := s.GetWorkflow(config.WorkflowId, userID); err != nil {
			return config, fmt.Errorf("脚本流程不存在或无权限")
		}
	} else {
		config.WorkflowId = 0
	}
	config.ConfigName = strings.TrimSpace(config.ConfigName)
	if config.ConfigName == "" {
		return config, fmt.Errorf("配置名称不能为空")
	}
	if category.CategoryType == ScriptResourceCategoryTypeFixed {
		config.Rows = sanitizeFixedResourceConfigRows(config.Rows)
	}
	config.PlaceholderKey = strings.TrimSpace(config.PlaceholderKey)
	if err := s.validateResourceConfigIdentity(config, category, userID); err != nil {
		return config, err
	}
	if config.ID == 0 {
		config.UserId = userID
		return config, global.GVA_DB.Create(&config).Error
	}
	err = global.GVA_DB.Model(&modelSystem.TbScriptResourceConfig{}).
		Where("id = ? AND user_id = ?", config.ID, userID).
		Updates(map[string]interface{}{
			"category_id":     config.CategoryId,
			"workflow_id":     config.WorkflowId,
			"config_name":     config.ConfigName,
			"placeholder_key": config.PlaceholderKey,
			"rows":            config.Rows,
		}).Error
	if err != nil {
		return config, err
	}
	return s.GetResourceConfig(config.ID, userID)
}

func sanitizeFixedResourceConfigRows(rawRows string) string {
	rows, err := parseScriptResourceConfigRows(rawRows)
	if err != nil {
		return rawRows
	}
	kept := make([]scriptResourceConfigRow, 0, len(rows))
	changed := false
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Name), "ROLE") ||
			strings.EqualFold(strings.TrimSpace(row.Placeholder), "配置用途") {
			changed = true
			continue
		}
		kept = append(kept, row)
	}
	if !changed {
		return rawRows
	}
	data, err := json.Marshal(kept)
	if err != nil {
		return rawRows
	}
	return string(data)
}

func (s *ScriptManagerService) GetResourceConfig(id uint, userID uint) (modelSystem.TbScriptResourceConfig, error) {
	var config modelSystem.TbScriptResourceConfig
	err := global.GVA_DB.Where("id = ? AND user_id = ?", id, userID).First(&config).Error
	return config, err
}

func (s *ScriptManagerService) DeleteResourceConfig(id uint, userID uint) error {
	config, err := s.GetResourceConfig(id, userID)
	if err != nil {
		return err
	}
	return global.GVA_DB.Delete(&config).Error
}

func (s *ScriptManagerService) ListExecutions(req systemReq.ScriptExecutionSearch, userID uint) ([]modelSystem.TbScriptExecution, int64, error) {
	db := global.GVA_DB.Model(&modelSystem.TbScriptExecution{}).Where("user_id = ?", userID)
	if req.WorkflowId != 0 {
		db = db.Where("workflow_id = ?", req.WorkflowId)
	}
	if req.StepId != 0 {
		db = db.Where("step_id = ?", req.StepId)
	}
	if req.Scope != "" {
		db = db.Where("scope = ?", req.Scope)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []modelSystem.TbScriptExecution
	err := db.Scopes(req.Paginate()).Order("id DESC").Find(&list).Error
	return list, total, err
}

func (s *ScriptManagerService) GetExecutionLog(id uint, userID uint) (modelSystem.TbScriptExecution, error) {
	var execution modelSystem.TbScriptExecution
	err := global.GVA_DB.Where("id = ? AND user_id = ?", id, userID).First(&execution).Error
	return execution, err
}

func (s *ScriptManagerService) ExecuteWorkflowWithLog(ctx context.Context, workflowID uint, userID uint, logCh chan string) error {
	workflow, err := s.GetWorkflow(workflowID, userID)
	if err != nil {
		return fmt.Errorf("获取脚本流程失败: %w", err)
	}
	now := time.Now()
	execution := modelSystem.TbScriptExecution{
		WorkflowId: workflow.ID,
		UserId:     userID,
		Scope:      scriptExecutionScopeWorkflow,
		Status:     scriptExecutionStatusRunning,
		StartedAt:  &now,
	}
	if err := global.GVA_DB.Create(&execution).Error; err != nil {
		return err
	}
	collector := newExecutionLogCollector(logCh)
	collector.send(fmt.Sprintf("执行记录 #%d", execution.ID))
	collector.send(fmt.Sprintf("开始执行流程: %s", workflow.WorkflowName))

	status := scriptExecutionStatusSuccess
	var runErr error
	for _, step := range workflow.Steps {
		collector.send(fmt.Sprintf("开始步骤: [%s] %s", scriptStepTypeLabel(step.StepType), step.StepName))
		err := s.executeStepContent(ctx, step, userID, collector)
		if err != nil {
			status = scriptExecutionStatusFailed
			runErr = fmt.Errorf("步骤「%s」执行失败: %w", step.StepName, err)
			collector.send(runErr.Error())
			break
		}
		collector.send(fmt.Sprintf("步骤完成: %s", step.StepName))
	}

	finishedAt := time.Now()
	execution.Status = status
	execution.FinishedAt = &finishedAt
	execution.DurationMs = finishedAt.Sub(now).Milliseconds()
	execution.LogText = collector.text()
	if runErr != nil {
		execution.ErrorMessage = runErr.Error()
	}
	_ = global.GVA_DB.Save(&execution).Error
	_ = global.GVA_DB.Model(&modelSystem.TbScriptWorkflow{}).
		Where("id = ?", workflow.ID).
		Updates(map[string]interface{}{"last_status": status, "last_run_at": &finishedAt}).Error
	return runErr
}

func (s *ScriptManagerService) ExecuteStepWithLog(ctx context.Context, stepID uint, userID uint, logCh chan string) error {
	step, err := s.GetStep(stepID, userID)
	if err != nil {
		return fmt.Errorf("获取脚本步骤失败: %w", err)
	}
	now := time.Now()
	execution := modelSystem.TbScriptExecution{
		WorkflowId: step.WorkflowId,
		StepId:     step.ID,
		UserId:     userID,
		Scope:      scriptExecutionScopeStep,
		Status:     scriptExecutionStatusRunning,
		StartedAt:  &now,
	}
	if err := global.GVA_DB.Create(&execution).Error; err != nil {
		return err
	}
	collector := newExecutionLogCollector(logCh)
	collector.send(fmt.Sprintf("执行记录 #%d", execution.ID))
	collector.send(fmt.Sprintf("开始执行步骤: [%s] %s", scriptStepTypeLabel(step.StepType), step.StepName))

	runErr := s.executeStepContent(ctx, step, userID, collector)
	status := scriptExecutionStatusSuccess
	if runErr != nil {
		status = scriptExecutionStatusFailed
		collector.send(fmt.Sprintf("步骤执行失败: %v", runErr))
	} else {
		collector.send("步骤执行完成")
	}

	finishedAt := time.Now()
	execution.Status = status
	execution.FinishedAt = &finishedAt
	execution.DurationMs = finishedAt.Sub(now).Milliseconds()
	execution.LogText = collector.text()
	if runErr != nil {
		execution.ErrorMessage = runErr.Error()
	}
	_ = global.GVA_DB.Save(&execution).Error
	return runErr
}

func (s *ScriptManagerService) executeStepContent(ctx context.Context, step modelSystem.TbScriptStep, userID uint, collector *executionLogCollector) error {
	script := strings.TrimSpace(step.ScriptContent)
	if script == "" {
		return fmt.Errorf("脚本内容为空")
	}
	renderedScript, err := s.renderScriptWithPlaceholders(script, step.Placeholders, userID)
	if err != nil {
		return err
	}
	switch step.StepType {
	case ScriptStepTypeLocalExec, ScriptStepTypeLocalUpload:
		return runLocalScript(ctx, renderedScript, collector)
	case ScriptStepTypeTargetDownload, ScriptStepTypeTargetExec:
		server, err := s.resolveTargetServer(step, userID)
		if err != nil {
			return err
		}
		if isLocalScriptTarget(server) {
			return runLocalScript(ctx, renderedScript, collector)
		}
		return runRemoteScript(ctx, server, renderedScript, collector)
	default:
		return fmt.Errorf("不支持的步骤类型: %s", step.StepType)
	}
}

type scriptStepPlaceholder struct {
	Placeholder        string `json:"placeholder"`
	Name               string `json:"name"`
	ValueKind          string `json:"valueKind"`
	Value              string `json:"value"`
	ResourceCategoryId uint   `json:"resourceCategoryId"`
	ResourceConfigId   uint   `json:"resourceConfigId"`
	CustomValue        string `json:"customValue"`
}

type scriptResourceConfigRow struct {
	Name        string `json:"name"`
	Placeholder string `json:"placeholder"`
	Value       string `json:"value"`
}

func (s *ScriptManagerService) resolveTargetServer(step modelSystem.TbScriptStep, userID uint) (modelSystem.TbServer, error) {
	serverID, err := firstServerPlaceholderID(step.Placeholders)
	if err != nil {
		return modelSystem.TbServer{}, err
	}
	if serverID != 0 {
		var server modelSystem.TbServer
		if err := global.GVA_DB.Where("id = ?", serverID).First(&server).Error; err != nil {
			return server, fmt.Errorf("目标服务器不存在: %w", err)
		}
		return server, nil
	}
	server, ok, err := s.resourceTargetServer(step.Placeholders, userID)
	if err != nil {
		return server, err
	}
	if ok {
		return server, nil
	}
	return modelSystem.TbServer{}, fmt.Errorf("目标步骤必须选择服务器占位符或服务器资源配置")
}

func firstServerPlaceholderID(rawPlaceholders string) (uint, error) {
	rawPlaceholders = strings.TrimSpace(rawPlaceholders)
	if rawPlaceholders == "" {
		return 0, nil
	}
	var placeholders []scriptStepPlaceholder
	if err := json.Unmarshal([]byte(rawPlaceholders), &placeholders); err != nil {
		return 0, fmt.Errorf("占位符配置解析失败: %w", err)
	}
	for _, placeholder := range placeholders {
		if strings.TrimSpace(placeholder.ValueKind) != "server" {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(placeholder.Value))
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("占位符 %s 未选择有效服务器配置", placeholderLabel(placeholder))
		}
		return uint(id), nil
	}
	return 0, nil
}

func (s *ScriptManagerService) renderScriptWithPlaceholders(script string, rawPlaceholders string, userID uint) (string, error) {
	rawPlaceholders = strings.TrimSpace(rawPlaceholders)
	if rawPlaceholders == "" {
		return script, nil
	}

	var placeholders []scriptStepPlaceholder
	if err := json.Unmarshal([]byte(rawPlaceholders), &placeholders); err != nil {
		return "", fmt.Errorf("占位符配置解析失败: %w", err)
	}

	var exports []string
	rendered := script
	for index, placeholder := range placeholders {
		name := normalizePlaceholderName(placeholderEnvName(placeholder))
		if name == "" {
			name = fmt.Sprintf("RESOURCE_%d", index+1)
		}
		primary, envMap, err := s.resolvePlaceholder(placeholder, name, userID)
		if err != nil {
			return "", err
		}
		rendered = strings.ReplaceAll(rendered, "{{"+name+"}}", primary)
		rendered = strings.ReplaceAll(rendered, "{{ "+name+" }}", primary)
		for key, value := range envMap {
			exports = append(exports, "export "+key+"="+shellQuote(value))
		}
	}
	if len(exports) == 0 {
		return rendered, nil
	}
	return strings.Join(exports, "\n") + "\n\n" + rendered, nil
}

func (s *ScriptManagerService) resolvePlaceholder(placeholder scriptStepPlaceholder, normalizedName string, userID uint) (string, map[string]string, error) {
	valueKind := strings.TrimSpace(placeholder.ValueKind)
	value := strings.TrimSpace(placeholder.Value)
	if valueKind == "" {
		valueKind = "manual"
	}
	envMap := map[string]string{}
	switch valueKind {
	case "manual":
		envMap[normalizedName] = value
		return value, envMap, nil
	case "connection":
		id, err := strconv.Atoi(value)
		if err != nil || id <= 0 {
			return "", nil, fmt.Errorf("占位符 %s 未选择有效数据库配置", placeholderLabel(placeholder))
		}
		var conn modelSystem.TbConnection
		if err := global.GVA_DB.Where("id = ?", id).First(&conn).Error; err != nil {
			return "", nil, fmt.Errorf("占位符 %s 数据库配置不存在: %w", placeholderLabel(placeholder), err)
		}
		envMap[normalizedName] = conn.ConnectionName
		envMap[normalizedName+"_ID"] = strconv.Itoa(int(conn.ID))
		envMap[normalizedName+"_NAME"] = conn.ConnectionName
		envMap[normalizedName+"_TYPE"] = conn.ConnectionType
		envMap[normalizedName+"_HOST"] = conn.ConnectionUrl
		envMap[normalizedName+"_PORT"] = strconv.Itoa(conn.Port)
		envMap[normalizedName+"_DATABASE"] = conn.DatabaseName
		envMap[normalizedName+"_USER"] = conn.DbLoginName
		envMap[normalizedName+"_PASSWORD"] = conn.DbLoginPassword
		envMap[normalizedName+"_GROUP"] = conn.ConnectionGroup
		envMap[normalizedName+"_ENV"] = conn.EnvName
		return conn.ConnectionName, envMap, nil
	case "server":
		id, err := strconv.Atoi(value)
		if err != nil || id <= 0 {
			return "", nil, fmt.Errorf("占位符 %s 未选择有效服务器配置", placeholderLabel(placeholder))
		}
		var server modelSystem.TbServer
		if err := global.GVA_DB.Where("id = ?", id).First(&server).Error; err != nil {
			return "", nil, fmt.Errorf("占位符 %s 服务器配置不存在: %w", placeholderLabel(placeholder), err)
		}
		envMap[normalizedName] = server.ServerIp
		envMap[normalizedName+"_ID"] = strconv.Itoa(int(server.ID))
		envMap[normalizedName+"_NAME"] = server.ServerName
		envMap[normalizedName+"_IP"] = server.ServerIp
		envMap[normalizedName+"_INTERNAL_IP"] = server.ServerInternalIp
		envMap[normalizedName+"_USER"] = server.ServerLoginName
		envMap[normalizedName+"_PASSWORD"] = server.ServerLoginPassword
		envMap[normalizedName+"_PORT"] = strconv.Itoa(server.ServerLoginPort)
		return server.ServerIp, envMap, nil
	case "resource":
		return s.resolveResourcePlaceholder(placeholder, normalizedName, userID)
	default:
		return "", nil, fmt.Errorf("占位符 %s 不支持的值类型: %s", placeholderLabel(placeholder), valueKind)
	}
}

func (s *ScriptManagerService) resolveResourcePlaceholder(placeholder scriptStepPlaceholder, normalizedName string, userID uint) (string, map[string]string, error) {
	config, category, rows, err := s.resourceConfigRows(placeholder.ResourceConfigId, userID)
	if err != nil {
		return "", nil, err
	}
	envMap := map[string]string{}
	prefixes := resourceConfigEnvPrefixes(normalizedName, config, rows, placeholder.CustomValue)
	for _, row := range rows {
		key := normalizePlaceholderName(row.Name)
		if key == "" {
			key = normalizePlaceholderName(row.Placeholder)
		}
		if key != "" {
			envMap[key] = row.Value
			for _, prefix := range prefixes {
				if prefix != "" && key != prefix && !strings.HasPrefix(key, prefix+"_") {
					envMap[prefix+"_"+key] = row.Value
				}
			}
		}
	}
	primary := config.ConfigName
	if category.CategoryType == ScriptResourceCategoryTypeDynamic && strings.TrimSpace(placeholder.CustomValue) != "" {
		primary = strings.TrimSpace(placeholder.CustomValue)
	} else if value, ok := envMap[normalizedName]; ok {
		primary = value
	}
	for _, prefix := range prefixes {
		if prefix != "" {
			envMap[prefix] = primary
		}
	}
	return primary, envMap, nil
}

func resourceConfigEnvPrefixes(normalizedName string, config modelSystem.TbScriptResourceConfig, rows []scriptResourceConfigRow, resourceRole string) []string {
	prefixes := make([]string, 0, 3)
	addPrefix := func(value string) {
		value = normalizePlaceholderName(value)
		if value == "" {
			return
		}
		for _, existing := range prefixes {
			if existing == value {
				return
			}
		}
		prefixes = append(prefixes, value)
	}

	configPrefix := resourceConfigIdentifierPrefix(config, rows)
	if configPrefix != "" {
		addPrefix(configPrefix)
	} else {
		legacyPrefix := legacyResourceConfigIdentifierPrefix(config.ConfigName, rows)
		addPrefix(legacyPrefix)
	}
	for _, rolePrefix := range resourceConfigRolePrefixes(resourceRole, rows) {
		addPrefix(rolePrefix)
	}
	addPrefix(normalizedName)
	return prefixes
}

func resourceConfigRolePrefixes(role string, rows []scriptResourceConfigRow) []string {
	role = normalizeResourceConfigRole(role)
	if role == "" {
		return nil
	}
	hasDatabaseType := resourceConfigRowValue(rows, "TYPE") != ""
	hasServerHost := resourceConfigRowValue(rows, "IP") != ""
	switch role {
	case "SOURCE", "SOURCE_DB", "SOURCE_SERVER":
		if hasDatabaseType {
			return []string{"SOURCE_DB"}
		}
		if hasServerHost {
			return []string{"SOURCE_SERVER"}
		}
	case "TARGET", "TARGET_DB", "TARGET_SERVER":
		if hasDatabaseType {
			return []string{"TARGET_DB"}
		}
		if hasServerHost {
			return []string{"TARGET_SERVER"}
		}
	}
	return nil
}

func normalizeResourceConfigRole(role string) string {
	return normalizePlaceholderName(role)
}

func resourceConfigIdentifierPrefix(config modelSystem.TbScriptResourceConfig, rows []scriptResourceConfigRow) string {
	if prefix := normalizePlaceholderName(config.PlaceholderKey); prefix != "" {
		return prefix
	}
	parentKey, childKey, ok := resourceConfigIdentityFromRows(rows)
	if ok {
		return normalizePlaceholderName(parentKey + "_" + childKey)
	}
	return ""
}

func legacyResourceConfigIdentifierPrefix(configName string, rows []scriptResourceConfigRow) string {
	for _, part := range strings.Split(configName, "/") {
		if prefix := normalizePlaceholderName(part); prefix != "" {
			return prefix
		}
		break
	}
	if host := resourceConfigRowValue(rows, "HOST"); host != "" {
		return normalizePlaceholderName("IP_" + host)
	}
	return ""
}

func resourceConfigRowValue(rows []scriptResourceConfigRow, name string) string {
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Name), name) {
			return strings.TrimSpace(row.Value)
		}
	}
	return ""
}

func resourceConfigIdentityFromRows(rows []scriptResourceConfigRow) (string, string, bool) {
	parentKey := normalizePlaceholderName(resourceConfigRowValue(rows, "PARENT_KEY"))
	childKey := normalizePlaceholderName(resourceConfigRowValue(rows, "CHILD_KEY"))
	if parentKey == "" || childKey == "" {
		return "", "", false
	}
	return parentKey, childKey, true
}

func parseScriptResourceConfigRows(rawRows string) ([]scriptResourceConfigRow, error) {
	if strings.TrimSpace(rawRows) == "" {
		return nil, nil
	}
	var rows []scriptResourceConfigRow
	if err := json.Unmarshal([]byte(rawRows), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *ScriptManagerService) validateResourceConfigIdentity(config modelSystem.TbScriptResourceConfig, category modelSystem.TbScriptResourceCategory, userID uint) error {
	if category.CategoryType != ScriptResourceCategoryTypeFixed {
		return nil
	}
	placeholderKey := normalizePlaceholderName(config.PlaceholderKey)
	if placeholderKey == "" {
		return nil
	}

	var configs []modelSystem.TbScriptResourceConfig
	if err := global.GVA_DB.
		Where("user_id = ? AND id <> ?", userID, config.ID).
		Find(&configs).Error; err != nil {
		return err
	}
	for _, existing := range configs {
		existingKey := normalizePlaceholderName(existing.PlaceholderKey)
		if existingKey == "" {
			existingRows, err := parseScriptResourceConfigRows(existing.Rows)
			if err == nil {
				parentKey, childKey, ok := resourceConfigIdentityFromRows(existingRows)
				if ok {
					existingKey = normalizePlaceholderName(parentKey + "_" + childKey)
				}
			}
		}
		if existingKey == placeholderKey {
			return fmt.Errorf("占位符标识已存在: %s（%s）", placeholderKey, existing.ConfigName)
		}
	}
	return nil
}

func (s *ScriptManagerService) resourceConfigRows(configID uint, userID uint) (modelSystem.TbScriptResourceConfig, modelSystem.TbScriptResourceCategory, []scriptResourceConfigRow, error) {
	if configID == 0 {
		return modelSystem.TbScriptResourceConfig{}, modelSystem.TbScriptResourceCategory{}, nil, fmt.Errorf("资源配置不能为空")
	}
	var config modelSystem.TbScriptResourceConfig
	if err := global.GVA_DB.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error; err != nil {
		return config, modelSystem.TbScriptResourceCategory{}, nil, fmt.Errorf("资源配置不存在: %w", err)
	}
	var category modelSystem.TbScriptResourceCategory
	if err := global.GVA_DB.Where("id = ? AND user_id = ?", config.CategoryId, userID).First(&category).Error; err != nil {
		return config, category, nil, fmt.Errorf("资源分类不存在: %w", err)
	}
	rows, err := parseScriptResourceConfigRows(config.Rows)
	if err != nil {
		return config, category, nil, fmt.Errorf("资源配置行解析失败: %w", err)
	}
	return config, category, rows, nil
}

func (s *ScriptManagerService) resourceTargetServer(rawPlaceholders string, userID uint) (modelSystem.TbServer, bool, error) {
	var placeholders []scriptStepPlaceholder
	if strings.TrimSpace(rawPlaceholders) == "" {
		return modelSystem.TbServer{}, false, nil
	}
	if err := json.Unmarshal([]byte(rawPlaceholders), &placeholders); err != nil {
		return modelSystem.TbServer{}, false, fmt.Errorf("占位符配置解析失败: %w", err)
	}
	for _, placeholder := range placeholders {
		if strings.TrimSpace(placeholder.ValueKind) != "resource" || placeholder.ResourceConfigId == 0 {
			continue
		}
		_, _, rows, err := s.resourceConfigRows(placeholder.ResourceConfigId, userID)
		if err != nil {
			return modelSystem.TbServer{}, false, err
		}
		if resourceConfigRowValue(rows, "TYPE") != "" {
			continue
		}
		server, ok := serverFromResourceRows(rows)
		if ok {
			return server, true, nil
		}
	}
	return modelSystem.TbServer{}, false, nil
}

func serverFromResourceRows(rows []scriptResourceConfigRow) (modelSystem.TbServer, bool) {
	values := map[string]string{}
	for _, row := range rows {
		key := normalizePlaceholderName(row.Name)
		if key == "" {
			key = normalizePlaceholderName(row.Placeholder)
		}
		if key != "" {
			values[key] = row.Value
		}
	}
	server := modelSystem.TbServer{
		ServerIp:            firstNonEmpty(values, "TARGET_SERVER_IP", "SERVER_IP", "HOST", "IP"),
		ServerInternalIp:    firstNonEmpty(values, "TARGET_SERVER_INTERNAL_IP", "SERVER_INTERNAL_IP", "INTERNAL_IP"),
		ServerLoginName:     firstNonEmpty(values, "TARGET_SERVER_USER", "SERVER_USER", "USER", "USERNAME", "LOGIN_NAME"),
		ServerLoginPassword: firstNonEmpty(values, "TARGET_SERVER_PASSWORD", "SERVER_PASSWORD", "PASSWORD"),
	}
	if portText := firstNonEmpty(values, "TARGET_SERVER_PORT", "SERVER_PORT", "PORT"); portText != "" {
		if port, err := strconv.Atoi(portText); err == nil {
			server.ServerLoginPort = port
		}
	}
	return server, strings.TrimSpace(server.ServerIp) != "" && strings.TrimSpace(server.ServerLoginName) != ""
}

func firstNonEmpty(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if strings.TrimSpace(values[key]) != "" {
			return strings.TrimSpace(values[key])
		}
	}
	return ""
}

func placeholderEnvName(placeholder scriptStepPlaceholder) string {
	for _, value := range []string{placeholder.Name, placeholder.Placeholder} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func placeholderLabel(placeholder scriptStepPlaceholder) string {
	for _, value := range []string{placeholder.Placeholder, placeholder.Name} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "未命名"
}

func normalizePlaceholderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func isValidScriptStepType(stepType string) bool {
	switch stepType {
	case ScriptStepTypeLocalExec, ScriptStepTypeLocalUpload, ScriptStepTypeTargetDownload, ScriptStepTypeTargetExec:
		return true
	default:
		return false
	}
}

func isValidScriptResourceCategoryType(categoryType string) bool {
	switch categoryType {
	case ScriptResourceCategoryTypeFixed, ScriptResourceCategoryTypeDynamic, ScriptResourceCategoryTypeConstant:
		return true
	default:
		return false
	}
}

func scriptStepTypeLabel(stepType string) string {
	switch stepType {
	case ScriptStepTypeLocalExec:
		return "本地执行"
	case ScriptStepTypeLocalUpload:
		return "本地上传"
	case ScriptStepTypeTargetDownload:
		return "目标下载"
	case ScriptStepTypeTargetExec:
		return "目标执行"
	default:
		return stepType
	}
}

type executionLogCollector struct {
	mu    sync.Mutex
	lines []string
	ch    chan string
}

func newExecutionLogCollector(ch chan string) *executionLogCollector {
	return &executionLogCollector{ch: ch}
}

func (c *executionLogCollector) send(line string) {
	c.mu.Lock()
	c.lines = append(c.lines, line)
	c.mu.Unlock()
	if c.ch != nil {
		select {
		case c.ch <- line:
		default:
		}
	}
}

func (c *executionLogCollector) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.lines, "\n")
}

func runLocalScript(ctx context.Context, script string, collector *executionLogCollector) error {
	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	cmd.Env = os.Environ()
	return runCommandPipes(cmd, collector)
}

func isLocalScriptTarget(server modelSystem.TbServer) bool {
	host := strings.TrimSpace(server.ServerIp)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.IsLoopback() {
			return true
		}
	}
	return false
}

func runRemoteScript(ctx context.Context, server modelSystem.TbServer, script string, collector *executionLogCollector) error {
	port := server.ServerLoginPort
	if port == 0 {
		port = 22
	}
	config := &ssh.ClientConfig{
		User: server.ServerLoginName,
		Auth: []ssh.AuthMethod{
			ssh.Password(server.ServerLoginPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(server.ServerIp, strconv.Itoa(port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	command := "bash -lc " + shellQuote(script)

	var wg sync.WaitGroup
	wg.Add(2)
	go scanCommandOutput(stdout, collector, &wg)
	go scanCommandOutput(stderr, collector, &wg)

	if err := session.Start(command); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- session.Wait()
	}()
	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		wg.Wait()
		return ctx.Err()
	case err := <-waitCh:
		wg.Wait()
		if err != nil {
			return fmt.Errorf("远程脚本执行失败: %w", err)
		}
		return nil
	}
}

func runCommandPipes(cmd *exec.Cmd, collector *executionLogCollector) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go scanCommandOutput(stdout, collector, &wg)
	go scanCommandOutput(stderr, collector, &wg)

	if err := cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		return fmt.Errorf("本地脚本执行失败: %w", err)
	}
	return nil
}

func scanCommandOutput(reader interface{ Read([]byte) (int, error) }, collector *executionLogCollector, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		collector.send(scanner.Text())
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
