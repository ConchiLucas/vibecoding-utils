package system

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type dockerRunnerStep struct {
	args   []string
	output string
	err    error
}

type fakeDockerCommandRunner struct {
	t     *testing.T
	steps []dockerRunnerStep
	calls [][]string
}

func (f *fakeDockerCommandRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.t.Helper()
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(f.steps) == 0 {
		f.t.Fatalf("unexpected docker call: %v", args)
	}
	step := f.steps[0]
	f.steps = f.steps[1:]
	if !reflect.DeepEqual(args, step.args) {
		f.t.Fatalf("docker args = %v, want %v", args, step.args)
	}
	return []byte(step.output), step.err
}

func (f *fakeDockerCommandRunner) assertDone(t *testing.T) {
	t.Helper()
	if len(f.steps) != 0 {
		t.Fatalf("%d expected docker calls were not made", len(f.steps))
	}
}

func inspectSharedNetworkStep(output string, err error) dockerRunnerStep {
	return dockerRunnerStep{
		args:   []string{"network", "inspect", "--format", "{{.Driver}}", SharedDockerNetworkName},
		output: output,
		err:    err,
	}
}

func createSharedNetworkStep(output string, err error) dockerRunnerStep {
	return dockerRunnerStep{
		args: []string{
			"network", "create",
			"--driver", "bridge",
			"--label", sharedDockerNetworkLabel,
			SharedDockerNetworkName,
		},
		output: output,
		err:    err,
	}
}

func TestSharedDockerNetworkReusesExistingBridge(t *testing.T) {
	runner := &fakeDockerCommandRunner{t: t, steps: []dockerRunnerStep{
		inspectSharedNetworkStep("bridge\n", nil),
	}}
	service := newSharedDockerNetworkService(runner)

	result, err := service.Ensure(context.Background())

	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if result.Created {
		t.Fatal("Ensure() reported an existing network as newly created")
	}
	runner.assertDone(t)
}

func TestSharedDockerNetworkCreatesMissingNetwork(t *testing.T) {
	runner := &fakeDockerCommandRunner{t: t, steps: []dockerRunnerStep{
		inspectSharedNetworkStep("", errors.New("not found")),
		createSharedNetworkStep("network-id\n", nil),
		inspectSharedNetworkStep("bridge\n", nil),
	}}
	service := newSharedDockerNetworkService(runner)

	result, err := service.Ensure(context.Background())

	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if !result.Created {
		t.Fatal("Ensure() did not report creating the missing network")
	}
	runner.assertDone(t)
}

func TestSharedDockerNetworkAcceptsConcurrentCreateRace(t *testing.T) {
	runner := &fakeDockerCommandRunner{t: t, steps: []dockerRunnerStep{
		inspectSharedNetworkStep("", errors.New("not found")),
		createSharedNetworkStep("", errors.New("already exists")),
		inspectSharedNetworkStep("bridge\n", nil),
	}}
	service := newSharedDockerNetworkService(runner)

	result, err := service.Ensure(context.Background())

	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if result.Created {
		t.Fatal("Ensure() reported a concurrently-created network as created by this process")
	}
	runner.assertDone(t)
}

func TestSharedDockerNetworkReturnsActionableDockerError(t *testing.T) {
	runner := &fakeDockerCommandRunner{t: t, steps: []dockerRunnerStep{
		inspectSharedNetworkStep("", errors.New("daemon unavailable")),
		createSharedNetworkStep("", errors.New("cannot connect")),
		inspectSharedNetworkStep("", errors.New("daemon unavailable")),
	}}
	service := newSharedDockerNetworkService(runner)

	_, err := service.Ensure(context.Background())

	if err == nil {
		t.Fatal("Ensure() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), SharedDockerNetworkName) || !strings.Contains(err.Error(), "cannot connect") {
		t.Fatalf("Ensure() error = %q, want network name and create diagnostic", err)
	}
	runner.assertDone(t)
}

func TestSharedDockerNetworkRejectsWrongDriver(t *testing.T) {
	runner := &fakeDockerCommandRunner{t: t, steps: []dockerRunnerStep{
		inspectSharedNetworkStep("overlay\n", nil),
	}}
	service := newSharedDockerNetworkService(runner)

	_, err := service.Ensure(context.Background())

	if err == nil || !strings.Contains(err.Error(), "overlay") {
		t.Fatalf("Ensure() error = %v, want incompatible driver diagnostic", err)
	}
	runner.assertDone(t)
}
