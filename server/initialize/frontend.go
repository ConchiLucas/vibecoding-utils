package initialize

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// FrontendFS 保存嵌入的前端文件系统，由 main.go 设置
var FrontendFS fs.FS

// RegisterFrontendRoutes 注册前端静态资源路由
// 使用 embed.FS 将前端打包进二进制，打包后无需依赖外部文件
func RegisterFrontendRoutes(router *gin.Engine) {
	if FrontendFS == nil {
		fmt.Println("[frontend] FrontendFS is nil, skipping embedded frontend routes")
		return
	}

	fmt.Println("[frontend] Registering embedded frontend routes...")

	// 读取 index.html 用于 SPA 回退
	indexHTML, err := fs.ReadFile(FrontendFS, "index.html")
	if err != nil {
		fmt.Printf("[frontend] WARNING: cannot read index.html from embedded FS: %v\n", err)
		return
	}

	fileServer := http.FileServer(http.FS(FrontendFS))

	// /assets/* 静态资源
	router.GET("/assets/*filepath", gin.WrapH(fileServer))

	// favicon.ico
	router.GET("/favicon.ico", gin.WrapH(fileServer))

	// logo.png
	router.GET("/logo.png", gin.WrapH(fileServer))

	// SPA 回退：根路径返回 index.html
	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	// NoRoute: 所有未匹配的路由也返回 index.html (SPA 路由)
	router.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	fmt.Println("[frontend] Embedded frontend routes registered successfully")
}
