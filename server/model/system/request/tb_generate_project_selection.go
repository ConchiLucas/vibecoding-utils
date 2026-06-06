package request

type UpdateSelectedProjectInstanceReq struct {
	TemplateProjectId int `json:"templateProjectId"`
	ProjectInstanceId int `json:"projectInstanceId"`
}

type UpdateSelectedPathSetReq struct {
	ProjectInstanceId       int    `json:"projectInstanceId"`
	SelectedPathSetIdentity string `json:"selectedPathSetIdentity"`
}
