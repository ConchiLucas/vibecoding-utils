# Local Dev Startup

以后需要我启动这个项目时，可以直接说：

```text
按 docs/start-dev.md 启动前后端
```

我应该运行：

```bash
./scripts/start-dev.sh
```

这个脚本会做这些事：

- 启动后端 HTTP 服务，不走 Wails 桌面入口：`go run ./cmd/http`
- 从 `server/config.template.yaml` 生成临时配置：`/private/tmp/easy-deploy-dev-config.yaml`
- 默认从后端端口 `8009`、前端端口 `5176` 开始找可用端口
- 如果 `web-react/node_modules` 不存在，会先执行 `npm ci`
- 启动 React/Vite 前端，并把 `VITE_BASE_API` 指向实际后端端口

默认访问地址通常是：

- Frontend: `http://localhost:5176/`
- Backend health: `http://localhost:8009/health`

如果端口被占用，脚本会自动使用后面的可用端口，并在终端输出实际地址。

可以手动指定起始端口：

```bash
BACKEND_PORT=8010 FRONTEND_PORT=5180 ./scripts/start-dev.sh
```

Codex 桌面沙盒下运行时，启动后端和前端可能需要批准在沙盒外监听本机端口、访问本机 PostgreSQL、以及首次安装依赖时访问网络。
