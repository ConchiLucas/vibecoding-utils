package request

type GenerateFieldSnippetTemplate struct {
	Key          string `json:"key"`
	Description  string `json:"description"`
	Template     string `json:"template"`
	Separator    string `json:"separator"`
	ExcludeAudit bool   `json:"excludeAudit"`
}

type PreviewGenerateFieldSnippetReq struct {
	BusinessType string                         `json:"businessType" form:"businessType"`
	SourceText   string                         `json:"sourceText"`
	Snippets     []GenerateFieldSnippetTemplate `json:"snippets"`
}

type SaveGenerateFieldSnippetReq struct {
	BusinessType string                         `json:"businessType" form:"businessType"`
	Name         string                         `json:"name" form:"name"`
	SourceText   string                         `json:"sourceText"`
	Snippets     []GenerateFieldSnippetTemplate `json:"snippets"`
	UserName     string                         `json:"userName" form:"userName"`
}
