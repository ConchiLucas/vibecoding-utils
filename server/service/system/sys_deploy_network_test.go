package system

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunLocalDeployWithSharedNetworkStopsBeforeDeployOnNetworkFailure(t *testing.T) {
	logCh := make(chan string, 4)
	deployCalled := false

	err := runLocalDeployWithSharedNetwork(
		logCh,
		func(context.Context) (SharedDockerNetworkResult, error) {
			return SharedDockerNetworkResult{}, errors.New("docker daemon unavailable")
		},
		func() error {
			deployCalled = true
			return nil
		},
	)

	if err == nil {
		t.Fatal("runLocalDeployWithSharedNetwork() unexpectedly succeeded")
	}
	if deployCalled {
		t.Fatal("local deployment ran after the network guard failed")
	}
	if !strings.Contains(err.Error(), SharedDockerNetworkName) || !strings.Contains(err.Error(), "docker daemon unavailable") {
		t.Fatalf("error = %q, want network name and Docker diagnostic", err)
	}
	logs := drainDeployLogs(logCh)
	if !strings.Contains(logs, "检查共享 Docker 网络") || !strings.Contains(logs, "不可用") {
		t.Fatalf("logs = %q, want guard start and failure diagnostics", logs)
	}
}

func TestRunLocalDeployWithSharedNetworkRunsDeployAfterGuard(t *testing.T) {
	logCh := make(chan string, 4)
	order := make([]string, 0, 2)

	err := runLocalDeployWithSharedNetwork(
		logCh,
		func(context.Context) (SharedDockerNetworkResult, error) {
			order = append(order, "ensure")
			return SharedDockerNetworkResult{Created: true}, nil
		},
		func() error {
			order = append(order, "deploy")
			return nil
		},
	)

	if err != nil {
		t.Fatalf("runLocalDeployWithSharedNetwork() error = %v", err)
	}
	if strings.Join(order, ",") != "ensure,deploy" {
		t.Fatalf("call order = %v, want ensure before deploy", order)
	}
	logs := drainDeployLogs(logCh)
	if !strings.Contains(logs, "已创建") {
		t.Fatalf("logs = %q, want created-network confirmation", logs)
	}
}

func drainDeployLogs(logCh chan string) string {
	close(logCh)
	var logs []string
	for line := range logCh {
		logs = append(logs, line)
	}
	return strings.Join(logs, "\n")
}
