package request

type CopyGenerateProjectReq struct {
	SourceProjectId int    `json:"sourceProjectId" form:"sourceProjectId"`
	ProjectName     string `json:"projectName" form:"projectName"`
	BusinessType    string `json:"businessType" form:"businessType"`
	ProjectType     string `json:"projectType" form:"projectType"`
	Remark          string `json:"remark" form:"remark"`
	UserName        string `json:"userName" form:"userName"`
}
