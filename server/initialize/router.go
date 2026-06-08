package initialize

import (
	"net/http"
	"os"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/middleware"
	"github.com/flipped-aurora/easy-deploy/server/router"
	"github.com/gin-gonic/gin"
)

type justFilesFilesystem struct {
	fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, os.ErrPermission
	}

	return f, nil
}

// 初始化总路由

func Routers() *gin.Engine {
	Router := gin.New()
	// 使用自定义的 Recovery 中间件，记录 panic 并入库
	Router.Use(middleware.GinRecovery(true))
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := router.RouterGroupApp.System

	// Router.Use(middleware.Cors()) // 直接放行全部跨域请求
	Router.Use(middleware.CorsByRules()) // 按照配置的规则放行跨域请求

	PublicGroup := Router.Group(global.GVA_CONFIG.System.RouterPrefix)
	PrivateGroup := Router.Group(global.GVA_CONFIG.System.RouterPrefix)

	PrivateGroup.Use(middleware.JWTAuth())

	{
		// 健康监测
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
	}
	{
		systemRouter.InitBaseRouter(PublicGroup)       // 注册基础功能路由 不做鉴权
		systemRouter.InitInitRouter(PublicGroup)       // 自动初始化相关
		systemRouter.InitAIProviderRouter(PublicGroup) // AI 厂商列表，不返回密钥
		systemRouter.InitTbGenerateProjectPublicRouter(PublicGroup)
	}

	{
		systemRouter.InitUserRouter(PrivateGroup)              // 注册用户路由
		systemRouter.InitServerRouter(PrivateGroup)            // 服务器管理
		systemRouter.InitProjectRouter(PrivateGroup)           // 项目管理
		systemRouter.InitProjectScriptRouter(PrivateGroup)     // 脚本管理
		systemRouter.InitScriptManagerRouter(PrivateGroup)     // 脚本库流程管理
		systemRouter.InitLogManagerRouter(PrivateGroup)        // 日志管理
		systemRouter.InitProjectRouteRouter(PrivateGroup)      // 项目路由配置
		systemRouter.InitProjectGroupRouter(PrivateGroup)      // 项目组管理
		systemRouter.InitTbDictDataRouter(PrivateGroup)        // 字典数据路由
		systemRouter.InitTbInterfaceServerRouter(PrivateGroup) // 服务管理路由
		systemRouter.InitTbInterfaceEnvRouter(PrivateGroup)    // 环境管理路由
		systemRouter.InitTbInterfaceRouter(PrivateGroup)       // 接口管理路由
		systemRouter.InitTbConnectionRouter(PrivateGroup)      // 数据库连接管理路由
		systemRouter.InitTbTableRouter(PrivateGroup)
		systemRouter.InitTbTableColumnRouter(PrivateGroup)
		systemRouter.InitTbTableRelateRouter(PrivateGroup)
		systemRouter.InitTbEntityRouter(PrivateGroup)
		systemRouter.InitTbColumnRouter(PrivateGroup)
		systemRouter.InitTbClientRouter(PrivateGroup)
		systemRouter.InitTbInterfaceParamsRouter(PrivateGroup)
		systemRouter.InitTbInterfaceLogRouter(PrivateGroup)
		systemRouter.InitTbAgileRequestRouter(PrivateGroup)
		systemRouter.InitTbAgileTableSampleRouter(PrivateGroup)
		systemRouter.InitTbTablePreferRouter(PrivateGroup)
		systemRouter.InitTbInterfaceServerUserRouter(PrivateGroup)
		systemRouter.InitTbInterfaceProjectRouter(PrivateGroup) // 项目配置路由
		systemRouter.InitTbGenerateProjectRouter(PrivateGroup)
		systemRouter.InitTbGenerateProjectInstanceRouter(PrivateGroup)
		systemRouter.InitTbGenerateDbTemplateRouter(PrivateGroup)
		systemRouter.InitTbGenerateProjectPathRouter(PrivateGroup)
		systemRouter.InitTbGenerateProjectPathModelRouter(PrivateGroup)
		systemRouter.InitAIChatRouter(PrivateGroup)        // AI 对话
		systemRouter.InitAIChatHistoryRouter(PrivateGroup) // AI 对话历史
		systemRouter.InitAIConfigRouter(PrivateGroup)      // AI 配置管理
	}

	// 注册业务路由
	initBizRouter(PrivateGroup, PublicGroup)

	global.GVA_ROUTERS = Router.Routes()

	// 注册嵌入的前端静态资源路由（必须在所有API路由之后，确保NoRoute不覆盖API）
	RegisterFrontendRoutes(Router)

	global.GVA_LOG.Info("router register success")
	return Router
}
