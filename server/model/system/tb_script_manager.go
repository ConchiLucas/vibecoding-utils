package system

import (
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
)

// TbScriptCategory 脚本库业务分类
type TbScriptCategory struct {
	global.GVA_MODEL
	CategoryName string `json:"categoryName" form:"categoryName" gorm:"column:category_name;comment:分类名称"`
	Description  string `json:"description" form:"description" gorm:"column:description;type:text;comment:分类描述"`
	UserId       uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`
}

func (TbScriptCategory) TableName() string {
	return "tb_script_category"
}

// TbScriptWorkflow 脚本库工作流卡片
type TbScriptWorkflow struct {
	global.GVA_MODEL
	CategoryId   uint           `json:"categoryId" form:"categoryId" gorm:"column:category_id;comment:分类ID"`
	WorkflowName string         `json:"workflowName" form:"workflowName" gorm:"column:workflow_name;comment:工作流名称"`
	Description  string         `json:"description" form:"description" gorm:"column:description;type:text;comment:工作流描述"`
	UserId       uint           `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`
	LastStatus   string         `json:"lastStatus" form:"lastStatus" gorm:"column:last_status;comment:最近执行状态"`
	LastRunAt    *time.Time     `json:"lastRunAt" form:"lastRunAt" gorm:"column:last_run_at;comment:最近执行时间"`
	Steps        []TbScriptStep `json:"steps" gorm:"foreignKey:WorkflowId"`
}

func (TbScriptWorkflow) TableName() string {
	return "tb_script_workflow"
}

// TbScriptStep 脚本库工作流步骤
type TbScriptStep struct {
	global.GVA_MODEL
	WorkflowId    uint   `json:"workflowId" form:"workflowId" gorm:"column:workflow_id;comment:工作流ID"`
	StepName      string `json:"stepName" form:"stepName" gorm:"column:step_name;comment:步骤名称"`
	StepType      string `json:"stepType" form:"stepType" gorm:"column:step_type;comment:步骤类型(local_exec/local_upload/target_download/target_exec)"`
	ScriptContent string `json:"scriptContent" form:"scriptContent" gorm:"column:script_content;type:text;comment:脚本内容"`
	Placeholders  string `json:"placeholders" form:"placeholders" gorm:"column:placeholders;type:text;comment:占位符配置JSON"`
}

func (TbScriptStep) TableName() string {
	return "tb_script_step"
}

// TbScriptResourceCategory 脚本库资源配置业务分类
type TbScriptResourceCategory struct {
	global.GVA_MODEL
	CategoryName string                   `json:"categoryName" form:"categoryName" gorm:"column:category_name;comment:分类名称"`
	CategoryType string                   `json:"categoryType" form:"categoryType" gorm:"column:category_type;comment:分类类型(fixed/dynamic/constant)"`
	UserId       uint                     `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`
	Configs      []TbScriptResourceConfig `json:"configs" gorm:"foreignKey:CategoryId"`
}

func (TbScriptResourceCategory) TableName() string {
	return "tb_script_resource_category"
}

// TbScriptResourceConfig 脚本库资源配置卡片
type TbScriptResourceConfig struct {
	global.GVA_MODEL
	CategoryId     uint   `json:"categoryId" form:"categoryId" gorm:"column:category_id;comment:资源分类ID"`
	WorkflowId     uint   `json:"workflowId" form:"workflowId" gorm:"column:workflow_id;default:0;comment:所属脚本流程ID，0表示全局配置"`
	ConfigName     string `json:"configName" form:"configName" gorm:"column:config_name;comment:配置名称"`
	PlaceholderKey string `json:"placeholderKey" form:"placeholderKey" gorm:"column:placeholder_key;comment:配置占位符标识"`
	Rows           string `json:"rows" form:"rows" gorm:"column:rows;type:text;comment:配置行JSON"`
	UserId         uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`
}

func (TbScriptResourceConfig) TableName() string {
	return "tb_script_resource_config"
}

// TbScriptExecution 脚本库执行记录
type TbScriptExecution struct {
	global.GVA_MODEL
	WorkflowId   uint       `json:"workflowId" form:"workflowId" gorm:"column:workflow_id;comment:工作流ID"`
	StepId       uint       `json:"stepId" form:"stepId" gorm:"column:step_id;default:0;comment:步骤ID"`
	UserId       uint       `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`
	Scope        string     `json:"scope" form:"scope" gorm:"column:scope;comment:执行范围(workflow/step)"`
	Status       string     `json:"status" form:"status" gorm:"column:status;comment:执行状态(running/success/failed)"`
	LogText      string     `json:"logText" form:"logText" gorm:"column:log_text;type:text;comment:执行日志"`
	ErrorMessage string     `json:"errorMessage" form:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
	StartedAt    *time.Time `json:"startedAt" form:"startedAt" gorm:"column:started_at;comment:开始时间"`
	FinishedAt   *time.Time `json:"finishedAt" form:"finishedAt" gorm:"column:finished_at;comment:结束时间"`
	DurationMs   int64      `json:"durationMs" form:"durationMs" gorm:"column:duration_ms;default:0;comment:耗时毫秒"`
}

func (TbScriptExecution) TableName() string {
	return "tb_script_execution"
}
