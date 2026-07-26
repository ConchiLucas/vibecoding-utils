package system

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"go.uber.org/zap"
)

type DeployService struct{}

var DeployServiceApp = new(DeployService)

const dockerLogTailLines = "200"

// ProcessDeploy 执行项目部署（兼容原有接口）
func (s *DeployService) ProcessDeploy(projectId uint, targetEnv string) error {
	return s.ProcessDeployWithLog(projectId, targetEnv, nil)
}

// ProcessDeployWithLog 执行项目部署（支持日志流式推送）
func (s *DeployService) ProcessDeployWithLog(projectId uint, targetEnv string, logCh chan string) error {
	// 辅助函数：向 logCh 发送阶段性日志
	sendLog := func(msg string) {
		if logCh != nil {
			select {
			case logCh <- msg:
			default:
			}
		}
	}

	sendLog("📋 开始获取项目信息...")
	// 1. 获取项目信息
	project, err := ProjectServiceApp.GetProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取项目信息失败: %w", err)
	}
	sendLog(fmt.Sprintf("✅ 项目: %s (ID=%d)", project.ProjectName, projectId))

	// 1.5 提取匹配的路由配置
	var currentRoute system.TbProjectRoute
	for _, r := range project.Routes {
		if r.RouteKey == targetEnv || fmt.Sprintf("%d", r.ID) == targetEnv {
			currentRoute = r
			break
		}
	}
	if currentRoute.RouteKey == "" {
		return fmt.Errorf("未找到对应环境标识的路由配置: %s", targetEnv)
	}
	sendLog(fmt.Sprintf("✅ 路由: %s", currentRoute.RouteKey))

	// 2. 获取服务器信息 (server 关联已移至路由层)
	svrId := uint(currentRoute.ServerId)
	var server system.TbServer
	if svrId != 0 {
		server, err = ServerServiceApp.GetServerById(svrId)
		if err != nil {
			return fmt.Errorf("获取服务器信息失败: %w", err)
		}
		sendLog(fmt.Sprintf("✅ 目标服务器: %s", server.ServerIp))
	}

	var serverIp, serverLoginName, serverLoginPassword string
	var serverPort int
	if server.ID != 0 {
		serverIp = server.ServerIp
		serverPort = server.ServerLoginPort
		serverLoginName = server.ServerLoginName
		serverLoginPassword = server.ServerLoginPassword
	}

	serverProjectPath := strings.TrimSpace(currentRoute.ServerProjectPath)
	localProjectPath := currentRoute.LocalProjectPath
	if localProjectPath == "" {
		localProjectPath = project.LocalProjectPath
	}
	localScriptPath := resolveLocalScriptPath(currentRoute, localProjectPath)
	if serverProjectPath == "" && server.ID != 0 {
		serverProjectPath = resolveServerProjectPathFromNodeConfig(server, project, currentRoute)
	}
	if server.ID != 0 && serverProjectPath == "" {
		return fmt.Errorf("远程部署路径未配置: 请在路线配置中填写远程服务器绝对路径，或在服务器节点配置中添加项目路径")
	}
	fileName := currentRoute.FileName
	// 仅远程部署（有目标服务器IP）时才需要 fileName（用于打包上传）；本机部署无需此字段
	if fileName == "" && serverIp != "" {
		return fmt.Errorf("打包参数不完整: 请在路由配置里补充填写 [压缩文件名] (如 dist.zip / app.zip)，远程部署必填")
	}
	localExecuteCommand := currentRoute.LocalExecuteCommand
	serverExecuteCommand := currentRoute.ServerExecuteCommand
	computerLanguage := project.ComputerLanguage

	cmdUtil := &utils.CmdUtil{LogCh: logCh}

	if strings.HasPrefix(currentRoute.RouteKey, "remote_") && currentRoute.ServerId == 0 {
		return fmt.Errorf("远程部署路线未绑定服务器，请先在路线配置中选择目标服务器")
	}
	if currentRoute.RouteKey == "local" || currentRoute.ServerId == 0 {
		sendLog("🏠 执行本地部署模式...")
		return runLocalDeployWithSharedNetwork(logCh, SharedDockerNetworkServiceApp.Ensure, func() error {
			if err := s.prepareAggregateChildDeployScripts(project, currentRoute, logCh); err != nil {
				return err
			}
			return s.processLocalDeploy(cmdUtil, projectId, currentRoute.ID, localProjectPath, localScriptPath, localExecuteCommand, currentRoute.LocalStartCommand)
		})
	}

	sftpUtil := &utils.SftpUtil{}

	deployLang := computerLanguage
	if computerLanguage == "python" && currentRoute.ServerId == 0 {
		if currentRoute.BuildType == "build_image" {
			deployLang = "python_image"
		} else if currentRoute.BuildType == "build_incremental_image" {
			deployLang = "python_image_add"
		}
	} else if computerLanguage == "python" && currentRoute.BuildType == "build_incremental_image" {
		deployLang = "python_remote_incremental"
	}

	switch deployLang {
	case "vue":
		return s.deployVue(cmdUtil, sftpUtil, projectId, currentRoute.ID, localProjectPath, fileName, localExecuteCommand, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
	case "python":
		return s.deployPython(cmdUtil, sftpUtil, projectId, currentRoute.ID, localProjectPath, fileName, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
	case "java":
		return s.deployJava(cmdUtil, sftpUtil, projectId, currentRoute.ID, localProjectPath, fileName, localExecuteCommand, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
	case "python_image":
		return s.deployPythonImage(cmdUtil, sftpUtil, projectId, currentRoute.ID, localProjectPath, fileName, localExecuteCommand, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath)
	case "python_image_add":
		return s.deployPythonImageAdd(cmdUtil, sftpUtil, projectId, currentRoute.ID, localProjectPath, fileName, localExecuteCommand, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath)
	case "python_remote_incremental":
		return s.deployPythonRemoteIncremental(cmdUtil, sftpUtil, projectId, currentRoute.ID, localProjectPath, fileName, localExecuteCommand, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
	default:
		return fmt.Errorf("不支持该语言类型: %s", deployLang)
	}
}

func runLocalDeployWithSharedNetwork(
	logCh chan string,
	ensure func(context.Context) (SharedDockerNetworkResult, error),
	deploy func() error,
) error {
	sendDeployNetworkLog(logCh, "🌐 检查共享 Docker 网络...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := ensure(ctx)
	if err != nil {
		guardErr := fmt.Errorf("共享 Docker 网络 %s 不可用: %w", SharedDockerNetworkName, err)
		sendDeployNetworkLog(logCh, "❌ "+guardErr.Error())
		return guardErr
	}
	if result.Created {
		sendDeployNetworkLog(logCh, "✅ 共享 Docker 网络已创建")
	} else {
		sendDeployNetworkLog(logCh, "✅ 共享 Docker 网络已存在")
	}
	return deploy()
}

func sendDeployNetworkLog(logCh chan string, message string) {
	if logCh == nil {
		return
	}
	select {
	case logCh <- message:
	default:
	}
}

// StreamDockerLogs tails docker compose/container logs for a deployment route.
func (s *DeployService) StreamDockerLogs(ctx context.Context, projectId uint, targetEnv string, serviceName string, logCh chan string) error {
	sendLog := func(msg string) {
		if logCh != nil {
			select {
			case logCh <- msg:
			default:
			}
		}
	}

	project, err := ProjectServiceApp.GetProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取项目信息失败: %w", err)
	}

	currentRoute, err := findRouteInLoadedProject(project.Routes, targetEnv)
	if err != nil {
		return err
	}
	if currentRoute.ServerId != 0 {
		return fmt.Errorf("当前仅支持本机 Docker 实时日志，远程路线请先登录服务器查看")
	}

	localProjectPath := strings.TrimSpace(currentRoute.LocalProjectPath)
	if localProjectPath == "" {
		localProjectPath = strings.TrimSpace(project.LocalProjectPath)
	}
	if localProjectPath == "" {
		return fmt.Errorf("项目本地路径为空，无法定位 Docker 日志")
	}

	sendLog(fmt.Sprintf("📋 项目: %s", project.ProjectName))
	sendLog(fmt.Sprintf("📋 路线: %s", currentRoute.RouteName))

	command, args, workDir := buildDockerLogCommand(project, currentRoute, localProjectPath, serviceName)
	sendLog(fmt.Sprintf("🐳 日志命令: %s %s", command, strings.Join(args, " ")))
	return streamCommandLines(ctx, workDir, command, args, logCh)
}

func findRouteInLoadedProject(routes []system.TbProjectRoute, targetEnv string) (system.TbProjectRoute, error) {
	for _, route := range routes {
		if route.RouteKey == targetEnv || fmt.Sprintf("%d", route.ID) == targetEnv {
			return route, nil
		}
	}
	return system.TbProjectRoute{}, fmt.Errorf("未找到对应环境标识的路由配置: %s", targetEnv)
}

func buildDockerLogCommand(project system.TbProject, route system.TbProjectRoute, localProjectPath string, serviceName string) (string, []string, string) {
	// 普通项目卡片与实际容器一一对应，直接跟踪容器日志，避免依赖 Compose
	// 项目名、配置文件路径和启动脚本工作目录。聚合 Compose 卡片没有同名容器，
	// 仍保留 Compose 服务级日志能力。
	if strings.TrimSpace(serviceName) == "" && !isAggregateComposeProject(project) {
		containerName := strings.TrimSpace(project.ProjectName)
		return "docker", []string{"logs", "--tail", dockerLogTailLines, "-f", containerName}, localProjectPath
	}

	executeCommand := strings.ToLower(route.LocalExecuteCommand + " " + route.LocalStopCommand + " " + route.BuildType)
	composeRoot := resolveLocalScriptPath(route, localProjectPath)
	composeFilePath := filepath.Join(composeRoot, "docker-compose.yml")
	composeYamlPath := filepath.Join(composeRoot, "docker-compose.yaml")
	_, composeFileErr := os.Stat(composeFilePath)
	_, composeYamlErr := os.Stat(composeYamlPath)
	composePath := ""
	if composeFileErr == nil {
		composePath = composeFilePath
	} else if composeYamlErr == nil {
		composePath = composeYamlPath
	}
	shouldUseCompose := strings.Contains(executeCommand, "docker compose") ||
		strings.Contains(executeCommand, "docker-compose") ||
		route.DockerComposeDeploy ||
		route.BuildType == "docker_compose_deploy" ||
		composeFileErr == nil ||
		composeYamlErr == nil

	if shouldUseCompose {
		args := []string{"compose"}
		if composePath != "" {
			args = append(args, "-f", composePath, "--project-directory", localProjectPath)
		}
		args = append(args, "logs", "--tail", dockerLogTailLines, "-f")
		if strings.TrimSpace(serviceName) != "" {
			args = append(args, strings.TrimSpace(serviceName))
		}
		return "docker", args, localProjectPath
	}

	containerName := strings.TrimSpace(project.ProjectName)
	args := []string{"logs", "--tail", dockerLogTailLines, "-f", containerName}
	return "docker", args, localProjectPath
}

func isAggregateComposeProject(project system.TbProject) bool {
	language := strings.ToLower(strings.TrimSpace(project.ComputerLanguage))
	projectName := strings.ToLower(strings.TrimSpace(project.ProjectName))
	return strings.Contains(language, "docker-compose") ||
		strings.Contains(language, "docker compose") ||
		strings.Contains(projectName, "compose")
}

func streamCommandLines(ctx context.Context, workDir string, command string, args []string, logCh chan string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建 stdout 管道失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建 stderr 管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 Docker 日志命令失败: %w", err)
	}

	scanDone := make(chan struct{}, 2)
	scan := func(scanner *bufio.Scanner) {
		defer func() { scanDone <- struct{}{} }()
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case logCh <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}
	go scan(bufio.NewScanner(stdout))
	go scan(bufio.NewScanner(stderr))

	err = cmd.Wait()
	<-scanDone
	<-scanDone
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Docker 日志命令退出: %w", err)
	}
	return nil
}

// deployVue Vue/React项目部署
func (s *DeployService) deployVue(cmdUtil *utils.CmdUtil, sftpUtil *utils.SftpUtil, projectId uint, routeId uint, localProjectPath, fileName, localExecuteCommand, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand string) error {
	// 本机服务器场景：下载脚本到本地项目路径，然后执行本机执行命令
	if serverIp == "" {
		global.GVA_LOG.Info(fmt.Sprintf("Vue/React本机部署: 项目ID=%d, 本地路径=%s", projectId, localProjectPath))
		// 1. 把脚本列表里的文件下载到本地项目绝对路径
		if err := s.downloadScriptsToLocalFromDB(projectId, routeId, localProjectPath); err != nil {
			return err
		}
		// 2. 执行本机执行命令（如 npm run build）
		if localExecuteCommand != "" {
			if err := cmdUtil.RunNpmBuild(localProjectPath, localExecuteCommand); err != nil {
				return fmt.Errorf("Vue/React本机执行命令失败: %w", err)
			}
		}
		return nil
	}

	// 远程服务器场景：打包 → 压缩 → 上传 → 上传脚本 → 执行远程命令
	// 执行打包命令
	if err := cmdUtil.RunNpmBuild(localProjectPath, localExecuteCommand); err != nil {
		return fmt.Errorf("npm打包失败: %w", err)
	}
	// 压缩dist目录
	distPath := filepath.Join(localProjectPath, "dist")
	zipPath := filepath.Join(localProjectPath, fileName)
	if err := cmdUtil.ZipDirectory(distPath, zipPath); err != nil {
		return fmt.Errorf("压缩dist目录失败: %w", err)
	}
	// 无论后续执行是否成功，在函数退出（即上传和相关操作完成后）删除本地生成的压缩包
	defer os.Remove(zipPath)

	// 上传到服务器
	targetDir := buildTargetPath(localProjectPath, fileName)
	if err := sftpUtil.UploadLocalPathWithPort(targetDir, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, fileName); err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}
	// 上传脚本到远程服务器
	if err := s.uploadScriptsFromDB(sftpUtil, projectId, routeId, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath); err != nil {
		return err
	}
	// 执行远程命令
	return sftpUtil.ExecuteShell(serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
}

// deployPython Python项目部署
func (s *DeployService) deployPython(cmdUtil *utils.CmdUtil, sftpUtil *utils.SftpUtil, projectId uint, routeId uint, localProjectPath, fileName, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand string) error {
	// 打包整个项目目录
	zipPath := filepath.Join(localProjectPath, fileName)
	if err := cmdUtil.ZipDirectory(localProjectPath, zipPath); err != nil {
		return fmt.Errorf("压缩目录失败: %w", err)
	}
	// 无论后续执行是否成功，在函数退出（即上传和相关操作完成后）删除本地生成的压缩包
	defer os.Remove(zipPath)

	// 上传到服务器
	targetDir := buildTargetPath(localProjectPath, fileName)
	if err := sftpUtil.UploadLocalPathWithPort(targetDir, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, fileName); err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}
	// 上传脚本当到远程服务器
	if err := s.uploadScriptsFromDB(sftpUtil, projectId, routeId, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath); err != nil {
		return err
	}
	// 执行远程命令
	return sftpUtil.ExecuteShell(serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
}

// deployJava Java项目部署
func (s *DeployService) deployJava(cmdUtil *utils.CmdUtil, sftpUtil *utils.SftpUtil, projectId uint, routeId uint, localProjectPath, fileName, localExecuteCommand, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand string) error {
	// 本机服务器场景：下载脚本到本地项目路径，然后执行本机命令
	if serverIp == "" {
		global.GVA_LOG.Info(fmt.Sprintf("Java本机部署: 项目ID=%d, 本地路径=%s", projectId, localProjectPath))
		// 1. 把脚本列表里的文件下载到本地项目绝对路径
		if err := s.downloadScriptsToLocalFromDB(projectId, routeId, localProjectPath); err != nil {
			return err
		}
		// 2. 执行本机执行命令
		if localExecuteCommand == "" {
			return fmt.Errorf("Java本机部署失败: 未配置本机执行命令 (localExecuteCommand)")
		}
		if err := cmdUtil.PackageProject(localProjectPath, localExecuteCommand); err != nil {
			return fmt.Errorf("Java本机执行命令失败: %w", err)
		}
		return nil
	}

	// 远程服务器场景：打包 → 上传 → 上传脚本 → 执行远程命令
	// 执行打包命令
	if err := cmdUtil.PackageProject(localProjectPath, localExecuteCommand); err != nil {
		return fmt.Errorf("Java打包失败: %w", err)
	}
	// 上传到服务器
	targetDir := buildTargetPath(localProjectPath, fileName)
	if err := sftpUtil.UploadLocalPathWithPort(targetDir, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, fileName); err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}
	// 上传脚本到远程服务器
	if err := s.uploadScriptsFromDB(sftpUtil, projectId, routeId, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath); err != nil {
		return err
	}
	// 执行远程命令
	return sftpUtil.ExecuteShell(serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
}

// deployPythonImage Python镜像部署
func (s *DeployService) deployPythonImage(cmdUtil *utils.CmdUtil, sftpUtil *utils.SftpUtil, projectId uint, routeId uint, localProjectPath, fileName, localExecuteCommand, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath string) error {
	// 释放DB记录的脚本到本地构建机器工作区
	if err := s.downloadScriptsToLocalFromDB(projectId, routeId, localProjectPath); err != nil {
		return err
	}
	// 删除原有镜像/执行前置命令
	if err := cmdUtil.PackageProject(localProjectPath, localExecuteCommand); err != nil {
		global.GVA_LOG.Warn("执行前置命令失败", zap.Error(err))
	}
	// 构建镜像
	if err := cmdUtil.BuildImage(localProjectPath, fileName); err != nil {
		return fmt.Errorf("构建Docker镜像失败: %w", err)
	}
	// 保存镜像
	if err := cmdUtil.SaveImage(localProjectPath, fileName); err != nil {
		return fmt.Errorf("保存Docker镜像失败: %w", err)
	}
	// 无论后续执行是否成功，在函数退出（即上传和相关操作完成后）删除本地生成的镜像tar包文件
	tarPath := filepath.Join(localProjectPath, fileName)
	defer os.Remove(tarPath)
	// 上传tar包到服务器
	targetDir := buildTargetPath(localProjectPath, fileName)
	if err := sftpUtil.UploadLocalPathWithPort(targetDir, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, fileName); err != nil {
		return fmt.Errorf("上传镜像文件失败: %w", err)
	}
	// 上传start.sh
	startSh := filepath.Join(localProjectPath, "start.sh")
	return sftpUtil.UploadLocalPathWithPort(startSh, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, "start.sh")
}

// deployPythonImageAdd Python镜像增量部署
func (s *DeployService) deployPythonImageAdd(cmdUtil *utils.CmdUtil, sftpUtil *utils.SftpUtil, projectId uint, routeId uint, localProjectPath, fileName, localExecuteCommand, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath string) error {
	// 释放DB记录的脚本到本地构建机器工作区
	if err := s.downloadScriptsToLocalFromDB(projectId, routeId, localProjectPath); err != nil {
		return err
	}
	// 执行前置令/环境同步
	if err := cmdUtil.PackageProject(localProjectPath, localExecuteCommand); err != nil {
		return fmt.Errorf("执行打包命令失败: %w", err)
	}
	// 保存镜像
	if err := cmdUtil.SaveImage(localProjectPath, fileName); err != nil {
		return fmt.Errorf("保存Docker镜像失败: %w", err)
	}
	// 无论后续执行是否成功，在函数退出（即上传和相关操作完成后）删除本地生成的镜像tar包文件
	tarPath := filepath.Join(localProjectPath, fileName)
	defer os.Remove(tarPath)
	// 上传tar包到服务器
	targetDir := buildTargetPath(localProjectPath, fileName)
	if err := sftpUtil.UploadLocalPathWithPort(targetDir, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, fileName); err != nil {
		return fmt.Errorf("上传镜像文件失败: %w", err)
	}
	// 上传start.sh
	startSh := filepath.Join(localProjectPath, "start.sh")
	return sftpUtil.UploadLocalPathWithPort(startSh, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, "start.sh")
}

// deployPythonRemoteIncremental 构建本地增量运行镜像，上传 tar 到远程并重建容器。
func (s *DeployService) deployPythonRemoteIncremental(cmdUtil *utils.CmdUtil, sftpUtil *utils.SftpUtil, projectId uint, routeId uint, localProjectPath, fileName, localExecuteCommand, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand string) error {
	if fileName == "" {
		return fmt.Errorf("远程增量部署失败: 未配置镜像 tar 文件名")
	}
	if localExecuteCommand == "" {
		return fmt.Errorf("远程增量部署失败: 未配置本机执行命令")
	}
	if serverExecuteCommand == "" {
		return fmt.Errorf("远程增量部署失败: 未配置远端执行命令")
	}

	if err := s.downloadScriptsToLocalFromDB(projectId, routeId, localProjectPath); err != nil {
		return err
	}
	if err := cmdUtil.PackageProject(localProjectPath, localExecuteCommand); err != nil {
		return fmt.Errorf("执行远程增量打包命令失败: %w", err)
	}

	tarPath := filepath.Join(localProjectPath, fileName)
	defer os.Remove(tarPath)
	if err := sftpUtil.UploadLocalPathWithPort(tarPath, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, fileName); err != nil {
		return fmt.Errorf("上传增量镜像文件失败: %w", err)
	}
	if err := s.uploadScriptsFromDB(sftpUtil, projectId, routeId, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath); err != nil {
		return err
	}
	return sftpUtil.ExecuteShell(serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, serverExecuteCommand)
}

// uploadScriptsFromDB 从数据库提取脚本并直接上传到服务器
func (s *DeployService) uploadScriptsFromDB(sftpUtil *utils.SftpUtil, projectId uint, routeId uint, serverIp string, serverPort int, serverLoginName, serverLoginPassword, serverProjectPath string) error {
	scriptList, err := ProjectScriptServiceApp.GetProjectScriptList(int(projectId), int(routeId))
	if err != nil {
		return fmt.Errorf("获取项目脚本列表失败: %w", err)
	}
	for _, script := range scriptList {
		if script.Content == "" {
			continue
		}
		// ScriptType 1 = 仅Local, 不要打到远程服务器
		if script.ScriptType == 1 {
			continue
		}
		// Directly read from the memory string
		reader := strings.NewReader(script.Content)
		if err := sftpUtil.UploadMemory(reader, serverIp, serverPort, serverLoginName, serverLoginPassword, serverProjectPath, script.FileName); err != nil {
			return fmt.Errorf("上传脚本文件失败(%s): %w", script.FileName, err)
		}
	}
	return nil
}

// downloadScriptsToLocalFromDB 从DB提取脚本并落盘到本地项目目录 (用于Docker构建等需要物理引用的步骤)
func (s *DeployService) downloadScriptsToLocalFromDB(projectId uint, routeId uint, localScriptPath string) error {
	var project system.TbProject
	if err := global.GVA_DB.First(&project, projectId).Error; err != nil {
		return fmt.Errorf("获取项目部署信息失败(project=%d): %w", projectId, err)
	}
	prepared, err := loadLocalScriptsForMaterialization(global.GVA_DB, []localScriptMaterializationRequest{{
		Project:    project,
		RouteID:    routeId,
		ScriptPath: localScriptPath,
	}})
	if err != nil {
		return err
	}
	return publishPreparedLocalScripts(global.GVA_DB, prepared, nil)
}

func resolveLocalScriptPath(route system.TbProjectRoute, localProjectPath string) string {
	if path := strings.TrimSpace(route.LocalScriptPath); path != "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(strings.TrimSpace(localProjectPath))
}

func safeLocalScriptFilePath(rootPath string, fileName string) (string, error) {
	rootPath = filepath.Clean(strings.TrimSpace(rootPath))
	fileName = filepath.Clean(filepath.FromSlash(strings.TrimSpace(fileName)))
	if rootPath == "" || rootPath == "." {
		return "", fmt.Errorf("脚本输出目录为空")
	}
	if fileName == "" || fileName == "." || filepath.IsAbs(fileName) || fileName == ".." || strings.HasPrefix(fileName, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("文件名必须是输出目录内的相对路径")
	}
	targetPath := filepath.Join(rootPath, fileName)
	relativePath, err := filepath.Rel(rootPath, targetPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("文件路径越过脚本输出目录")
	}
	return targetPath, nil
}

func isFrontendDeployProject(project system.TbProject) bool {
	language := strings.ToLower(strings.TrimSpace(project.ComputerLanguage))
	return language == "react" || language == "vue"
}

func isPythonDeployProject(project system.TbProject) bool {
	return strings.ToLower(strings.TrimSpace(project.ComputerLanguage)) == "python"
}

func isPythonDependencyDockerfile(fileName string) bool {
	return fileName == "Dockerfile.project" || fileName == "Dockerfile.deps"
}

func normalizePythonDependencyDockerfileForDeploy(content string) string {
	if strings.Contains(content, "build-base linux-headers") || !strings.Contains(content, "pip install") {
		return content
	}

	buildToolsCommand := "RUN if command -v apk >/dev/null 2>&1; then apk add --no-cache build-base linux-headers; fi\n\n"
	locationMarker := "WORKDIR /app\n\n"
	if index := strings.Index(content, locationMarker); index >= 0 {
		insertAt := index + len(locationMarker)
		return content[:insertAt] + buildToolsCommand + content[insertAt:]
	}

	return buildToolsCommand + content
}

func inferPythonAppPortForDeploy(project system.TbProject, composeContent string) int {
	if port, ok := extractAccessURLPort(project.AccessUrl); ok {
		return port
	}
	return extractFirstComposePort(composeContent)
}

func extractFirstComposePort(content string) int {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") || !strings.Contains(trimmed, ":") {
			continue
		}
		value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), `"'`)
		parts := strings.Split(value, ":")
		if len(parts) < 2 {
			continue
		}
		port, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	return 0
}

func normalizePythonComposeForDeploy(content string, appPort int, localProjectPath string) string {
	if appPort <= 0 || !strings.Contains(content, "environment:") {
		return content
	}

	required := []string{
		"      APP_HOST: 0.0.0.0",
		fmt.Sprintf("      APP_PORT: %d", appPort),
		"      BUSINESS_CLICKHOUSE_HOST: host.docker.internal",
		"      CH_HOST: host.docker.internal",
		"      REDIS_HOST: host.docker.internal",
	}
	if detectSnailJobPythonProject(localProjectPath) {
		required = append(required,
			"      SNAIL_SERVER_HOST: host.docker.internal",
			"      SNAIL_SERVER_PORT: 17888",
			"      SNAIL_HOST_IP: host.docker.internal",
			fmt.Sprintf("      SNAIL_HOST_PORT: %d", appPort),
		)
	}
	return ensureComposeEnvironmentLines(content, required)
}

func ensureComposeEnvironmentLines(content string, required []string) string {
	result := content
	insertAfter := "      APP_ENV: prod\n"
	for _, line := range required {
		key := strings.TrimSpace(strings.SplitN(line, ":", 2)[0]) + ":"
		if strings.Contains(result, key) {
			continue
		}
		if index := strings.Index(result, insertAfter); index >= 0 {
			insertAt := index + len(insertAfter)
			result = result[:insertAt] + line + "\n" + result[insertAt:]
			continue
		}
		environmentMarker := "    environment:\n"
		if index := strings.Index(result, environmentMarker); index >= 0 {
			insertAt := index + len(environmentMarker)
			result = result[:insertAt] + line + "\n" + result[insertAt:]
		}
	}
	return result
}

func inferBackendPortForFrontendDeploy(project system.TbProject, nginxContent string) int {
	if port, ok := inferGroupBackendHTTPPort(uint(project.GroupId)); ok {
		return port
	}
	return extractHostDockerInternalPort(nginxContent)
}

func extractHostDockerInternalPort(content string) int {
	const marker = "host.docker.internal:"
	index := strings.Index(content, marker)
	if index < 0 {
		return 0
	}
	start := index + len(marker)
	end := start
	for end < len(content) && content[end] >= '0' && content[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	port, err := strconv.Atoi(content[start:end])
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

func normalizeFrontendNginxScriptForDeploy(content string, backendPort int, localProjectPath string) string {
	if backendPort <= 0 {
		return content
	}

	result := content
	if !strings.Contains(result, "location /api/") {
		proxyPass := fmt.Sprintf("http://host.docker.internal:%d", backendPort)
		if detectFrontendAPIProxyStripPrefix(localProjectPath) {
			proxyPass += "/"
		}
		apiProxy := fmt.Sprintf(`    location /api/ {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

`, proxyPass)
		result = insertNginxLocation(result, apiProxy)
	} else if detectFrontendAPIProxyStripPrefix(localProjectPath) {
		result = rewriteAPIProxyPassToStripPrefix(result, backendPort)
	}

	return result
}

func rewriteAPIProxyPassToStripPrefix(content string, backendPort int) string {
	const apiLocationMarker = "location /api/"
	locationStart := strings.Index(content, apiLocationMarker)
	if locationStart < 0 {
		return content
	}
	blockStartOffset := strings.Index(content[locationStart:], "{")
	if blockStartOffset < 0 {
		return content
	}
	blockStart := locationStart + blockStartOffset
	blockEndOffset := strings.Index(content[blockStart:], "}")
	if blockEndOffset < 0 {
		return content
	}
	blockEnd := blockStart + blockEndOffset + 1
	apiLocation := content[locationStart:blockEnd]

	oldLine := fmt.Sprintf("proxy_pass http://host.docker.internal:%d;", backendPort)
	newLine := fmt.Sprintf("proxy_pass http://host.docker.internal:%d/;", backendPort)
	rewrittenLocation := strings.Replace(apiLocation, oldLine, newLine, 1)
	return content[:locationStart] + rewrittenLocation + content[blockEnd:]
}

func insertNginxLocation(content string, locationBlock string) string {
	locationMarker := "    location / {"
	if index := strings.Index(content, locationMarker); index >= 0 {
		return content[:index] + locationBlock + content[index:]
	}

	lastBrace := strings.LastIndex(content, "}")
	if lastBrace < 0 {
		return content + "\n\n" + locationBlock
	}
	return content[:lastBrace] + locationBlock + content[lastBrace:]
}

// buildTargetPath 构建目标文件路径
func buildTargetPath(localProjectPath, fileName string) string {
	if localProjectPath != "" {
		return filepath.Join(strings.TrimRight(localProjectPath, "/"), fileName)
	}
	return fileName
}

func resolveServerProjectPathFromNodeConfig(server system.TbServer, project system.TbProject, route system.TbProjectRoute) string {
	raw := strings.TrimSpace(server.ExtendParams)
	if raw == "" {
		return ""
	}

	var groups map[string]map[string]string
	if err := json.Unmarshal([]byte(raw), &groups); err != nil {
		return ""
	}
	if len(groups) == 0 {
		return ""
	}

	pathKeys := []string{
		"serverProjectPath",
		"server_project_path",
		"projectPath",
		"project_path",
		"deployPath",
		"deploy_path",
		"remotePath",
		"remote_path",
		"workDir",
		"workdir",
		"path",
	}

	groupNames := []string{
		project.ProjectName,
		fmt.Sprintf("project:%s", project.ProjectName),
		fmt.Sprintf("项目:%s", project.ProjectName),
		route.RouteKey,
		route.RouteName,
		project.ComputerLanguage,
		"deploy",
		"deployment",
		"remote",
		"远程部署",
		"部署",
	}

	for _, groupName := range groupNames {
		if path := lookupExtendParamPath(groups, groupName, pathKeys); path != "" {
			return path
		}
	}

	projectScopedKeys := []string{
		project.ProjectName,
		fmt.Sprintf("%s.path", project.ProjectName),
		fmt.Sprintf("%s.serverProjectPath", project.ProjectName),
		fmt.Sprintf("%s.%s.path", project.ProjectName, route.RouteKey),
		fmt.Sprintf("%s.%s.serverProjectPath", project.ProjectName, route.RouteKey),
		fmt.Sprintf("%s_%s_path", project.ProjectName, route.RouteKey),
	}
	for _, groupItems := range groups {
		for _, key := range projectScopedKeys {
			if value := strings.TrimSpace(groupItems[key]); value != "" {
				return value
			}
		}
	}

	return ""
}

func lookupExtendParamPath(groups map[string]map[string]string, groupName string, pathKeys []string) string {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return ""
	}
	for name, items := range groups {
		if !strings.EqualFold(strings.TrimSpace(name), groupName) {
			continue
		}
		for _, key := range pathKeys {
			if value := strings.TrimSpace(items[key]); value != "" {
				return value
			}
		}
	}
	return ""
}

// processLocalDeploy 本地机器运行与部署流
func (s *DeployService) processLocalDeploy(cmdUtil *utils.CmdUtil, projectId uint, routeId uint, localProjectPath, localScriptPath, localExecuteCommand, localStartCommand string) error {
	global.GVA_LOG.Info(fmt.Sprintf("开始执行本地部署, 项目ID: %d", projectId))

	// 释放DB记录的通用和本地脚本到本地构建机器工作区
	if err := s.downloadScriptsToLocalFromDB(projectId, routeId, localScriptPath); err != nil {
		return err
	}

	// 1. 本地打包/编译命令
	if localExecuteCommand != "" {
		if err := cmdUtil.PackageProject(localProjectPath, localExecuteCommand); err != nil {
			return fmt.Errorf("执行本地打包前置命令失败: %w", err)
		}
	}

	// 2. 本地挂载启动命令
	if localStartCommand != "" {
		if err := cmdUtil.RunBackgroundCommand(localProjectPath, localStartCommand); err != nil {
			return fmt.Errorf("挂起本地常驻服务失败: %w", err)
		}
	}

	return nil
}

// ProcessStopWithLog 执行本地关闭（支持日志流式推送）
func (s *DeployService) ProcessStopWithLog(projectId uint, targetEnv string, logCh chan string) error {
	sendLog := func(msg string) {
		if logCh != nil {
			select {
			case logCh <- msg:
			default:
			}
		}
	}

	sendLog("📋 开始获取项目信息...")
	project, err := ProjectServiceApp.GetProjectById(projectId)
	if err != nil {
		return fmt.Errorf("获取项目信息失败: %w", err)
	}
	sendLog(fmt.Sprintf("✅ 项目: %s (ID=%d)", project.ProjectName, projectId))

	// 提取匹配的路由配置
	var currentRoute system.TbProjectRoute
	for _, r := range project.Routes {
		if r.RouteKey == targetEnv || fmt.Sprintf("%d", r.ID) == targetEnv {
			currentRoute = r
			break
		}
	}
	if currentRoute.RouteKey == "" {
		return fmt.Errorf("未找到对应环境标识的路由配置: %s", targetEnv)
	}

	localProjectPath := currentRoute.LocalProjectPath
	if localProjectPath == "" {
		localProjectPath = project.LocalProjectPath
	}
	localScriptPath := resolveLocalScriptPath(currentRoute, localProjectPath)

	stopCommand := currentRoute.LocalStopCommand
	if stopCommand == "" {
		stopCommand = "docker compose down" // 默认关闭命令
	}

	cmdUtil := &utils.CmdUtil{LogCh: logCh}
	sendLog("🛑 执行本地关闭流程...")

	// 1. 聚合项目先准备子项目脚本，确保独立脚本目录被清理后仍能正常关闭。
	if err := s.prepareAggregateChildDeployScripts(project, currentRoute, logCh); err != nil {
		return err
	}

	// 2. 下载脚本到本地
	if err := s.downloadScriptsToLocalFromDB(projectId, currentRoute.ID, localScriptPath); err != nil {
		return err
	}
	sendLog("✅ 脚本已同步")

	// 3. 执行关闭命令
	sendLog(fmt.Sprintf("⏹️ 执行关闭命令: %s", stopCommand))
	if err := cmdUtil.PackageProject(localProjectPath, stopCommand); err != nil {
		return fmt.Errorf("执行关闭命令失败: %w", err)
	}

	sendLog("✅ 关闭完成")
	return nil
}
