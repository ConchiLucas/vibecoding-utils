package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/core"
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/initialize"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

//go:embed config.template.yaml
var configBytes []byte

//go:embed frontend/dist/*
var frontendDist embed.FS

// init 初始化函数，用于将配置文件释放到用户目录解决打包后找不到配置的问题
func init() {
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configDir := filepath.Join(homeDir, ".easy-deploy")
			os.MkdirAll(configDir, 0755)
			configPath := filepath.Join(configDir, "config.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				os.WriteFile(configPath, configBytes, 0644)
			}
			os.Setenv("GVA_CONFIG", configPath)
		}
	}
}

// App struct — Wails 应用结构体
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// @title                       easy-deploy Swagger API接口文档
// @version                     v2.9.0
// @description                 使用gin+vue进行极速开发的全栈开发基础平台
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        x-token
// @BasePath                    /
// WailsSSEPort Wails 模式下 SSE sidecar HTTP 服务端口
// 前端 EventSource 在 Wails 环境下会连接此端口
const WailsSSEPort = 48009

func main() {
	// 初始化系统底层组件
	initializeSystem()

	// 获取嵌入的前端资源（提取 dist 子目录）
	distFS, _ := fs.Sub(frontendDist, "frontend/dist")

	// 初始化 Gin 路由器（作为 Wails 的 API Handler）
	ginRouter := initialize.Routers()

	// 启动 SSE sidecar HTTP 服务（真实 TCP 连接）
	// Wails 的 AssetServer.Handler 通过内部 IPC 转发请求，不支持 SSE 流式推送，
	// 因此需要一个真实的 HTTP 服务让 EventSource 能正常建立长连接。
	go startSSESidecar(ginRouter)

	app := NewApp()

	// 启动 Wails 桌面应用
	// - Assets: 嵌入的前端静态文件（HTML/CSS/JS）
	// - Handler: 包装 Gin 路由器，自动剥离 /api 前缀
	//   （与 Vite Proxy 和 Nginx rewrite 行为一致）
	err := wails.Run(&options.App{
		Title:     "easy-deploy",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 600,
		Debug: options.Debug{
			OpenInspectorOnStartup: true,
		},
		AssetServer: &assetserver.Options{
			Assets:  distFS,
			Handler: newAPIStripHandler(ginRouter),
		},
		OnStartup:  app.startup,
		OnShutdown: func(ctx context.Context) {},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		zap.L().Error("Wails 启动失败", zap.Error(err))
		fmt.Printf("Wails 启动失败: %v\n", err)

		// 回退到 HTTP 服务器模式
		fmt.Println("回退到浏览器模式，请手动访问 http://127.0.0.1:8008")
		core.RunServer()
	}
}

// initializeSystem 初始化系统所有组件
func initializeSystem() {
	// 设置嵌入的前端文件系统（供 router.go 中的 RegisterFrontendRoutes 使用）
	distFS, _ := fs.Sub(frontendDist, "frontend/dist")
	initialize.FrontendFS = distFS

	global.GVA_VP = core.Viper()
	initialize.OtherInit()
	global.GVA_LOG = core.Zap()
	zap.ReplaceGlobals(global.GVA_LOG)
	global.GVA_DB = initialize.Gorm()
	initialize.DBList()
	initialize.SetupHandlers()
	if global.GVA_DB != nil {
		initialize.RegisterTables()
	}
}

// startSSESidecar 在 127.0.0.1:WailsSSEPort 启动一个真实的 HTTP 服务
// 专供 SSE (Server-Sent Events) 流式连接使用
func startSSESidecar(ginRouter http.Handler) {
	addr := fmt.Sprintf("127.0.0.1:%d", WailsSSEPort)
	srv := &http.Server{
		Addr:           addr,
		Handler:        ginRouter,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Printf("[SSE Sidecar] 启动真实 HTTP 服务用于 SSE 流式推送: http://%s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("[SSE Sidecar] 启动失败: %v\n", err)
		zap.L().Error("SSE sidecar 启动失败", zap.Error(err))
	}
}

// newAPIStripHandler 创建一个 HTTP Handler 包装器，自动剥离请求路径中的 /api 前缀
// 这与 Vite Proxy (rewrite: path => path.replace(/^\/api/, '')) 和
// Nginx (rewrite ^/api/(.*)$ /$1 break;) 的行为完全一致
func newAPIStripHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origPath := r.URL.Path
		if strings.HasPrefix(r.URL.Path, "/api/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
			r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, "/api")
		} else if r.URL.Path == "/api" {
			r.URL.Path = "/"
			r.URL.RawPath = "/"
		}
		fmt.Printf("[Wails Handler] %s %s -> %s\n", r.Method, origPath, r.URL.Path)
		handler.ServeHTTP(w, r)
	})
}
