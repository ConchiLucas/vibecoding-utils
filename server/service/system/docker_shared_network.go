package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
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
