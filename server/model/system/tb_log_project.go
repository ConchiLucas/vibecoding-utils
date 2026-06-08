package system

import "github.com/flipped-aurora/easy-deploy/server/global"

// TbLogProjectGroup 日志管理项目组
type TbLogProjectGroup struct {
	global.GVA_MODEL
	GroupName string `json:"groupName" form:"groupName" gorm:"column:group_name;type:varchar(128);comment:项目组名称"`
	Sort      int    `json:"sort" form:"sort" gorm:"column:sort;comment:排序"`
	UserId    uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`
}

func (TbLogProjectGroup) TableName() string {
	return "tb_log_project_group"
}

// TbLogProject 日志管理项目
type TbLogProject struct {
	global.GVA_MODEL
	GroupId           uint   `json:"groupId" form:"groupId" gorm:"column:group_id;comment:日志项目组ID"`
	ProjectConfigId   uint   `json:"projectConfigId" form:"projectConfigId" gorm:"column:project_config_id;comment:所属项目配置ID"`
	ProjectConfigName string `json:"projectConfigName" form:"projectConfigName" gorm:"column:project_config_name;type:varchar(255);comment:所属项目配置名称"`
	ComputerLanguage  string `json:"computerLanguage" form:"computerLanguage" gorm:"column:computer_language;type:varchar(64);comment:语言类型"`
	ProjectName       string `json:"projectName" form:"projectName" gorm:"column:project_name;type:varchar(255);comment:项目名称"`
	Description       string `json:"description" form:"description" gorm:"column:description;type:text;comment:项目描述"`
	LocalProjectPath  string `json:"localProjectPath" form:"localProjectPath" gorm:"column:local_project_path;type:varchar(512);comment:本地项目路径"`
	UserId            uint   `json:"userId" form:"userId" gorm:"column:user_id;comment:归属用户ID"`

	Routes []TbLogProjectRoute `json:"routes" gorm:"foreignKey:ProjectId;references:ID"`
}

func (TbLogProject) TableName() string {
	return "tb_log_project"
}

// TbLogProjectRoute 日志管理服务路线
type TbLogProjectRoute struct {
	global.GVA_MODEL
	ProjectId           int    `json:"projectId" form:"projectId" gorm:"column:project_id;comment:日志项目ID"`
	RouteKey            string `json:"routeKey" form:"routeKey" gorm:"column:route_key;type:varchar(128);comment:路线标识"`
	RouteName           string `json:"routeName" form:"routeName" gorm:"column:route_name;type:varchar(255);comment:路线名称"`
	LocalProjectPath    string `json:"localProjectPath" form:"localProjectPath" gorm:"column:local_project_path;type:varchar(512);comment:本地项目路径"`
	LocalExecuteCommand string `json:"localExecuteCommand" form:"localExecuteCommand" gorm:"column:local_execute_command;type:varchar(1024);comment:启动命令"`
	LocalStartCommand   string `json:"localStartCommand" form:"localStartCommand" gorm:"column:local_start_command;type:varchar(1024);comment:附加启动命令"`
	LocalStopCommand    string `json:"localStopCommand" form:"localStopCommand" gorm:"column:local_stop_command;type:varchar(1024);comment:关闭命令"`
	BuildType           string `json:"buildType" form:"buildType" gorm:"column:build_type;type:varchar(128);comment:构建模式"`
	DockerComposeDeploy bool   `json:"dockerComposeDeploy" form:"dockerComposeDeploy" gorm:"column:docker_compose_deploy;comment:是否docker-compose发布"`
	Color               string `json:"color" form:"color" gorm:"column:color;type:varchar(255);comment:按钮展示色"`
	Icon                string `json:"icon" form:"icon" gorm:"column:icon;type:varchar(64);comment:图标"`
	Sort                int    `json:"sort" form:"sort" gorm:"column:sort;comment:排序"`
}

func (TbLogProjectRoute) TableName() string {
	return "tb_log_project_route"
}
