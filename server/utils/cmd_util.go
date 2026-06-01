package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// CmdUtil 命令行工具
type CmdUtil struct{
	LogCh chan string // 可选：日志推送 channel，为 nil 时仅输出到控制台
}

// enrichPath 为 exec.Cmd 补充完整 PATH 环境变量
// macOS 下通过 GUI / Wails 启动的进程不会加载用户的 shell profile（~/.zshrc / ~/.bash_profile），
// 导致 docker、npm、go 等安装在 /usr/local/bin 或 /opt/homebrew/bin 的工具找不到。
// 此函数将常见的可执行文件路径追加到 PATH 中解决此问题。
func enrichPath(cmd *exec.Cmd) {
	// 继承已被 init() 补全的进程环境变量即可
	// 注意：cmd.Env 为 nil 时 Go 默认继承 os.Environ()，所以这里无需额外操作。
	// 保留此函数以保持调用点一致性，将来有需要时可在此做命令级别的定制。
}

func init() {
	// 在进程启动时立刻补全 PATH，确保后续所有 exec.Command / exec.LookPath 都能找到工具链。
	// macOS GUI 应用（Wails）不会加载 ~/.zshrc 等 shell profile，PATH 极度精简。
	extraPaths := []string{
		"/usr/local/bin",
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/sbin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	}

	if runtime.GOOS != "windows" {
		home, _ := os.UserHomeDir()
		if home != "" {
			extraPaths = append(extraPaths,
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".local", "bin"),
			)
		}
	}

	currentPath := os.Getenv("PATH")
	existingPaths := make(map[string]bool)
	for _, p := range strings.Split(currentPath, string(os.PathListSeparator)) {
		existingPaths[p] = true
	}
	newPaths := []string{currentPath}
	for _, p := range extraPaths {
		if !existingPaths[p] {
			newPaths = append(newPaths, p)
		}
	}

	os.Setenv("PATH", strings.Join(newPaths, string(os.PathListSeparator)))
}

// getWriters 根据是否有 LogCh 返回合适的 stdout/stderr writer
func (c *CmdUtil) getWriters() (stdout io.Writer, stderr io.Writer) {
	if c.LogCh != nil {
		logWriter := NewLogWriter(c.LogCh, os.Stdout)
		logWriterErr := NewLogWriter(c.LogCh, os.Stderr)
		return logWriter, logWriterErr
	}
	return os.Stdout, os.Stderr
}

// sendLog 向 channel 发送阶段性日志（不阻塞）
func (c *CmdUtil) sendLog(msg string) {
	if c.LogCh != nil {
		select {
		case c.LogCh <- msg:
		default:
		}
	}
}

// RunNpmBuild 执行npm打包命令
func (c *CmdUtil) RunNpmBuild(projectPath, command string) error {
	zap.L().Info(fmt.Sprintf("开始执行npm打包: %s, 命令: %s", projectPath, command))
	c.sendLog(fmt.Sprintf("🔨 开始执行npm打包: %s", command))
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("命令为空")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = projectPath
	enrichPath(cmd)
	cmd.Stdout, cmd.Stderr = c.getWriters()
	return cmd.Run()
}

// ZipDirectory 压缩目录为zip文件
func (c *CmdUtil) ZipDirectory(sourceDir, targetPath string) error {
	zap.L().Info(fmt.Sprintf("开始压缩目录: %s -> %s", sourceDir, targetPath))
	c.sendLog(fmt.Sprintf("📦 开始压缩目录: %s", sourceDir))

	zipFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("创建zip文件失败: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	ignoreDirs := map[string]bool{
		".git":           true,
		".venv":          true,
		"venv":           true,
		"env":            true,
		"__pycache__":    true,
		".pytest_cache":  true,
		".idea":          true,
		".vscode":        true,
		"node_modules":   true,
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() && ignoreDirs[info.Name()] {
			return filepath.SkipDir
		}

		// 跳过zip文件本身
		if path == targetPath {
			return nil
		}
		// 计算相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			_, err = zipWriter.Create(relPath + "/")
			return err
		}
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
}

// PackageProject 执行项目打包命令
func (c *CmdUtil) PackageProject(projectPath, command string) error {
	zap.L().Info(fmt.Sprintf("开始执行打包命令: %s, 命令: %s", projectPath, command))
	c.sendLog(fmt.Sprintf("🔨 开始执行打包命令: %s", command))
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("命令为空")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = projectPath
	enrichPath(cmd)
	cmd.Stdout, cmd.Stderr = c.getWriters()
	return cmd.Run()
}

// BuildImage 构建Docker镜像
func (c *CmdUtil) BuildImage(projectPath, imageName string) error {
	zap.L().Info(fmt.Sprintf("开始构建Docker镜像: %s", imageName))
	c.sendLog(fmt.Sprintf("🐳 开始构建Docker镜像: %s", imageName))
	// 去掉.tar后缀作为镜像名
	imageTag := strings.TrimSuffix(imageName, ".tar")
	cmd := exec.Command("docker", "build", "-t", imageTag, ".")
	cmd.Dir = projectPath
	enrichPath(cmd)
	cmd.Stdout, cmd.Stderr = c.getWriters()
	return cmd.Run()
}

// SaveImage 保存Docker镜像为tar包
func (c *CmdUtil) SaveImage(projectPath, fileName string) error {
	zap.L().Info(fmt.Sprintf("开始保存Docker镜像: %s", fileName))
	c.sendLog(fmt.Sprintf("💾 开始保存Docker镜像: %s", fileName))
	imageTag := strings.TrimSuffix(fileName, ".tar")
	outputPath := filepath.Join(projectPath, fileName)
	cmd := exec.Command("docker", "save", "-o", outputPath, imageTag)
	cmd.Dir = projectPath
	enrichPath(cmd)
	cmd.Stdout, cmd.Stderr = c.getWriters()
	return cmd.Run()
}

// RunBackgroundCommand 在环境后台执行常驻任务
func (c *CmdUtil) RunBackgroundCommand(projectPath, command string) error {
	zap.L().Info(fmt.Sprintf("开始本地执行后台挂起命令: %s, 命令: %s", projectPath, command))
	c.sendLog(fmt.Sprintf("🚀 开始挂起后台服务: %s", command))
	fullCommand := fmt.Sprintf("nohup %s > local_deploy.log 2>&1 &", command)
	cmd := exec.Command("bash", "-c", fullCommand)
	cmd.Dir = projectPath
	enrichPath(cmd)
	return cmd.Run()
}
