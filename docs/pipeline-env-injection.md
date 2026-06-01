# Pipeline 服务器环境变量注入

> 实现日期: 2026-04-25

## 功能说明

在多服务器编排（pipeline）模式下，自动将所有阶段的服务器连接信息作为环境变量注入到每个阶段的脚本中，避免在脚本中硬编码 IP、用户名等信息。

### 使用示例

```bash
# 阶段1（本机）- 直接引用阶段2的服务器信息，无需硬编码
pg_dump -U postgres mydb > /tmp/dump.sql
scp /tmp/dump.sql $STAGE_2_USER@$STAGE_2_IP:~/dump.sql

# 阶段2（远程服务器）
psql -U postgres mydb < ~/dump.sql
```

### 注入的环境变量

| 变量名 | 说明 | 示例值 |
|--------|------|--------|
| `STAGE_N_IP` | 第 N 阶段服务器的公网 IP | `192.168.5.44` |
| `STAGE_N_USER` | 第 N 阶段服务器的登录用户 | `conchi` |
| `STAGE_N_PORT` | 第 N 阶段服务器的 SSH 端口 | `22` |
| `STAGE_N_NAME` | 第 N 阶段服务器的名称 | `macbook-m3max` |
| `STAGE_N_INTERNAL_IP` | 第 N 阶段服务器的内网 IP | `192.168.5.44` |

> **注意**: 当某阶段为"本机执行"（`serverId=0`）时，对应的 `STAGE_N_*` 变量不注入，避免混淆。

---

## 修改的文件

### 后端: `server/service/system/sys_script_manager.go`

#### 1. 新增 `buildPipelineServerEnvVars` 函数

- 遍历所有 pipeline steps，批量加载引用的服务器信息（单次 DB 查询）
- 为每个 `serverId > 0` 的阶段构建 `STAGE_N_*` 环境变量
- `serverId=0` 的阶段自动跳过

#### 2. 修改 `executePipeline` 函数

- 在执行循环前调用 `buildPipelineServerEnvVars` 一次性构建
- 本机执行：`command.Env = append(command.Env, pipelineEnvVars...)`
- SSH 远程执行：传入 `executeScriptViaSSH(..., pipelineEnvVars)`

#### 3. 修改 `ExecuteScriptWithLog` 函数 (pipeline 分支)

- 同样在循环前构建 `pipelineEnvVars`
- 传递给 `executeScriptViaSSH` 和 `executeLocalWithLog`

#### 4. 修改 `executeScriptViaSSH` 签名

```go
// 之前
func executeScriptViaSSH(ctx, server, scriptContent, inputParams, cwd) 
// 之后
func executeScriptViaSSH(ctx, server, scriptContent, inputParams, cwd, extraEnvVars []string)
```

- 在 scriptBuilder 头部 export 额外的环境变量
- 非 pipeline 调用处传入 `nil`

#### 5. 修改 `executeLocalWithLog` 签名

```go
// 之前
func executeLocalWithLog(ctx, scriptContent, workingDir, inputParams, logCh, allOutput)
// 之后  
func executeLocalWithLog(ctx, scriptContent, workingDir, inputParams, logCh, allOutput, extraEnvVars []string)
```

- 追加到 `command.Env`
- 非 pipeline 调用处传入 `nil`

---

### 前端: `web-react/src/views/script-manager/ScriptManager.tsx`

#### 多服务器编排变量提示条

- 在每个阶段的 `<textarea>` 下方添加动态提示条
- 显示效果：`📌 可用变量: $STAGE_2_IP (阶段2 (macbook-m3max)) · $STAGE_2_USER ...`
- 只显示**其他阶段**（非当前阶段）的远程服务器变量
- `serverId=0` 的阶段标注 "(本机)" 且不显示变量
- 切换服务器时，提示内容**实时动态更新**

---

## 核心代码

### `buildPipelineServerEnvVars` (Go)

```go
func buildPipelineServerEnvVars(steps []pipelineStep) []string {
    var envVars []string
    // 收集 server IDs，批量查询
    serverIDs := make([]uint, 0)
    for _, step := range steps {
        if step.ServerID > 0 {
            serverIDs = append(serverIDs, step.ServerID)
        }
    }
    if len(serverIDs) == 0 {
        return envVars
    }

    var servers []modelSystem.TbServer
    global.GVA_DB.Where("id IN ?", serverIDs).Find(&servers)
    serverMap := make(map[uint]modelSystem.TbServer)
    for _, srv := range servers {
        serverMap[srv.ID] = srv
    }

    for i, step := range steps {
        stageNum := strconv.Itoa(i + 1)
        prefix := "STAGE_" + stageNum + "_"
        if step.ServerID == 0 {
            continue // 本机阶段跳过
        }
        srv, ok := serverMap[step.ServerID]
        if !ok {
            continue
        }
        envVars = append(envVars,
            prefix+"IP="+srv.ServerIp,
            prefix+"USER="+srv.ServerLoginName,
            prefix+"PORT="+strconv.Itoa(srv.ServerLoginPort),
            prefix+"NAME="+srv.ServerName,
            prefix+"INTERNAL_IP="+srv.ServerInternalIp,
        )
    }
    return envVars
}
```
