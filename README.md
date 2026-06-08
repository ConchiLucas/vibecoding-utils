# VibeDeploy

[English](./README-en.md) | 简体中文

VibeDeploy 是一个面向多项目研发、数据库分析、代码生成和部署运维的本地工作台。它基于 Go + Gin 后端和 React 前端构建，保留了 easy-deploy 的基础能力，并围绕项目配置、脚本流程、接口调试、表关系分析和 AI 辅助代码生成做了定制。

这个仓库不是单纯的后台管理模板，而是一个把“项目资料、部署命令、数据库结构、接口文档、脚本流程、生成配置”集中到同一个页面体系里的工具。

## 我的项目功能是做什么的

顶部导航里的“项目池”就是当前系统里的“我的项目”入口。它不是业务项目源码本身，而是每个业务项目在 VibeDeploy 里的管理卡片。

你可以用它做这些事：

- 维护项目档案：项目名、语言类型、分组、说明、访问地址、本地路径等。
- 管理部署路线：给一个项目配置本地全量部署、本地增量部署、远程增量部署、依赖镜像构建等操作入口。
- 执行和停止部署：从项目卡片直接触发命令，并查看实时部署日志。
- 关联服务器配置：远程部署路线可以绑定服务器，统一维护 SSH 信息和执行命令。
- 进入项目脚本：项目可以挂载脚本文件，便于管理部署或辅助脚本内容。

简单说，“我的项目/项目池”是项目运维和部署动作的总入口；真正的业务代码仍然在你填写的本地项目路径或远程仓库里。

## 主要功能

### 项目池

- 项目分组、搜索、创建、编辑和删除。
- 为项目配置不同部署路线和展示颜色。
- 支持本地执行、远程执行、镜像构建、增量部署等命令模式。
- 部署日志面板可跟踪执行过程。

### 配置管理

- 维护当前活跃项目。
- 管理数据库连接，支持按环境区分数据源。
- 管理服务器连接信息。
- 配置 AI Provider、模型、Base URL、API Key 和默认模型。

### 数据库浏览和查询

- 浏览远程数据库、表和表注释。
- 预览表数据，查看字段注释、主键标识和建表 SQL。
- 对支持的数据库执行表数据编辑、删除和 AI 造数。
- 提供只读 SQL 查询窗口和成功查询历史。
- 当前代码包含 MySQL、PostgreSQL、Oracle、SQL Server、SQLite、ClickHouse 等数据源相关支持，具体写入能力以对应后端实现为准。

### 表关系

- 以数据源为入口打开数据库浏览、SQL 查询和表关系设置。
- 通过关键字在关联表中探查数据链路。
- 维护字段级表关系，并按表查看出入方向血缘。
- 提供 AI 导入表关系接口，便于外部分析结果批量写入。

### 代码生成

- 管理代码生成项目卡片，按业务类型分组。
- 配置前端/后端项目类型、实例、路径组、模型和模板。
- 管理数据库模板脚本，生成 SQL 片段。
- 生成代码后展示文件绝对路径和可交给 Codex 使用的提示词入口。

### 脚本库

- 按分类维护脚本流程和步骤。
- 支持本地执行、本地上传、目标下载、目标执行等步骤类型。
- 资源配置可以绑定数据库、服务器或其他变量。
- 多阶段 pipeline 会注入服务器环境变量，脚本中可直接引用阶段信息。

### 接口转发和接口调试

- 支持上传 Swagger JSON 构建项目/服务/接口树。
- 管理接口环境、接口用户、请求头、入参、出参和调用日志。
- 内置接口测试面板，适合调试后端接口和联调流程。

### 表样本和敏捷请求

- 表样本用于保存某个项目下常用业务表组合，并可按方案恢复。
- 敏捷请求提供轻量 API 请求工具，支持从浏览器 Copy as fetch 导入请求。
- 请求历史会保存在后端，方便复查和复用。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | React 18, Vite 5, TypeScript, Zustand, Tailwind CSS, Monaco Editor |
| 后端 | Go 1.24, Gin, Gorm, Viper, Zap |
| 数据库/连接 | MySQL, PostgreSQL, Oracle, SQL Server, SQLite, ClickHouse, Redis 可选 |
| 桌面端 | Wails v2 |
| 构建/部署 | Docker, docker compose, Makefile, Kubernetes 模板 |

## 目录结构

```text
.
├── web-react/                  # React 前端
│   ├── src/api/                # 前端 API 封装
│   ├── src/components/         # 通用组件，例如数据库浏览器、表数据预览
│   ├── src/views/              # 业务页面
│   └── package.json
├── server/                     # Go 后端和 Wails 桌面端入口
│   ├── api/v1/                 # HTTP API
│   ├── cmd/http/               # 纯 HTTP 开发入口
│   ├── config/                 # 配置结构
│   ├── initialize/             # 初始化、路由、数据库
│   ├── model/                  # 数据模型
│   ├── resource/               # 代码生成和部署模板
│   ├── router/                 # Gin 路由注册
│   ├── service/                # 业务逻辑
│   └── wails.json
├── docs/                       # 功能说明和打包说明
├── scripts/restart-dev.sh      # 本地前后端启动脚本
├── docker-compose.yml
└── Makefile
```

## 本地启动

### 环境要求

- Go 1.24+
- Node.js 18+
- npm 9+
- 一个可用的系统数据库配置，默认模板在 `server/config.template.yaml`

### 一键启动开发环境

在仓库根目录执行：

```bash
./scripts/restart-dev.sh restart
```

脚本会自动完成：

- 从 `server/config.template.yaml` 生成临时配置文件。
- 随机选择可用端口启动后端和前端。
- 后端使用 `go run ./cmd/http` 启动纯 HTTP 服务。
- 前端使用 Vite 启动，并把 `VITE_BASE_API` 指向实际后端端口。
- 如果 `web-react/node_modules` 不存在，会先执行 `npm ci`。

启动后终端会输出实际地址，例如：

```text
Backend:  http://localhost:23638
Frontend: http://localhost:29527
```

停止已记录的开发服务：

```bash
./scripts/restart-dev.sh stop
```

固定端口启动：

```bash
BACKEND_PORT=8008 FRONTEND_PORT=5175 ./scripts/restart-dev.sh restart
```

开发环境默认登录页会预填：

```text
username: admin
password: 123456
```

是否能直接登录取决于你的本地数据库是否已经初始化了对应账号。

### 手动启动

后端：

```bash
cd server
go run ./cmd/http -c /path/to/config.yaml
```

前端：

```bash
cd web-react
npm install
VITE_BASE_API=http://localhost:8008 npm run dev
```

## 构建

前端构建：

```bash
cd web-react
npm run build
```

构建产物会输出到：

```text
server/frontend/dist
```

后端 HTTP 入口构建：

```bash
cd server
go build ./cmd/http
```

桌面端打包请参考：

```text
docs/desktop-build.md
```

## 常用文档

- `docs/start-dev.md`：本地开发启动说明。
- `docs/desktop-build.md`：Wails 桌面端打包说明。
- `docs/ai-table-relations-import.md`：AI 导入表关系接口说明。
- `docs/pipeline-env-injection.md`：脚本 pipeline 环境变量注入说明。
- `docs/ai-deploy-phase1.md`：AI 部署流程阶段说明。

## 开发提示

- 前端统一使用 `web-react/src/utils/request.ts` 中的 Axios 实例。
- 当前活跃项目和数据源保存在 Zustand store 中。
- 私有接口需要 `x-token`，登录后前端会自动携带。
- 表数据编辑、删除、造数会直接写入远程数据库，操作前请确认目标环境。
- 数据库浏览、表关系和代码生成都依赖“配置管理”中的项目和数据源配置。

## License

本项目基于 easy-deploy 演进，仓库保留原项目的开源协议文件。使用、分发或商用前请阅读 [LICENSE](./LICENSE)。
