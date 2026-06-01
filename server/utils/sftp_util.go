package utils

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// SftpUtil SFTP工具
type SftpUtil struct{}

// createSSHClient 创建SSH客户端连接
func createSSHClient(host string, port int, username, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}
	return client, nil
}

// createSFTPClient 创建SFTP客户端
func createSFTPClient(sshClient *ssh.Client) (*sftp.Client, error) {
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("SFTP连接失败: %w", err)
	}
	return sftpClient, nil
}

// UploadLocalPath 上传本地文件到远程服务器
func (s *SftpUtil) UploadLocalPath(localPath, host, username, password, remotePath, fileName string) error {
	return s.UploadLocalPathWithPort(localPath, host, 22, username, password, remotePath, fileName)
}

// UploadLocalPathWithPort 上传本地文件到远程服务器（指定端口）
func (s *SftpUtil) UploadLocalPathWithPort(localPath, host string, port int, username, password, remotePath, fileName string) error {
	sshClient, err := createSSHClient(host, port, username, password)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sftpClient, err := createSFTPClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 确保远程目录存在
	sftpClient.MkdirAll(remotePath)

	// 打开本地文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建远程文件
	remoteFilePath := filepath.Join(remotePath, fileName)
	dstFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制文件
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	zap.L().Info(fmt.Sprintf("文件上传成功: %s -> %s", localPath, remoteFilePath))
	return nil
}

// UploadMemory 上传内存数据到远程服务器
func (s *SftpUtil) UploadMemory(reader io.Reader, host string, port int, username, password, remotePath, fileName string) error {
	sshClient, err := createSSHClient(host, port, username, password)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sftpClient, err := createSFTPClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 确保远程目录存在
	sftpClient.MkdirAll(remotePath)

	// 创建远程文件
	remoteFilePath := filepath.Join(remotePath, fileName)
	dstFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制内容
	_, err = io.Copy(dstFile, reader)
	if err != nil {
		return fmt.Errorf("上传内存文件失败: %w", err)
	}

	zap.L().Info(fmt.Sprintf("内存文件上传成功: %s", remoteFilePath))
	return nil
}

// ExecuteShell 在远程服务器上执行Shell命令
func (s *SftpUtil) ExecuteShell(host string, port int, username, password, workDir, command string) error {
	sshClient, err := createSSHClient(host, port, username, password)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 在工作目录下执行命令
	fullCmd := fmt.Sprintf("cd %s && %s", workDir, command)
	output, err := session.CombinedOutput(fullCmd)
	if err != nil {
		// 检查是否是网络错误（命令可能已经执行但连接断开）
		if _, ok := err.(*net.OpError); ok {
			zap.L().Warn("SSH连接在命令执行后断开（可能是正常的服务重启）", zap.String("output", string(output)))
			return nil
		}
		return fmt.Errorf("执行命令失败: %w, output: %s", err, string(output))
	}

	zap.L().Info(fmt.Sprintf("远程命令执行成功: %s, 输出: %s", fullCmd, string(output)))
	return nil
}
