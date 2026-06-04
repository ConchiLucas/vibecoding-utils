package response

type GenerateCodeProjectResult struct {
	ProjectId      int      `json:"projectId"`
	ProjectName    string   `json:"projectName"`
	DiskPath       string   `json:"diskPath"`
	GeneratedFiles []string `json:"generatedFiles"`
}
