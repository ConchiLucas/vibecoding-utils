# 启动

Codex 看到任何“启动项目 / 启动前后端 / 读取启动脚本启动”类请求时，只做这一件事：在仓库根目录直接执行：

```bash
./scripts/restart-dev.sh restart
```

不要先列文件；不要读 `README.md`、`server/README.md`、`web-react/package.json`、`Makefile` 或脚本源码；不要分析项目结构；不要手动查端口或 PID；不要额外健康检查。脚本会自己停止旧进程、清理占用端口、启动前后端、检查可用性并打印地址。

在 Codex Desktop 里，如果因为沙盒导致端口监听、清理进程或连接 PostgreSQL 失败，立刻申请沙盒外权限重跑同一条命令，不要改命令。

命令成功后只回复这两个地址：

- 前端：`http://localhost:29527/projects`
- 后端健康检查：`http://localhost:23638/health`

只有命令失败时，才看日志：

- 后端：`/private/tmp/vibecoding-utils-dev-logs/backend.log`
- 前端：`/private/tmp/vibecoding-utils-dev-logs/frontend.log`

停止：

```bash
./scripts/restart-dev.sh stop
```

容器化启动请使用共享网络包装器，避免新机器上外部网络尚未创建：

```bash
./scripts/docker-compose-up.sh -f docker-compose.yml up -d
```
