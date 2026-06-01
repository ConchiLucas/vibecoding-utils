# Easy Deploy 桌面端打包指南

## 技术方案

使用 [Wails v2](https://wails.io/) 将 Go 后端 + React 前端打包为原生桌面应用。

- 前端通过 `go:embed` 嵌入到 Go 二进制中
- Gin 路由器作为 Wails AssetServer 的 Handler 处理 API 请求
- 最终产物为单个可执行文件，无需额外运行时

## 环境准备

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 检查环境是否满足要求
wails doctor
```

### 依赖要求

| 工具 | 最低版本 |
|------|---------|
| Go   | 1.24+   |
| Node.js | 18+  |
| npm  | 9+      |
| Xcode CLI (macOS) | 最新版 |

## 本地打包

### 一键打包脚本

```bash
# 在项目根目录执行

# 1. 构建前端
cd web-react && npm install && npm run build && cd ..

# 2. 复制前端产物到嵌入目录
rm -rf server/frontend/dist && cp -r web-react/dist server/frontend/dist

# 3. Wails 打包（当前平台）
cd server && wails build -clean
```

### 打包产物位置

```
server/build/bin/
├── easy-deploy-desktop.app   (macOS)
├── easy-deploy-desktop.exe   (Windows)
└── easy-deploy-desktop       (Linux)
```

## 指定平台打包

| 目标平台 | 命令 |
|----------|------|
| macOS Apple Silicon (M1/M2/M3) | `wails build -clean -platform darwin/arm64` |
| macOS Intel | `wails build -clean -platform darwin/amd64` |
| macOS 通用二进制 | `wails build -clean -platform darwin/universal` |
| Windows x86_64 | `wails build -clean -platform windows/amd64` |
| Linux x86_64 | `wails build -clean -platform linux/amd64` |

> **注意**: macOS 环境下可以交叉编译 Windows，但无法直接编译 Linux（因为 CGO 依赖须在 Linux 环境下构建）。
> 要构建全平台包，推荐使用 GitHub Actions CI（见下文）。

## 开发模式

```bash
cd server
wails dev
```

Wails 开发模式支持：
- 前端热重载（自动监听 web-react 目录变化）
- 后端自动重编译
- 打开 DevTools 调试窗口

## GitHub Actions 自动构建（全平台）

项目已配置 `.github/workflows/desktop-release.yml`，支持自动构建 **4 个平台**：

| 平台 | 架构 | CI Runner |
|------|------|-----------|
| macOS | Apple Silicon (ARM64) | `macos-latest` |
| macOS | Intel (AMD64) | `macos-13` |
| Windows | x86_64 | `windows-latest` |
| Linux | x86_64 | `ubuntu-latest` |

### 触发方式

#### 方式一：打 Tag 自动发布

```bash
git add -A
git commit -m "feat: desktop v1.0.0"
git tag v1.0.0
git push && git push --tags
```

推送 tag 后，CI 自动：
1. 在 4 个平台并行构建
2. 上传构建产物
3. 创建 GitHub Release 并附带下载链接

#### 方式二：手动触发

1. 进入 GitHub 仓库 → **Actions** 标签页
2. 选择 **Desktop Release** 工作流
3. 点击 **Run workflow**
4. 输入版本号（如 `v1.0.0`）
5. 点击绿色按钮执行

### Release 产物

| 文件名 | 说明 |
|--------|------|
| `easy-deploy-desktop-macos-arm64.zip` | macOS Apple Silicon |
| `easy-deploy-desktop-macos-amd64.zip` | macOS Intel |
| `easy-deploy-desktop-windows-amd64.exe` | Windows 64 位 |
| `easy-deploy-desktop-linux-amd64` | Linux 64 位 |

## 配置说明

桌面端首次运行时会自动将 `config.template.yaml` 释放到用户主目录：

```
~/.easy-deploy/config.yaml
```

如需修改数据库等配置，编辑此文件即可。

## 常见问题

### macOS 提示"无法验证开发者"

```bash
# 方式1：系统设置 > 隐私与安全性 > 仍要打开
# 方式2：命令行移除隔离标记
xattr -cr /path/to/easy-deploy-desktop.app
```

### Windows SmartScreen 拦截

首次运行时 Windows 可能弹出 SmartScreen 警告，点击"更多信息" → "仍要运行"即可。

### Linux 运行

```bash
chmod +x easy-deploy-desktop-linux-amd64
./easy-deploy-desktop-linux-amd64
```

注意 Linux 需要安装 WebKit2GTK：
```bash
# Ubuntu/Debian
sudo apt install libwebkit2gtk-4.0-dev

# Fedora
sudo dnf install webkit2gtk4.0-devel

# Arch
sudo pacman -S webkit2gtk
```

## 项目结构（桌面端相关）

```
easy-deploy/
├── server/
│   ├── main.go              # Wails 入口，包含嵌入资源和 API Strip Handler
│   ├── wails.json            # Wails 项目配置
│   ├── config.template.yaml  # 嵌入的默认配置文件
│   ├── frontend/dist/        # 前端构建产物（嵌入到二进制）
│   └── build/bin/            # 打包产物输出目录
├── web-react/                # React 前端源码
└── .github/workflows/
    └── desktop-release.yml   # CI 跨平台构建工作流
```
