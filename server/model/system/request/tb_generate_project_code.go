package request

type GenerateProjectCodeReq struct {
	TemplateProjectId int    `json:"templateProjectId"`
	ProjectInstanceId int    `json:"projectInstanceId"`
	PathSet           int    `json:"pathSet"`
	PathSetIdentity   string `json:"pathSetIdentity"`
	PathIds           []int  `json:"pathIds"`
	Module            string `json:"module"`
	TableName         string `json:"tableName"`
	Overwrite         bool   `json:"overwrite"`
}
