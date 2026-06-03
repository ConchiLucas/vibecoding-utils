package system

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/config"
	"github.com/flipped-aurora/easy-deploy/server/global"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"go.uber.org/zap"
)

type AIChatService struct{}

// ─── OpenAI-Compatible Types ────────────────────────────────────────────

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDef struct {
	Type     string          `json:"type"`
	Function ToolFunctionDef `json:"function"`
}

type ToolFunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	Tools     []ToolDef     `json:"tools,omitempty"`
	Stream    bool          `json:"stream"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type ChatChoice struct {
	Delta        ChatMessage `json:"delta"`
	Message      ChatMessage `json:"message"`
	FinishReason *string     `json:"finish_reason"`
}

type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
}

func formatGenerateDeployInfoToolResult(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "生成部署信息失败：工具没有返回结果，请检查提示词或稍后重试。", true
	}
	if strings.HasPrefix(raw, "工具执行失败:") {
		return "生成部署信息失败：" + strings.TrimSpace(strings.TrimPrefix(raw, "工具执行失败:")), true
	}

	var result GenerateDeployInfoResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return "生成部署信息失败：工具返回结果无法解析，请检查提示词或稍后重试。", true
	}
	if result.ProjectId == 0 {
		return "生成部署信息失败：工具返回的项目ID为空，已停止生成成功提示，请检查提示词或数据库写入结果。", true
	}

	return fmt.Sprintf(`已成功为您生成部署信息！

项目信息：
|项目|详情|
|------|------|
|项目ID|%d|
|项目名称|%s|
|项目类型|%s|
|本地路径|%s|
|访问地址|%s|

项目组信息：
|项目组|详情|
|--------|------|
|组ID|%d|
|组名称|%s|

生成结果：
- 创建了%d条部署路由
- 创建了%d个部署脚本

您现在可以前往 VibeDeploy 项目池，点击对应的部署按钮执行部署。`,
		result.ProjectId,
		result.ProjectName,
		result.ComputerLanguage,
		result.LocalProjectPath,
		result.AccessUrl,
		result.GroupId,
		result.GroupName,
		result.RoutesCreated,
		result.ScriptsCreated,
	), true
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type AnthropicRequest struct {
	Model     string             `json:"model"`
	System    string             `json:"system,omitempty"`
	Messages  []AnthropicMessage `json:"messages"`
	Tools     []AnthropicToolDef `json:"tools,omitempty"`
	Stream    bool               `json:"stream"`
	MaxTokens int                `json:"max_tokens"`
}

type anthropicToolUseBuffer struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// ─── System Prompt ──────────────────────────────────────────────────────

const systemPrompt = `你是 VibeDeploy 部署管理平台的 AI 助手。你可以帮助用户：
1. 识别本地目录的可部署项目类型（Vue、React、Python、Java、Go、前后端 docker-compose、未知）
2. 根据用户提供的本地项目目录自动创建模板化部署项目（含路由和脚本）
3. 查询现有项目列表
4. 导入 AI 整理出的表字段血缘/关联关系
5. 回答部署相关的问题

当用户请求涉及系统操作时，请使用提供的工具来执行。
当用户整理出表之间的关联关系、血缘关系、外键映射并要求写入系统时，优先调用 import_table_relations。调用时只需要按标准格式填 projectConfigId 和 relations；source 表示当前主表字段，target 表示被关联的目标表字段。不要把关系推理过程传入工具，只传结构化参数。
当用户选择“生成部署信息”或说“生成部署配置/生成部署信息”时，优先调用 generate_deploy_info。调用时必须把用户输入中的本地目录提取到 local_path，只保留真实目录路径，不要把“生成部署信息/放在某组里”等自然语言拼进路径；如果用户指定项目组，提取到 group_name，组名由你根据自然语言完整识别，像“AI数据库组”这类以“组”结尾的名称要完整保留“组”字；同时必须识别 group_action：用户要新建/创建项目组时为 create，用户要放入/复用/使用已有项目组时为 reuse，语义不明确时为 auto；如果用户说再生成一份、compare/对比，allow_duplicate_path=true。input 保留用户原始文本用于后端兜底。
如果用户一次提供多个本地目录并要求放入同一个项目组，请为每个目录分别调用一次 generate_deploy_info；第一条如果是新建/创建某组，group_action=create，后续放入同名组的调用 group_action=reuse，并继续完整传入同一个 group_name。
当用户说“在某个目录生成部署信息”、“给某个目录创建部署项目”、“自动生成部署脚本”、“加全量部署”这类需求时，优先调用 auto_create_deploy_project；用户只需要提供目录路径，不需要手动填写项目名、框架语言或端口。
当用户要求“再生成一个组名”、“再生成一份”、“重新生成一组”、“compare/对比组”时，先根据用户意图生成一个新的项目组名并调用 create_project_group；随后调用 auto_create_deploy_project，只把 create_project_group 返回的 group_id 传入，并设置 allow_duplicate_path=true。这个场景不要查找相似项目组，也不要因为 local_path 已经存在而停止。不要在 auto_create_deploy_project 中再次传 group_name。
当用户想判断一个目录是什么项目类型、框架语言或部署项目类型时，优先调用 detect_deploy_project_type。
最终项目类型只能从 Vue、React、Python、Java、Go、前后端 docker-compose、未知 中选择。
当前 Vue、React、Python、Java、Go 已接入部署模板；Python 默认生成构建项目镜像、构建依赖增量镜像、本地增量部署三种部署路线，三种构建都使用同一个项目镜像名，不再生成额外的 xxx-base 或 xxx-deps 镜像名。只有用户明确要求远程部署、远程增量部署、部署到服务器或上传服务器时，才为 Python 额外生成远程增量部署路线。
auto_create_deploy_project 会自动检测 Vue/React/Java/Go/Python 项目。Vue/React 会自动分配前端访问端口和后端代理端口，并生成本地全量部署路线；Java/Go 会自动分配后端访问端口，并生成本地全量部署和本地增量部署路线；Python 默认生成构建项目镜像、构建依赖增量镜像、本地增量部署路线。Python 本地增量部署只更新代码运行镜像，复用已存在的项目镜像；依赖文件变化时提醒用户先执行构建依赖增量镜像。远程增量路线不写死服务器，只有用户明确要求远程时才生成；执行前需要用户在路线里选择服务器节点；远端项目路径可在路线配置中填写，也可从服务器节点配置中按项目名读取。创建完成后请告诉用户去项目池点击对应部署按钮即可执行 docker compose 部署。
如果用户在生成部署信息时明确指定端口号，例如“端口号用17889”“使用17889端口”，必须传给 generate_deploy_info：后端/Java/Go/Python 放到 app_port，Vue/React 前端访问端口放到 frontend_deploy_port；显式端口优先级最高，不再调用建议端口递增规则覆盖。
当手动调用 create_deploy_project 且用户没有指定端口时，先调用 get_next_deploy_port 获取建议端口：Vue/React 使用 frontend，Java/Go/Python 使用 backend；建议端口只依据项目池数据库记录计算，不依据当前机器端口占用，且同类项目按 10 个端口为一个隔离槽递增。
对于普通问题，直接用文字回答。
请用中文回答。`

// ─── Tool Definitions ───────────────────────────────────────────────────

func getToolDefinitions() []ToolDef {
	return []ToolDef{
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "generate_deploy_info",
				Description: "极简生成部署信息入口。用户最少输入本地项目目录，也可以补充组名、项目名、再生成一份等要求；工具会自动识别项目类型、生成或创建项目组、分配端口并创建部署项目、路线和脚本。只有路径无效、指定组名已存在等异常才需要用户补充。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{
							"type":        "string",
							"description": "用户原始输入，原样传入，用于后端兜底解析和意图判断",
						},
						"local_path": map[string]interface{}{
							"type":        "string",
							"description": "从用户输入中提取出的本地项目绝对路径，只包含目录路径，不要包含“生成部署信息/放在某组里”等自然语言",
						},
						"group_name": map[string]interface{}{
							"type":        "string",
							"description": "你从用户自然语言中识别出的纯项目组名，例如“英语抢词_compare”“AI数据库组”“定时任务”；如果名称本身以“组”结尾，需要完整保留；没有指定时留空",
						},
						"group_action": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"create", "reuse", "auto"},
							"description": "项目组动作由你根据用户自然语言识别：新建/创建项目组为 create；放入/复用/使用已有项目组为 reuse；用户只给了组名但没有明确动作时为 auto",
						},
						"project_name": map[string]interface{}{
							"type":        "string",
							"description": "用户指定的项目名或服务名；没有指定时留空",
						},
						"app_port": map[string]interface{}{
							"type":        "integer",
							"description": "用户明确指定的后端/Java/Go/Python 部署端口，例如“端口号用17889”；没有指定时留空或 0",
						},
						"frontend_deploy_port": map[string]interface{}{
							"type":        "integer",
							"description": "用户明确指定的 Vue/React 前端访问端口；没有指定时留空或 0",
						},
						"use_existing_group": map[string]interface{}{
							"type":        "boolean",
							"description": "兼容旧字段。优先填写 group_action；只有无法填写 group_action 时才使用此字段",
						},
						"allow_duplicate_path": map[string]interface{}{
							"type":        "boolean",
							"description": "用户要求再生成一份、compare/对比、重新生成，或同一路径需要新建一份部署信息时为 true",
						},
						"include_remote": map[string]interface{}{
							"type":        "boolean",
							"description": "只有用户明确要求远程部署、部署到服务器、ssh/sftp、上传服务器时为 true",
						},
					},
					"required": []string{"input", "local_path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "detect_deploy_project_type",
				Description: "识别本地目录最适合归类成哪一种可部署项目类型，只返回 Vue、React、Python、Java、Go、前后端 docker-compose、未知 这些类型及证据",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"local_path": map[string]interface{}{
							"type":        "string",
							"description": "本地项目目录的绝对路径",
						},
					},
					"required": []string{"local_path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "scan_project",
				Description: "扫描本地项目目录，自动检测语言类型（go/java/python/vue/react）、项目名称和入口文件",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"local_path": map[string]interface{}{
							"type":        "string",
							"description": "本地项目的绝对路径",
						},
					},
					"required": []string{"local_path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "create_deploy_project",
				Description: "为 Vue、React、Python、Java、Go 项目一键创建模板化部署项目，自动从模板生成部署脚本和路由配置。用户未指定端口时，应先调用 get_next_deploy_port 获取建议端口",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"project_name": map[string]interface{}{
							"type":        "string",
							"description": "项目名称",
						},
						"computer_language": map[string]interface{}{
							"type":        "string",
							"description": "编程语言: go, java, python, vue, react",
						},
						"local_project_path": map[string]interface{}{
							"type":        "string",
							"description": "本地项目路径",
						},
						"app_port": map[string]interface{}{
							"type":        "integer",
							"description": "后端部署端口号，前端反代会通过 host.docker.internal 访问；未指定时先用 get_next_deploy_port(project_type=backend) 获取",
						},
						"frontend_deploy_port": map[string]interface{}{
							"type":        "integer",
							"description": "前端本地全量部署宿主机端口，不能和 React/Vite 本地开发端口或后端端口一致；未指定时先用 get_next_deploy_port(project_type=frontend) 获取",
						},
					},
					"required": []string{"project_name", "computer_language", "local_project_path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "auto_create_deploy_project",
				Description: "根据本地项目目录自动检测项目类型并创建部署项目、路由和脚本。Vue/React 项目会生成本地全量部署路线；Java/Go 项目会生成本地全量部署和本地增量部署路线；Python 默认生成构建项目镜像、构建依赖增量镜像、本地增量部署三条路线，但都使用同一个项目镜像名，不生成额外 xxx-base 或 xxx-deps 镜像名；只有用户明确要求远程部署时才设置 include_remote=true 额外生成远程增量路线。用户要求再生成一组或对比组时，传入 group_id 并设置 allow_duplicate_path=true",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"local_path": map[string]interface{}{
							"type":        "string",
							"description": "本地项目目录的绝对路径",
						},
						"project_name": map[string]interface{}{
							"type":        "string",
							"description": "可选项目名称；未传时自动从 package.json 或目录名读取",
						},
						"group_id": map[string]interface{}{
							"type":        "integer",
							"description": "可选项目组 ID；未传时为 0",
						},
						"container_name": map[string]interface{}{
							"type":        "string",
							"description": "可选容器名称；未传时根据项目名自动生成",
						},
						"app_port": map[string]interface{}{
							"type":        "integer",
							"description": "可选后端代理端口；未传时自动取 backend 建议端口",
						},
						"frontend_deploy_port": map[string]interface{}{
							"type":        "integer",
							"description": "可选前端宿主机访问端口；未传时自动取 frontend 建议端口",
						},
						"allow_duplicate_path": map[string]interface{}{
							"type":        "boolean",
							"description": "是否允许同一个 local_path 再创建一份部署信息；用户要求再生成一个组名、再生成一份或 compare 对比组时必须为 true",
						},
						"include_remote": map[string]interface{}{
							"type":        "boolean",
							"description": "仅当用户明确要求远程部署、远程增量部署、部署到服务器或上传服务器时为 true；默认 false",
						},
					},
					"required": []string{"local_path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "create_project_group",
				Description: "创建一个新的项目组。项目组名由大模型根据用户提示生成或直接使用用户指定名称；本工具总是新建，不查找相似组名",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"group_name": map[string]interface{}{
							"type":        "string",
							"description": "要创建的新项目组名称",
						},
					},
					"required": []string{"group_name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "get_next_deploy_port",
				Description: "根据 tb_project.access_url 中已有访问地址计算下一次部署建议端口，只看项目池数据库，不检测当前机器端口占用。frontend 从 6001 起按 6001/6011/6021 递增，backend 从 10001 起按 10001/10011/10021 递增",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"project_type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"frontend", "backend"},
							"description": "部署项目类型：frontend 表示 Vue/React 前端端口，backend 表示 Java/Go/Python 后端端口",
						},
					},
					"required": []string{"project_type"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "import_table_relations",
				Description: "批量导入表字段血缘/关联关系到 tb_table_relate。source 表示当前主表字段，target 表示被关联的目标表字段。适合 AI 从其他项目整理出表关系后一次性写入系统。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"projectConfigId": map[string]interface{}{
							"type":        "integer",
							"description": "当前项目配置 ID，用于区分要写入哪个项目下的血缘关系",
						},
						"userName": map[string]interface{}{
							"type":        "string",
							"description": "可选操作人标识；未传时后端使用当前用户或 ai",
						},
						"relations": map[string]interface{}{
							"type":        "array",
							"description": "要批量导入的表字段关联关系列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"source": map[string]interface{}{
										"type":        "object",
										"description": "源字段，即当前主表上的字段",
										"properties": map[string]interface{}{
											"databaseName": map[string]interface{}{
												"type":        "string",
												"description": "源表所在数据库名",
											},
											"tableName": map[string]interface{}{
												"type":        "string",
												"description": "源表名",
											},
											"columnName": map[string]interface{}{
												"type":        "string",
												"description": "源表字段名",
											},
											"columnType": map[string]interface{}{
												"type":        "string",
												"description": "可选源字段类型，例如 bigint、varchar、int",
											},
										},
										"required": []string{"databaseName", "tableName", "columnName"},
									},
									"target": map[string]interface{}{
										"type":        "object",
										"description": "目标字段，即 source 关联到的表字段",
										"properties": map[string]interface{}{
											"databaseName": map[string]interface{}{
												"type":        "string",
												"description": "目标表所在数据库名",
											},
											"tableName": map[string]interface{}{
												"type":        "string",
												"description": "目标表名",
											},
											"columnName": map[string]interface{}{
												"type":        "string",
												"description": "目标表字段名",
											},
											"columnType": map[string]interface{}{
												"type":        "string",
												"description": "可选目标字段类型，例如 bigint、varchar、int",
											},
										},
										"required": []string{"databaseName", "tableName", "columnName"},
									},
								},
								"required": []string{"source", "target"},
							},
						},
					},
					"required": []string{"projectConfigId", "relations"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "list_projects",
				Description: "获取所有已创建的部署项目列表，包含项目名称、语言、路由等信息",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

// ─── Tool Execution ─────────────────────────────────────────────────────

func (s *AIChatService) executeToolCall(tc ToolCall) (string, error) {
	global.GVA_LOG.Info("执行工具调用", zap.String("tool", tc.Function.Name), zap.String("args", tc.Function.Arguments))

	switch tc.Function.Name {
	case "generate_deploy_info":
		var args struct {
			Input              string `json:"input"`
			LocalPath          string `json:"local_path"`
			GroupName          string `json:"group_name"`
			GroupAction        string `json:"group_action"`
			ProjectName        string `json:"project_name"`
			AppPort            int    `json:"app_port"`
			FrontendDeployPort int    `json:"frontend_deploy_port"`
			UseExistingGroup   bool   `json:"use_existing_group"`
			AllowDuplicatePath bool   `json:"allow_duplicate_path"`
			IncludeRemote      bool   `json:"include_remote"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := DeployToolServiceApp.GenerateDeployInfo(GenerateDeployInfoRequest{
			Input:              args.Input,
			LocalPath:          args.LocalPath,
			GroupName:          args.GroupName,
			GroupAction:        args.GroupAction,
			ProjectName:        args.ProjectName,
			AppPort:            args.AppPort,
			FrontendDeployPort: args.FrontendDeployPort,
			UseExistingGroup:   args.UseExistingGroup,
			AllowDuplicatePath: args.AllowDuplicatePath,
			IncludeRemote:      args.IncludeRemote,
		})
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "detect_deploy_project_type":
		var args struct {
			LocalPath string `json:"local_path"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := DetectDeployProjectType(args.LocalPath)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "scan_project":
		var args struct {
			LocalPath string `json:"local_path"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := DeployToolServiceApp.ScanProject(args.LocalPath)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "create_deploy_project":
		var args struct {
			ProjectName        string `json:"project_name"`
			ComputerLanguage   string `json:"computer_language"`
			LocalProjectPath   string `json:"local_project_path"`
			AppPort            int    `json:"app_port"`
			FrontendDeployPort int    `json:"frontend_deploy_port"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		if args.AppPort == 0 {
			portResult, err := ProjectServiceApp.GetNextDeployPort("backend")
			if err != nil {
				return "", err
			}
			args.AppPort = portResult.NextPort
		}
		normalizedLanguage := normalizeDeployLanguage(args.ComputerLanguage)
		if args.FrontendDeployPort == 0 && (normalizedLanguage == "react" || normalizedLanguage == "vue") {
			portResult, err := ProjectServiceApp.GetNextDeployPort("frontend")
			if err != nil {
				return "", err
			}
			args.FrontendDeployPort = portResult.NextPort
		}
		result, err := DeployToolServiceApp.QuickInitProject(QuickInitRequest{
			ProjectName:        args.ProjectName,
			ComputerLanguage:   args.ComputerLanguage,
			LocalProjectPath:   args.LocalProjectPath,
			AppPort:            args.AppPort,
			FrontendDeployPort: args.FrontendDeployPort,
		})
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "auto_create_deploy_project":
		var args struct {
			LocalPath          string `json:"local_path"`
			ProjectName        string `json:"project_name"`
			GroupId            uint   `json:"group_id"`
			ContainerName      string `json:"container_name"`
			AppPort            int    `json:"app_port"`
			FrontendDeployPort int    `json:"frontend_deploy_port"`
			AllowDuplicatePath bool   `json:"allow_duplicate_path"`
			IncludeRemote      bool   `json:"include_remote"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := DeployToolServiceApp.AutoCreateDeployProject(AutoCreateDeployProjectRequest{
			LocalPath:          args.LocalPath,
			ProjectName:        args.ProjectName,
			GroupId:            args.GroupId,
			ContainerName:      args.ContainerName,
			AppPort:            args.AppPort,
			FrontendDeployPort: args.FrontendDeployPort,
			AllowDuplicatePath: args.AllowDuplicatePath,
			IncludeRemote:      args.IncludeRemote,
		})
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "create_project_group":
		var args struct {
			GroupName string `json:"group_name"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := DeployToolServiceApp.CreateProjectGroup(args.GroupName)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "get_next_deploy_port":
		var args struct {
			ProjectType string `json:"project_type"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := ProjectServiceApp.GetNextDeployPort(args.ProjectType)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "import_table_relations":
		var args systemReq.ImportTableRelationsRequest
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
		result, err := (&TbTableRelateService{}).ImportTableRelations(args, "ai")
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(result)
		return string(b), nil

	case "list_projects":
		projects, err := DeployToolServiceApp.ListAllProjects()
		if err != nil {
			return "", err
		}
		// 简化输出
		type projectSummary struct {
			ID       uint   `json:"id"`
			Name     string `json:"name"`
			Language string `json:"language"`
			Path     string `json:"path"`
		}
		var summaries []projectSummary
		for _, p := range projects {
			summaries = append(summaries, projectSummary{
				ID:       p.ID,
				Name:     p.ProjectName,
				Language: p.ComputerLanguage,
				Path:     p.LocalProjectPath,
			})
		}
		b, _ := json.Marshal(summaries)
		return string(b), nil

	default:
		return "", fmt.Errorf("未知工具: %s", tc.Function.Name)
	}
}

// ─── HTTP Helper ────────────────────────────────────────────────────────

func aiProviderEndpoint(provider config.ResolvedAIProvider) string {
	switch provider.Type {
	case config.AIProviderTypeAnthropicCompatible:
		return appendAIEndpoint(provider.BaseURL, "/v1/messages")
	default:
		return appendAIEndpoint(provider.BaseURL, "/v1/chat/completions")
	}
}

func appendAIEndpoint(baseURL string, suffix string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(suffix, "/v1/") {
		return base + strings.TrimPrefix(suffix, "/v1")
	}
	return base + suffix
}

// CompleteOnce calls the configured AI provider once and returns the final text.
func (s *AIChatService) CompleteOnce(messages []ChatMessage, providerID string) (string, config.ResolvedAIProvider, error) {
	provider, err := global.GVA_CONFIG.AI.ResolveProvider(providerID)
	if err != nil {
		return "", config.ResolvedAIProvider{}, err
	}

	if provider.Type == config.AIProviderTypeAnthropicCompatible {
		content, err := s.completeAnthropicOnce(provider, messages)
		return content, provider, err
	}

	content, err := s.completeOpenAIOnce(provider, messages)
	return content, provider, err
}

func (s *AIChatService) completeOpenAIOnce(provider config.ResolvedAIProvider, messages []ChatMessage) (string, error) {
	req := ChatRequest{
		Model:     provider.Model,
		Messages:  messages,
		Stream:    false,
		MaxTokens: normalizedAIProviderMaxTokens(provider),
	}
	reqBody, _ := json.Marshal(req)
	resp, err := s.doAIRequest(provider, reqBody)
	if err != nil {
		return "", fmt.Errorf("请求 AI 模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI 模型返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 模型没有返回内容")
	}
	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("AI 模型返回空内容")
	}
	return content, nil
}

func (s *AIChatService) completeAnthropicOnce(provider config.ResolvedAIProvider, messages []ChatMessage) (string, error) {
	systemText, nonSystemMessages := splitSystemMessages(messages)
	req := AnthropicRequest{
		Model:     provider.Model,
		System:    systemText,
		Messages:  openAIToAnthropicMessages(nonSystemMessages),
		Stream:    false,
		MaxTokens: normalizedAIProviderMaxTokens(provider),
	}
	reqBody, _ := json.Marshal(req)
	resp, err := s.doAIRequest(provider, reqBody)
	if err != nil {
		return "", fmt.Errorf("请求 AI 模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI 模型返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	var builder strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" || block.Type == "" {
			builder.WriteString(block.Text)
		}
	}
	content := strings.TrimSpace(builder.String())
	if content == "" {
		return "", fmt.Errorf("AI 模型返回空内容")
	}
	return content, nil
}

func normalizedAIProviderMaxTokens(provider config.ResolvedAIProvider) int {
	if provider.MaxTokens > 0 {
		return provider.MaxTokens
	}
	return 4096
}

func splitSystemMessages(messages []ChatMessage) (string, []ChatMessage) {
	var systemParts []string
	nonSystemMessages := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" {
			if strings.TrimSpace(message.Content) != "" {
				systemParts = append(systemParts, message.Content)
			}
			continue
		}
		nonSystemMessages = append(nonSystemMessages, message)
	}
	return strings.Join(systemParts, "\n\n"), nonSystemMessages
}

// doAIRequest 发送请求到 AI 模型
func (s *AIChatService) doAIRequest(provider config.ResolvedAIProvider, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", aiProviderEndpoint(provider), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if provider.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.ApiKey)
		if provider.Type == config.AIProviderTypeAnthropicCompatible {
			req.Header.Set("x-api-key", provider.ApiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	}
	return http.DefaultClient.Do(req)
}

// ─── SSE Streaming Chat with Function Calling Loop ──────────────────────

// ChatStreamHandler 处理 AI 对话的 SSE 流式接口
// 始终使用流式请求，实时输出内容。如果 LLM 要调用工具，则从流中收集 tool_calls，
// 执行后把结果发回，再流式输出最终回复。
func (s *AIChatService) ChatStreamHandler(messages []ChatMessage, providerID string, onEvent func(eventType, data string)) error {
	provider, err := global.GVA_CONFIG.AI.ResolveProvider(providerID)
	if err != nil {
		return err
	}
	if provider.Type == config.AIProviderTypeAnthropicCompatible {
		return s.chatAnthropicStreamHandler(provider, messages, onEvent)
	}
	return s.chatOpenAIStreamHandler(provider, messages, onEvent)
}

func (s *AIChatService) chatOpenAIStreamHandler(provider config.ResolvedAIProvider, messages []ChatMessage, onEvent func(eventType, data string)) error {
	// 注入 system prompt
	fullMessages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	fullMessages = append(fullMessages, messages...)

	tools := getToolDefinitions()

	// Function Calling 循环（最多 5 轮工具调用）
	for round := 0; round < 5; round++ {
		streamReq := ChatRequest{
			Model:     provider.Model,
			Messages:  fullMessages,
			Tools:     tools,
			Stream:    true,
			MaxTokens: provider.MaxTokens,
		}

		reqBody, _ := json.Marshal(streamReq)
		resp, err := s.doAIRequest(provider, reqBody)
		if err != nil {
			return fmt.Errorf("请求 AI 模型失败: %w", err)
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("AI 模型返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
		}

		// 从流中读取，同时收集 content 和 tool_calls
		var contentBuf strings.Builder
		collectedToolCalls := make(map[int]*ToolCall) // index -> ToolCall
		finishReason := ""

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk ChatResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta

			// 实时转发文本内容
			if delta.Content != "" {
				contentBuf.WriteString(delta.Content)
				onEvent("content", delta.Content)
			}

			// 收集 tool_calls（流式模式下分多个 chunk 发送）
			for _, tc := range delta.ToolCalls {
				idx := 0 // 默认 index 0
				if existing, ok := collectedToolCalls[idx]; ok {
					// 追加到已有的 tool call
					existing.Function.Name += tc.Function.Name
					existing.Function.Arguments += tc.Function.Arguments
				} else {
					// 新建 tool call
					collectedToolCalls[idx] = &ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}

			if chunk.Choices[0].FinishReason != nil {
				finishReason = *chunk.Choices[0].FinishReason
			}
		}
		resp.Body.Close()

		// 如果是工具调用
		if finishReason == "tool_calls" && len(collectedToolCalls) > 0 {
			var toolCalls []ToolCall
			for _, tc := range collectedToolCalls {
				toolCalls = append(toolCalls, *tc)
				toolInfo, _ := json.Marshal(map[string]string{"name": tc.Function.Name, "arguments": tc.Function.Arguments})
				onEvent("tool_call", string(toolInfo))
			}

			// 追加 assistant 的 tool_calls 消息
			fullMessages = append(fullMessages, ChatMessage{
				Role:      "assistant",
				ToolCalls: toolCalls,
			})

			// 执行每个工具调用
			var deterministicSummaries []string
			for _, tc := range toolCalls {
				result, err := s.executeToolCall(tc)
				if err != nil {
					result = fmt.Sprintf("工具执行失败: %s", err.Error())
				}
				fullMessages = append(fullMessages, ChatMessage{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
				toolResult, _ := json.Marshal(map[string]string{"name": tc.Function.Name, "result": result})
				onEvent("tool_result", string(toolResult))
				if tc.Function.Name == "generate_deploy_info" {
					if summary, ok := formatGenerateDeployInfoToolResult(result); ok {
						deterministicSummaries = append(deterministicSummaries, summary)
					}
				}
			}

			if len(deterministicSummaries) > 0 {
				onEvent("content", strings.Join(deterministicSummaries, "\n\n---\n\n"))
				onEvent("done", "")
				return nil
			}

			// 下一轮不带 tools，让 LLM 直接用文字总结
			tools = nil
			continue
		}

		// 普通回复已经在上面实时输出了，直接结束
		onEvent("done", "")
		return nil
	}

	onEvent("content", "\n\n⚠️ 工具调用轮次超限，请简化你的请求。")
	onEvent("done", "")
	return nil
}

func (s *AIChatService) chatAnthropicStreamHandler(provider config.ResolvedAIProvider, messages []ChatMessage, onEvent func(eventType, data string)) error {
	fullMessages := openAIToAnthropicMessages(messages)
	tools := getAnthropicToolDefinitions()
	maxTokens := provider.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	for round := 0; round < 5; round++ {
		streamReq := AnthropicRequest{
			Model:     provider.Model,
			System:    systemPrompt,
			Messages:  fullMessages,
			Tools:     tools,
			Stream:    true,
			MaxTokens: maxTokens,
		}

		reqBody, _ := json.Marshal(streamReq)
		resp, err := s.doAIRequest(provider, reqBody)
		if err != nil {
			return fmt.Errorf("请求 AI 模型失败: %w", err)
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("AI 模型返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
		}

		toolUses, err := s.readAnthropicStream(resp.Body, onEvent)
		resp.Body.Close()
		if err != nil {
			return err
		}

		if len(toolUses) == 0 {
			onEvent("done", "")
			return nil
		}

		assistantBlocks := make([]map[string]interface{}, 0, len(toolUses))
		toolResults := make([]map[string]interface{}, 0, len(toolUses))
		var deterministicSummaries []string
		for _, toolUse := range toolUses {
			args := toolUse.Arguments.String()
			if strings.TrimSpace(args) == "" {
				args = "{}"
			}
			var input interface{}
			if err := json.Unmarshal([]byte(args), &input); err != nil {
				input = map[string]interface{}{}
			}
			assistantBlocks = append(assistantBlocks, map[string]interface{}{
				"type":  "tool_use",
				"id":    toolUse.ID,
				"name":  toolUse.Name,
				"input": input,
			})

			info, _ := json.Marshal(map[string]string{"name": toolUse.Name, "arguments": args})
			onEvent("tool_call", string(info))

			result, err := s.executeToolCall(ToolCall{
				ID:   toolUse.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      toolUse.Name,
					Arguments: args,
				},
			})
			if err != nil {
				result = fmt.Sprintf("工具执行失败: %v", err)
			}

			resultInfo, _ := json.Marshal(map[string]string{"name": toolUse.Name, "result": result})
			onEvent("tool_result", string(resultInfo))
			if toolUse.Name == "generate_deploy_info" {
				if summary, ok := formatGenerateDeployInfoToolResult(result); ok {
					deterministicSummaries = append(deterministicSummaries, summary)
				}
			}
			toolResults = append(toolResults, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": toolUse.ID,
				"content":     result,
			})
		}

		if len(deterministicSummaries) > 0 {
			onEvent("content", strings.Join(deterministicSummaries, "\n\n---\n\n"))
			onEvent("done", "")
			return nil
		}

		fullMessages = append(fullMessages,
			AnthropicMessage{Role: "assistant", Content: assistantBlocks},
			AnthropicMessage{Role: "user", Content: toolResults},
		)
		tools = nil
	}

	onEvent("content", "\n\n⚠️ 工具调用轮次超限，请简化你的请求。")
	onEvent("done", "")
	return nil
}

func (s *AIChatService) readAnthropicStream(body io.Reader, onEvent func(eventType, data string)) ([]*anthropicToolUseBuffer, error) {
	toolUses := make(map[int]*anthropicToolUseBuffer)
	eventType := ""

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if value, ok := readSSEField(line, "event"); ok {
			eventType = value
			continue
		}
		data, ok := readSSEField(line, "data")
		if !ok {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var payload struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type  string          `json:"type"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			continue
		}

		currentEventType := eventType
		if currentEventType == "" {
			currentEventType = payload.Type
		}

		switch currentEventType {
		case "content_block_start":
			if payload.ContentBlock.Type == "tool_use" {
				toolUses[payload.Index] = &anthropicToolUseBuffer{
					ID:   payload.ContentBlock.ID,
					Name: payload.ContentBlock.Name,
				}
				if len(payload.ContentBlock.Input) > 0 && string(payload.ContentBlock.Input) != "null" {
					trimmedInput := strings.TrimSpace(string(payload.ContentBlock.Input))
					if trimmedInput != "" && trimmedInput != "{}" {
						toolUses[payload.Index].Arguments.WriteString(trimmedInput)
					}
				}
			}
		case "content_block_delta":
			if payload.Delta.Type == "text_delta" && payload.Delta.Text != "" {
				onEvent("content", payload.Delta.Text)
			}
			if payload.Delta.Type == "input_json_delta" {
				if toolUse, ok := toolUses[payload.Index]; ok {
					toolUse.Arguments.WriteString(payload.Delta.PartialJSON)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 AI 流失败: %w", err)
	}

	indexes := make([]int, 0, len(toolUses))
	for index := range toolUses {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	result := make([]*anthropicToolUseBuffer, 0, len(indexes))
	for _, index := range indexes {
		result = append(result, toolUses[index])
	}
	return result, nil
}

func readSSEField(line string, field string) (string, bool) {
	prefix := field + ":"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return strings.TrimLeft(strings.TrimPrefix(line, prefix), " \t"), true
}

func openAIToAnthropicMessages(messages []ChatMessage) []AnthropicMessage {
	result := make([]AnthropicMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" {
			continue
		}
		role := message.Role
		if role != "assistant" {
			role = "user"
		}
		result = append(result, AnthropicMessage{
			Role:    role,
			Content: message.Content,
		})
	}
	return result
}

func getAnthropicToolDefinitions() []AnthropicToolDef {
	openAITools := getToolDefinitions()
	tools := make([]AnthropicToolDef, 0, len(openAITools))
	for _, tool := range openAITools {
		tools = append(tools, AnthropicToolDef{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}
	return tools
}
