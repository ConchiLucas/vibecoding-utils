package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	SharedDockerNetworkName  = "vibedeploy-shared"
	sharedDockerNetworkLabel = "com.vibedeploy.managed=true"
)

type dockerCommandRunner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type execDockerCommandRunner struct{}

func (execDockerCommandRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

type SharedDockerNetworkResult struct {
	Created bool
}

type SharedDockerNetworkService struct {
	runner dockerCommandRunner
}

var SharedDockerNetworkServiceApp = newSharedDockerNetworkService(execDockerCommandRunner{})

func newSharedDockerNetworkService(runner dockerCommandRunner) *SharedDockerNetworkService {
	return &SharedDockerNetworkService{runner: runner}
}

func (s *SharedDockerNetworkService) Ensure(ctx context.Context) (SharedDockerNetworkResult, error) {
	driver, inspectErr := s.inspectDriver(ctx)
	if inspectErr == nil {
		return SharedDockerNetworkResult{}, validateSharedDockerNetworkDriver(driver)
	}

	createOutput, createErr := s.runner.Run(
		ctx,
		"network", "create",
		"--driver", "bridge",
		"--label", sharedDockerNetworkLabel,
		SharedDockerNetworkName,
	)
	driver, finalInspectErr := s.inspectDriver(ctx)
	if finalInspectErr == nil {
		if err := validateSharedDockerNetworkDriver(driver); err != nil {
			return SharedDockerNetworkResult{}, err
		}
		return SharedDockerNetworkResult{Created: createErr == nil}, nil
	}

	diagnostic := strings.TrimSpace(string(createOutput))
	if diagnostic == "" && createErr != nil {
		diagnostic = createErr.Error()
	}
	return SharedDockerNetworkResult{}, fmt.Errorf(
		"创建 Docker 网络 %s 失败: %s (inspect: %v)",
		SharedDockerNetworkName,
		diagnostic,
		finalInspectErr,
	)
}

func (s *SharedDockerNetworkService) inspectDriver(ctx context.Context) (string, error) {
	output, err := s.runner.Run(ctx, "network", "inspect", "--format", "{{.Driver}}", SharedDockerNetworkName)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func validateSharedDockerNetworkDriver(driver string) error {
	if driver != "bridge" {
		return fmt.Errorf("Docker 网络 %s 的驱动为 %q，期望 bridge", SharedDockerNetworkName, driver)
	}
	return nil
}

func (s *SharedDockerNetworkService) EnsureOnStartup() {
	go func() {
		err := reconcileSharedDockerNetworkOnStartup(waitForDockerReady, s.Ensure)
		if err != nil {
			zap.L().Warn("共享 Docker 网络启动自愈失败", zap.String("network", SharedDockerNetworkName), zap.Error(err))
			return
		}
		zap.L().Info("共享 Docker 网络启动检查完成", zap.String("network", SharedDockerNetworkName))
	}()
}

func reconcileSharedDockerNetworkOnStartup(
	waitForDocker func() error,
	ensure func(context.Context) (SharedDockerNetworkResult, error),
) error {
	if err := waitForDocker(); err != nil {
		return fmt.Errorf("等待 Docker 就绪失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := ensure(ctx)
	if err != nil {
		return fmt.Errorf("确保网络 %s 失败: %w", SharedDockerNetworkName, err)
	}
	return nil
}
