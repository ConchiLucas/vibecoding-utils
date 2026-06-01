package request

import "github.com/flipped-aurora/easy-deploy/server/model/common/request"

type ScriptWorkflowSearch struct {
	request.PageInfo
	CategoryId uint   `json:"categoryId" form:"categoryId"`
	Keyword    string `json:"keyword" form:"keyword"`
}

type ScriptExecutionSearch struct {
	request.PageInfo
	WorkflowId uint   `json:"workflowId" form:"workflowId"`
	StepId     uint   `json:"stepId" form:"stepId"`
	Scope      string `json:"scope" form:"scope"`
}
