package system

import api "github.com/flipped-aurora/easy-deploy/server/api/v1"

type RouterGroup struct {
	BaseRouter
	InitRouter
	UserRouter
	ServerRouter
	ProjectRouter
	ProjectScriptRouter
	ScriptManagerRouter
	ProjectRouteRouter
	ProjectGroupRouter
	TbDictDataRouter
	TbInterfaceServerRouter
	TbInterfaceEnvRouter
	TbInterfaceRouter
	TbConnectionRouter
	TbTableRouter
	TbTableColumnRouter
	TbTableRelateRouter
	TbEntityRouter
	TbColumnRouter
	TbClientRouter
	TbInterfaceParamsRouter
	TbInterfaceLogRouter
	TbAgileRequestRouter
	TbAgileTableSampleRouter
	TbTablePreferRouter
	TbInterfaceServerUserRouter
	TbInterfaceProjectRouter
	TbGeneratePlaceHolderRouter
	TbGenerateProjectRouter
	TbGenerateProjectPathRouter
	TbGenerateProjectPathModelRouter
	TbGenerateProjectPlaceHolderRouter
	TbGenerateRecordRouter
	AIChatRouter
	AIChatHistoryRouter
}

var (
	dbApi                           = api.ApiGroupApp.SystemApiGroup.DBApi
	baseApi                         = api.ApiGroupApp.SystemApiGroup.BaseApi
	serverApi                       = api.ApiGroupApp.SystemApiGroup.ServerApi
	projectApi                      = api.ApiGroupApp.SystemApiGroup.ProjectApi
	scriptApi                       = api.ApiGroupApp.SystemApiGroup.ProjectScriptApi
	scriptManagerApi                = api.ApiGroupApp.SystemApiGroup.ScriptManagerApi
	projectRouteApi                 = api.ApiGroupApp.SystemApiGroup.ProjectRouteApi
	projectGroupApi                 = api.ApiGroupApp.SystemApiGroup.ProjectGroupApi
	tbDictDataApi                   = api.ApiGroupApp.SystemApiGroup.TbDictDataApi
	tbInterfaceServerApi            = api.ApiGroupApp.SystemApiGroup.TbInterfaceServerApi
	tbInterfaceEnvApi               = api.ApiGroupApp.SystemApiGroup.TbInterfaceEnvApi
	tbInterfaceApi                  = api.ApiGroupApp.SystemApiGroup.TbInterfaceApi
	tbConnectionApi                 = api.ApiGroupApp.SystemApiGroup.TbConnectionApi
	tbTableApi                      = api.ApiGroupApp.SystemApiGroup.TbTableApi
	tbTableColumnApi                = api.ApiGroupApp.SystemApiGroup.TbTableColumnApi
	tbTableRelateApi                = api.ApiGroupApp.SystemApiGroup.TbTableRelateApi
	tbEntityApi                     = api.ApiGroupApp.SystemApiGroup.TbEntityApi
	tbColumnApi                     = api.ApiGroupApp.SystemApiGroup.TbColumnApi
	tbClientApi                     = api.ApiGroupApp.SystemApiGroup.TbClientApi
	tbInterfaceParamsApi            = api.ApiGroupApp.SystemApiGroup.TbInterfaceParamsApi
	tbInterfaceLogApi               = api.ApiGroupApp.SystemApiGroup.TbInterfaceLogApi
	tbAgileRequestApi               = api.ApiGroupApp.SystemApiGroup.TbAgileRequestApi
	tbAgileTableSampleApi           = api.ApiGroupApp.SystemApiGroup.TbAgileTableSampleApi
	tbTablePreferApi                = api.ApiGroupApp.SystemApiGroup.TbTablePreferApi
	tbInterfaceServerUserApi        = api.ApiGroupApp.SystemApiGroup.TbInterfaceServerUserApi
	tbInterfaceProjectApi           = api.ApiGroupApp.SystemApiGroup.TbInterfaceProjectApi
	tbGeneratePlaceHolderApi        = api.ApiGroupApp.SystemApiGroup.TbGeneratePlaceHolderApi
	tbGenerateProjectApi            = api.ApiGroupApp.SystemApiGroup.TbGenerateProjectApi
	tbGenerateProjectPathApi        = api.ApiGroupApp.SystemApiGroup.TbGenerateProjectPathApi
	tbGenerateProjectPathModelApi   = api.ApiGroupApp.SystemApiGroup.TbGenerateProjectPathModelApi
	tbGenerateProjectPlaceHolderApi = api.ApiGroupApp.SystemApiGroup.TbGenerateProjectPlaceHolderApi
	tbGenerateRecordApi             = api.ApiGroupApp.SystemApiGroup.TbGenerateRecordApi
	aiChatApi                       = api.ApiGroupApp.SystemApiGroup.AIChatApi
	aiChatHistoryApi                = api.ApiGroupApp.SystemApiGroup.AIChatHistoryApi
)
