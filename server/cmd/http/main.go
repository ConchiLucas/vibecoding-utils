package main

import (
	"github.com/flipped-aurora/easy-deploy/server/core"
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/initialize"
	systemService "github.com/flipped-aurora/easy-deploy/server/service/system"
	"go.uber.org/zap"
)

// main starts the Gin HTTP server without the Wails desktop shell.
// Use this entrypoint for local web-react development.
func main() {
	global.GVA_VP = core.Viper()
	initialize.OtherInit()
	global.GVA_LOG = core.Zap()
	zap.ReplaceGlobals(global.GVA_LOG)
	global.GVA_DB = initialize.Gorm()
	initialize.DBList()
	initialize.SetupHandlers()
	systemService.SharedDockerNetworkServiceApp.EnsureOnStartup()
	if global.GVA_DB != nil {
		initialize.RegisterTables()
		systemService.ProjectGroupServiceApp.StartEnabledGroupsOnStartup()
	}
	core.RunServer()
}
