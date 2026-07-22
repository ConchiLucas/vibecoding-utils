# 移除 AI 助手及部署配置生成链设计

## 背景

VibeDeploy 顶部导航当前提供一个紫色 `AI` 按钮。按钮通过全局事件打开
`AIChatWidget`，聊天窗口支持模型选择、SSE 流式对话、聊天历史和工具调用。
后端 AI 助手的主要工具会扫描本地项目、判断技术类型、创建项目组、分配端口，
并向项目池写入项目、部署路线、Dockerfile、Compose 文件和启动脚本。

用户决定彻底删除这个 AI 助手的前后端功能，重点删除由聊天助手驱动的部署配置
生成代码。同时必须保留：

- 配置管理中的 AI 厂商和模型配置；
- `tb_ai_config` 中现有配置；
- 表样本页面的 AI 造数；
- 项目池原有的手工项目、路线、脚本和 Docker 部署功能。

用户还决定永久删除 AI 聊天历史表及其中的数据。当前开发数据库中
`tb_ai_chat_history` 实际尚不存在，但代码仍包含自动建表路径，因此仍需删除模型
注册并执行幂等清理，防止其他环境或未来启动重新创建该表。

## 目标

- 页面中不再出现顶部 AI 按钮或聊天窗口。
- 前端不再包含聊天、模型列表、历史记录和工具状态处理代码。
- 后端不再注册聊天、模型列表、Provider 列表或聊天历史接口。
- 删除 AI 助手专用的部署信息生成、项目扫描和项目池写入工具链。
- 将表样本 AI 造数依赖的通用模型调用从聊天服务中独立出来。
- AI 配置页面、持久化配置和 AI 造数继续工作。
- 永久删除 `tb_ai_chat_history`，并保证后续启动不会重新创建。
- 保留既有项目、路线、部署脚本和容器配置数据，不追溯删除过去由 AI 创建的项目。

## 非目标

- 不删除整个 AI 配置系统。
- 不删除 `tb_ai_config` 或配置文件中的 AI Provider 配置。
- 不删除表样本 AI 造数按钮和远程表造数接口。
- 不修改普通项目池的手工创建、模板生成、路线编辑或部署执行能力。
- 不识别并删除历史上由 AI 创建的项目、项目组、路线或脚本；这些记录与普通
  手工记录没有可靠的来源标记，删除会造成业务数据误伤。
- 不为已经删除的聊天接口保留兼容桩或专门的 404 Handler。

## 方案比较

### 方案一：拆除助手，提取共享 Completion 能力（采用）

删除 AI 助手的 UI、API、历史记录和部署工具，将通用模型请求与 AI 配置拆成独立
服务，供表样本造数继续使用。这一方案删除干净、依赖边界明确，并保护用户要求
保留的 AI 造数。

### 方案二：只隐藏按钮并注销路由

改动较小，但聊天、部署生成和数据库模型仍成为不可达死代码，未来可能被误注册，
也不符合“前后端都删除”的要求，因此不采用。

### 方案三：删除全部 AI 代码

边界最简单，但会删除 AI 配置和表样本造数，与确认范围冲突，因此不采用。

## 前端设计

### 导航和布局

修改 `web-react/src/views/layout/Header.tsx`：

- 删除 `Sparkles` 导入；
- 删除 AI 按钮；
- 删除 `toggle-ai-chat` 自定义事件触发。

修改 `web-react/src/views/layout/Layout.tsx`：

- 删除 `AIChatWidget` 导入；
- 删除全局 Widget 渲染节点。

删除整个 `web-react/src/components/AIChatWidget/`：

- `AIChatWidget.tsx`；
- `AIChatWidget.css`；
- `AIChatWidgetHistory.ts`；
- `AIChatWidgetIntent.ts`。

### API 边界

删除 `web-react/src/api/aiChat.ts`。这个文件当前混合了聊天和配置能力，因此先将
仍需要的部分迁移到 `web-react/src/api/aiConfig.ts`：

- `AIProviderConfigItem`；
- `AIConfigResponse`；
- `getAIConfig`；
- `saveAIConfig`；
- `saveAIActiveProvider`。

`AIConfigManager.tsx` 改为从 `aiConfig.ts` 导入。以下内容不迁移：

- Chat Message、History、Stream Callback 类型；
- Provider 列表请求；
- SSE 解析；
- 聊天历史查询和保存；
- 工具调用和工具结果事件处理。

配置管理中的 AI 页签保持不变，表样本组件继续通过 `sysConnection.ts` 调用造数
接口，不依赖任何聊天前端代码。

## 后端设计

### AI 配置服务

保留 `server/config/ai.go`、`server/model/system/tb_ai_config.go` 和
`server/service/system/sys_ai_config.go` 的配置语义。

将配置职责显式命名为 `AIConfigService`：

- `CurrentAIConfig`；
- `SaveAIConfig`；
- `SaveActiveAIConfig`；
- 数据库加载、Provider 归一化和可用性判断。

新建仅包含配置管理 Handler 的 `server/api/v1/system/sys_ai_config.go` 和
`server/router/system/sys_ai_config.go`。保留：

- `GET /ai/config`；
- `POST /ai/config`；
- `POST /ai/config/active`。

原 `AIChatApi`、`AIChatRouter` 名称不继续用于配置功能，避免删除助手后仍留下错误
语义。

### 通用模型 Completion 服务

从 `server/service/system/sys_ai_chat.go` 提取
`server/service/system/sys_ai_completion.go`，定义 `AICompletionService`。只保留
表样本造数所需的通用能力：

- 通用消息结构，重命名为 `AIMessage`；
- `CompleteOnce` 和 `CompleteJSONOnce`；
- OpenAI-compatible 与 Anthropic-compatible 的非流式请求；
- Provider endpoint 拼接、请求发送、响应读取、Max Tokens 归一化；
- 严格 JSON Response Format 和不支持时的普通 Completion 回退。

`TbConnectionService.GenerateRemoteTableData` 及其消息构建函数改为使用
`AICompletionService` 和 `AIMessage`。Completion Service 通过
`AIConfigService.CurrentAIConfig` 获取 Provider，不直接依赖聊天 API。

以下能力不进入 Completion Service：

- SSE 流式响应；
- Chat Tool Call 协议；
- AI 助手系统提示词；
- 部署生成成功消息格式化；
- 聊天模式、历史记录或项目工具调用。

### 删除聊天与历史链路

删除：

- `server/api/v1/system/sys_ai_chat.go` 中聊天、Models、Providers 部分；
- `server/api/v1/system/sys_ai_chat_history.go`；
- `server/router/system/sys_ai_chat.go` 中 Chat、Models、Providers、History 路由；
- `server/service/system/sys_ai_chat.go` 中所有助手专属代码；
- `server/service/system/sys_ai_chat_history.go`；
- `server/model/system/sys_ai_chat_history.go`。

从 API、Router、Service Group 和初始化代码中移除对应注册。删除完成后，不再注册：

- `POST /ai/chat`；
- `GET /ai/models`；
- `GET /ai/providers`；
- `POST /ai/chat/history`；
- `GET /ai/chat/history`；
- `GET /ai/chat/history/:chatId`。

旧客户端请求这些路径时由 Gin 的标准未匹配路由处理；产品不保留专门的兼容接口。
新前端不会再发出这些请求。

### 删除部署配置生成工具链

静态引用检查确认以下代码只由 AI 助手工具调用链使用，没有被普通项目池路由、
部署服务或表样本造数引用，因此删除：

- `server/service/system/sys_deploy_tools.go`；
- `server/service/system/project_type_detector.go`。

随文件删除的能力包括：

- `generate_deploy_info`；
- `detect_deploy_project_type`；
- `scan_project`；
- `create_deploy_project`；
- `auto_create_deploy_project`；
- `create_project_group`；
- `get_next_deploy_port` 的 AI Tool 包装；
- `list_projects` 的 AI Tool 包装；
- `import_table_relations` 的 AI Tool 包装；
- 对应 Tool Definition、参数解析、提示词和执行分发。

普通项目池仍使用 `ProjectService`、`ProjectRouteService`、`ProjectScriptService`、
模板资源和部署服务；这些代码不删除。即便工具链内部调用过这些服务，删除工具链
不会删除底层项目池能力。

## 数据库设计

### 保留的数据

- `tb_ai_config` 及现有 Provider 配置；
- 项目、项目组、路线和脚本表中的全部现有记录；
- 表样本造数涉及的连接和业务表数据。

### 永久清理的数据

确保 `RegisterTables` 继续不注册聊天历史表，并从
`ensureTables.MigrateTable`、`ensureTables.TableCreated` 中删除
`TbAIChatHistory`。增加一个幂等清理函数，并在应用的两个数据库初始化路径中调用：

```sql
DROP TABLE IF EXISTS tb_ai_chat_history;
```

这样当前不存在该表时不会报错；其他环境中存在时会永久删除表和数据；之后自动
迁移也不会重新创建它。

这是有意的不可逆数据删除。代码可以通过 Git 回退，但已经删除的聊天历史数据不
保证恢复。当前开发数据库没有该表，因此本机没有聊天记录可丢失。

## 数据流

删除后仅保留以下 AI 数据流：

```text
AI 配置页面
  -> /ai/config 或 /ai/config/active
  -> AIConfigApi
  -> AIConfigService
  -> tb_ai_config

表样本 AI 造数
  -> /connection/generateRemoteTableData
  -> TbConnectionService
  -> AICompletionService
  -> AIConfigService.CurrentAIConfig
  -> 配置的大模型
  -> 解析 JSON 并写入目标业务表
```

普通部署数据流保持独立：

```text
项目池手工操作
  -> Project / Route / Script API
  -> 项目池服务和部署模板
  -> Docker 构建与部署
```

系统中不再存在“聊天输入 -> Tool Call -> 自动写入部署配置”的路径。

## 错误处理

- AI 配置不存在或 Provider 不可用时，表样本造数返回明确的 Provider 配置错误。
- 模型网络、JSON 解析和重试错误继续由 Completion Service 与现有造数逻辑处理。
- 删除助手后不再产生 SSE 中断、工具调用失败或历史保存失败等聊天错误。
- 数据库清理使用 `DROP TABLE IF EXISTS`，不存在表不视为失败。
- 如果 `tb_ai_chat_history` 删除失败，应用初始化必须记录并返回错误，不能报告清理
  成功后继续悄悄保留旧表。

## 测试设计

### TDD 保护顺序

1. 先补充或调整 Completion/AI Config 契约测试，证明 AI 造数依赖的能力可脱离
   `AIChatService` 工作。
2. 观察测试因 `AICompletionService` 尚不存在而失败。
3. 提取最小 Completion 与 Config 服务，使造数测试恢复通过。
4. 增加路由契约，要求仅保留三条 AI Config 路由，不注册 Chat、Models、Providers
   和 History 路由。
5. 增加数据库迁移测试，先创建临时 `tb_ai_chat_history`，执行清理后确认表消失且
   `tb_ai_config` 保留。
6. 删除助手和部署工具代码，执行完整回归。

### 后端验证

- `tb_connection_generate_test.go` 全部通过，覆盖首次生成、修复和重试路径。
- AI Config 保存、读取和 Active Provider 切换测试通过。
- 路由清单只包含 `/ai/config` 与 `/ai/config/active`，不包含已删除接口。
- `go test ./service/system -count=1` 通过。
- 相关 API、Router、Initialize 包编译通过。
- 静态搜索不存在助手工具名、`DeployToolServiceApp`、`AIChatHistoryService` 或
  `AIChatWidget` 后端引用。

### 前端验证

- `npm run lint` 通过。
- `npm run build` 通过。
- Header 不包含 AI 按钮或 `toggle-ai-chat`。
- Layout 不导入或渲染 `AIChatWidget`。
- 不存在聊天、History、Stream 和 Tool Callback 前端代码。
- AI 配置页面仍能加载、保存并切换默认 Provider。
- 表样本页面仍显示 AI 造数入口。

### 集成验收

- 重启 VibeDeploy 前后端后，导航栏没有 AI 图标。
- 页面中没有聊天弹窗或聊天历史入口。
- 已删除的 Chat、Models、Providers 和 History 路径不在 Gin 路由清单中。
- `GET /ai/config`、保存配置和切换 Active Provider 正常。
- AI 造数单元/集成契约保持通过；不在生产业务表上执行破坏性验收造数。
- PostgreSQL 查询 `to_regclass('public.tb_ai_chat_history')` 返回空。
- `tb_ai_config` 存在且修改前后的配置内容一致。
- 项目池可读取项目、路线和脚本，现有 Docker 部署入口仍正常。

## 实施与回滚边界

实施分为三个可验证提交：

1. 提取并保护 AI Config/Completion 与造数依赖；
2. 删除后端助手、历史和部署生成工具，加入数据库清理；
3. 删除前端入口、Widget 和聊天 API，完成集成验收。

如果第一步无法保持 AI 造数测试通过，停止删除并恢复该提交。如果后续代码验证
失败，回退尚未通过的代码提交。数据库表删除按用户要求是永久操作：代码回退可
重新创建空表，但不能恢复已经删除的聊天历史数据。当前开发库中该表不存在，
因此本机实施不会删除现有聊天记录。
