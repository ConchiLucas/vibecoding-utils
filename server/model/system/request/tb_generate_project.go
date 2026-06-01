package request

type GlobalReplace struct {
	Id         int    `json:"id"`
	FormerStr  string `json:"formerStr"`
	ReplaceStr string `json:"replaceStr"`
}

type GenerateCodeModel struct {
	Id             string `json:"id"`
	ModuleName     string `json:"moduleName"`
	ModuleComment  string `json:"moduleComment"`
	TableStructure string `json:"tableStructure"`
	DbType         string `json:"dbType"`
}
