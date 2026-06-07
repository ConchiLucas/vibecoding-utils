package request

type CopyGenerateProjectPathSetReq struct {
	ProjectId         int    `json:"projectId"`
	ProjectInstanceId int    `json:"projectInstanceId"`
	PathSet           int    `json:"pathSet"`
	PathIds           []uint `json:"pathIds"`
	GroupIds          []uint `json:"groupIds"`
}

type DeleteGenerateProjectPathSetReq struct {
	ProjectId         int    `json:"projectId"`
	ProjectInstanceId int    `json:"projectInstanceId"`
	PathSet           int    `json:"pathSet"`
	PathIds           []uint `json:"pathIds"`
	GroupIds          []uint `json:"groupIds"`
}

type RenameGenerateProjectPathSetReq struct {
	ProjectId         int    `json:"projectId"`
	ProjectInstanceId int    `json:"projectInstanceId"`
	PathSet           int    `json:"pathSet"`
	PathIds           []uint `json:"pathIds"`
	GroupIds          []uint `json:"groupIds"`
	PathSetName       string `json:"pathSetName"`
}

type BuildGenerateProjectPromptSummaryReq struct {
	ProjectInstanceId int    `json:"projectInstanceId"`
	PathSet           int    `json:"pathSet"`
	PathIds           []uint `json:"pathIds"`
	Module            string `json:"module"`
	TableName         string `json:"tableName"`
	Overwrite         bool   `json:"overwrite"`
}
