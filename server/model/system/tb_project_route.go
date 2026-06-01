package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
)

// TbProjectRoute 项目部署路由配置表
type TbProjectRoute struct {
	global.GVA_MODEL
	ProjectId            int    `json:"projectId" form:"projectId" gorm:"column:project_id;comment:项目ID"`
	RouteKey             string `json:"routeKey" form:"routeKey" gorm:"column:route_key;comment:标识(如 local/server/etc)"`
	RouteName            string `json:"routeName" form:"routeName" gorm:"column:route_name;comment:配置名称"`
	ServerId             int    `json:"serverId" form:"serverId" gorm:"column:server_id;comment:特定服务器关联(0为主项目服务器)"`
	LocalProjectPath     string `json:"localProjectPath" form:"localProjectPath" gorm:"column:local_project_path;comment:本地项目路径"`
	ServerProjectPath    string `json:"serverProjectPath" form:"serverProjectPath" gorm:"column:server_project_path;comment:服务器项目路径"`
	LocalExecuteCommand  string `json:"localExecuteCommand" form:"localExecuteCommand" gorm:"column:local_execute_command;comment:本地打包指令"`
	LocalStopCommand     string `json:"localStopCommand" form:"localStopCommand" gorm:"column:local_stop_command;comment:本地关闭指令"`
	LocalStartCommand    string `json:"localStartCommand" form:"localStartCommand" gorm:"column:local_start_command;comment:本地运行指令"`
	ServerExecuteCommand  string `json:"serverExecuteCommand" form:"serverExecuteCommand" gorm:"column:server_execute_command;comment:远端执行指令"`
	Color                 string `json:"color" form:"color" gorm:"column:color;comment:按钮展示色"`
	Icon                  string `json:"icon" form:"icon" gorm:"column:icon;comment:按钮图标"`
	BuildImage            bool   `json:"buildImage" form:"buildImage" gorm:"column:build_image;comment:是否构建镜像"`
	BuildIncrementalImage bool   `json:"buildIncrementalImage" form:"buildIncrementalImage" gorm:"column:build_incremental_image;comment:是否构建增量镜像"`
	DockerComposeDeploy   bool   `json:"dockerComposeDeploy" form:"dockerComposeDeploy" gorm:"column:docker_compose_deploy;comment:是否docker-compose发布"`
	BuildType             string `json:"buildType" form:"buildType" gorm:"column:build_type;comment:构建模式下拉选"`
	FileName              string `json:"fileName" form:"fileName" gorm:"column:file_name;comment:环境专属压缩文件名"`
}

func (TbProjectRoute) TableName() string {
	return "tb_project_route"
}
