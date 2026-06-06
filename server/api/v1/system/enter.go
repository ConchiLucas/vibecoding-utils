package system

import "github.com/flipped-aurora/easy-deploy/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
	ServerApi
	ProjectApi
	ProjectScriptApi
	ScriptManagerApi
	ProjectRouteApi
	ProjectGroupApi
	TbDictDataApi
	TbInterfaceServerApi
	TbInterfaceEnvApi
	TbInterfaceApi
	TbConnectionApi
	TbTableApi
	TbTableColumnApi
	TbTableRelateApi
	TbEntityApi
	TbColumnApi
	TbClientApi
	TbInterfaceParamsApi
	TbInterfaceLogApi
	TbAgileRequestApi
	TbAgileTableSampleApi
	TbTablePreferApi
	TbInterfaceServerUserApi
	TbInterfaceProjectApi
	TbGenerateProjectApi
	TbGenerateProjectInstanceApi
	TbGenerateDbTemplateTypeApi
	TbGenerateDbTemplateScriptApi
	TbGenerateProjectPathApi
	TbGenerateProjectPathModelApi
	AIChatApi
	AIChatHistoryApi
}

var (
	userService                       = service.ServiceGroupApp.SystemServiceGroup.UserService
	initDBService                     = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	serverService                     = service.ServiceGroupApp.SystemServiceGroup.ServerService
	projectService                    = service.ServiceGroupApp.SystemServiceGroup.ProjectService
	projectScriptService              = service.ServiceGroupApp.SystemServiceGroup.ProjectScriptService
	scriptManagerService              = service.ServiceGroupApp.SystemServiceGroup.ScriptManagerService
	deployService                     = service.ServiceGroupApp.SystemServiceGroup.DeployService
	projectRouteService               = service.ServiceGroupApp.SystemServiceGroup.ProjectRouteService
	projectGroupService               = service.ServiceGroupApp.SystemServiceGroup.ProjectGroupService
	tbDictDataService                 = service.ServiceGroupApp.SystemServiceGroup.TbDictDataService
	tbInterfaceServerService          = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceServerService
	tbInterfaceEnvService             = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceEnvService
	tbInterfaceService                = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceService
	tbConnectionService               = service.ServiceGroupApp.SystemServiceGroup.TbConnectionService
	tbTableService                    = service.ServiceGroupApp.SystemServiceGroup.TbTableService
	tbTableColumnService              = service.ServiceGroupApp.SystemServiceGroup.TbTableColumnService
	tbTableRelateService              = service.ServiceGroupApp.SystemServiceGroup.TbTableRelateService
	tbEntityService                   = service.ServiceGroupApp.SystemServiceGroup.TbEntityService
	tbColumnService                   = service.ServiceGroupApp.SystemServiceGroup.TbColumnService
	tbClientService                   = service.ServiceGroupApp.SystemServiceGroup.TbClientService
	tbInterfaceParamsService          = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceParamsService
	tbInterfaceLogService             = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceLogService
	tbAgileRequestService             = service.ServiceGroupApp.SystemServiceGroup.TbAgileRequestService
	tbAgileTableSampleService         = service.ServiceGroupApp.SystemServiceGroup.TbAgileTableSampleService
	tbTablePreferService              = service.ServiceGroupApp.SystemServiceGroup.TbTablePreferService
	tbInterfaceServerUserService      = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceServerUserService
	tbInterfaceProjectService         = service.ServiceGroupApp.SystemServiceGroup.TbInterfaceProjectService
	tbGenerateProjectService          = service.ServiceGroupApp.SystemServiceGroup.TbGenerateProjectService
	tbGenerateProjectInstanceService  = service.ServiceGroupApp.SystemServiceGroup.TbGenerateProjectInstanceService
	tbGenerateDbTemplateTypeService   = service.ServiceGroupApp.SystemServiceGroup.TbGenerateDbTemplateTypeService
	tbGenerateDbTemplateScriptService = service.ServiceGroupApp.SystemServiceGroup.TbGenerateDbTemplateScriptService
	tbGenerateProjectPathService      = service.ServiceGroupApp.SystemServiceGroup.TbGenerateProjectPathService
	tbGenerateProjectPathModelService = service.ServiceGroupApp.SystemServiceGroup.TbGenerateProjectPathModelService
	aiChatService                     = service.ServiceGroupApp.SystemServiceGroup.AIChatService
	aiChatHistoryService              = service.ServiceGroupApp.SystemServiceGroup.AIChatHistoryService
)
