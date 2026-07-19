# Shared Docker Network Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `vibedeploy-shared` the only user-defined Docker network for VibeDeploy-managed business containers and middleware, with automatic creation before local deployments and persistent Compose configuration.

**Architecture:** Add a testable backend lifecycle service that inspects and creates the labeled bridge network, call it asynchronously at both backend startup entry points, and enforce it synchronously before every local deployment. Normalize database-backed Compose scripts to external `default`, add an idempotent host bootstrap script, migrate checked-in and middleware Compose files, then migrate live containers before deleting only verified-unused legacy networks.

**Tech Stack:** Go 1.24, standard `os/exec`, `go.yaml.in/yaml/v3`, Bash, Docker Compose, PostgreSQL/GORM, Go `testing`.

---

## Task 1: Add the shared-network lifecycle service

**Files:**
- Create: `server/service/system/docker_shared_network.go`
- Create: `server/service/system/docker_shared_network_test.go`

- [x] Write failing tests for an existing bridge network, a missing network, a create race, an inspect failure, and a wrong driver.
- [x] Run `cd server && go test ./service/system -run 'TestSharedDockerNetwork' -count=1` and confirm the tests fail because the service does not exist.
- [x] Add constants and an injectable runner:

```go
const (
    SharedDockerNetworkName  = "vibedeploy-shared"
    sharedDockerNetworkLabel = "com.vibedeploy.managed=true"
)

type dockerCommandRunner interface {
    Run(ctx context.Context, args ...string) ([]byte, error)
}

type SharedDockerNetworkService struct {
    runner dockerCommandRunner
}
```

- [x] Implement `Ensure(ctx)` by inspecting `{{.Driver}}`, accepting only `bridge`, creating with `network create --driver bridge --label ...`, and re-inspecting after a failed create to tolerate races.
- [x] Add `EnsureOnStartup()` that waits for Docker in a goroutine, calls `Ensure`, and logs without blocking application startup.
- [x] Run the focused tests and confirm they pass.
- [x] Commit: `git commit -m "feat: manage shared Docker network"`

## Task 2: Enforce the network before local deployment

**Files:**
- Modify: `server/service/system/sys_deploy.go`
- Create: `server/service/system/sys_deploy_network_test.go`

- [x] Write a failing test around a small deployment-guard helper proving a network error is returned and logged before the local deployment callback runs.
- [x] Run `cd server && go test ./service/system -run 'TestRunLocalDeployWithSharedNetwork' -count=1` and confirm failure.
- [x] Extract a helper with injectable `ensure` and `deploy` callbacks, then call it in `ProcessDeployWithLog` before `prepareAggregateChildDeployScripts`:

```go
func runLocalDeployWithSharedNetwork(logCh chan string, ensure func(context.Context) error, deploy func() error) error {
    sendDeployLog(logCh, "🌐 检查共享 Docker 网络...")
    if err := ensure(context.Background()); err != nil {
        return fmt.Errorf("共享 Docker 网络 %s 不可用: %w", SharedDockerNetworkName, err)
    }
    sendDeployLog(logCh, "✅ 共享 Docker 网络已就绪")
    return deploy()
}
```

- [x] Keep remote deployments unchanged.
- [x] Run focused and full `service/system` tests.
- [x] Commit: `git commit -m "feat: guard local deploys with shared network"`

## Task 3: Reconcile the network at both backend startup paths

**Files:**
- Modify: `server/main.go`
- Modify: `server/cmd/http/main.go`
- Modify: `server/service/system/docker_shared_network_test.go`

- [x] Add a test for startup reconciliation delegating to the bounded Docker readiness check and swallowing/logging failures.
- [x] Call `SharedDockerNetworkServiceApp.EnsureOnStartup()` immediately after backend initialization in both entry points, independently of whether any project group has auto-start enabled.
- [x] Retain project-group auto-start behavior and let the synchronous deployment guard handle its deployments.
- [x] Run `cd server && go test ./service/system -run 'TestSharedDockerNetwork|TestResolveAutoStart' -count=1`.
- [x] Commit: `git commit -m "feat: reconcile shared network on startup"`

## Task 4: Normalize stored Compose scripts

**Files:**
- Create: `server/service/system/docker_compose_network.go`
- Create: `server/service/system/docker_compose_network_test.go`
- Modify: `server/service/system/sys_deploy.go`
- Modify: `server/service/system/docker_shared_network.go`

- [x] Write table-driven failing tests for:
  - an already-correct external-default Compose document;
  - an implicit default network;
  - service-level project networks;
  - multiple user-defined networks;
  - malformed YAML.
- [x] Parse through `yaml.Node`, remove service-level `networks`, and replace the top-level `networks` mapping with:

```yaml
networks:
  default:
    name: vibedeploy-shared
    external: true
```

- [x] Preserve service definitions, volumes, secrets, environment variables, and published ports.
- [x] In `downloadScriptsToLocalFromDB`, normalize every Compose filename (`docker-compose.yml`, `docker-compose.yaml`, `compose.yml`, `compose.yaml`) before writing, persisting changed content back to `tb_project_script`.
- [x] Add asynchronous startup reconciliation over all non-remote stored Compose scripts so the existing 32 scripts are normalized even before their next deployment.
- [x] Return malformed Compose errors with project/script IDs and do not write partial changes.
- [x] Run `cd server && go test ./service/system -run 'TestNormalizeComposeSharedNetwork|TestReconcileStoredCompose' -count=1` and then the full package suite.
- [x] Commit: `git commit -m "feat: normalize managed Compose networks"`

## Task 5: Add host bootstrap and migrate repository Compose files

**Files:**
- Create: `scripts/ensure-docker-network.sh`
- Create: `scripts/ensure-docker-network_test.sh`
- Modify: `scripts/restart-dev.sh`
- Modify: `docker-compose.yml`
- Modify: `deploy/docker-compose/docker-compose.yaml`

- [x] Write a fake-`docker` shell test proving the script reuses an existing network and creates a missing network with the required driver and label.
- [x] Implement an idempotent bootstrap script using exact `docker network inspect` and `docker network create` commands.
- [x] Invoke it from `scripts/restart-dev.sh` before backend startup so native development also repairs the network early.
- [x] Replace repository custom IPAM/static IPs and per-service `network` attachments with external `default`.
- [x] Run `bash scripts/ensure-docker-network_test.sh`.
- [x] Run `docker compose -f docker-compose.yml config` and `docker compose -f deploy/docker-compose/docker-compose.yaml config` after the shared network exists.
- [x] Commit: `git commit -m "feat: bootstrap shared Docker network"`

## Task 6: Migrate middleware Compose declarations

**Files:**
- Modify: `/Users/conchi/docker-compose/minio/docker-compose.yml`
- Modify: `/Users/conchi/docker-compose/snail-job/docker-compose.yml`
- Modify: `/Users/conchi/docker-compose/nginx/docker-compose.yml`
- Modify: `/Users/conchi/database/postgresql/docker-compose.yml`
- Modify: `/Users/conchi/middleware/redis/docker-compose.yml`

- [x] Append the external-default block to each file without changing images, ports, volumes, commands, environment, health checks, or restart policies.
- [x] Validate each file from its own directory with `docker compose config`.
- [x] Confirm config output resolves every service's default network to `vibedeploy-shared`.
- [x] Do not recreate the containers yet.

## Task 7: Migrate the live Docker network topology

**Files:**
- Runtime state only; no file edits.

- [x] Record exact network IDs, names, labels, drivers, and endpoint container names.
- [x] Create `vibedeploy-shared` with bridge driver and ownership label if absent.
- [x] Validate the five middleware Compose files before changing live containers.
- [x] Recreate `minio`, `snail-job-server`, `local-nginx`, `postgres16`, and `redis-7.2` one at a time from their updated Compose files so persisted `HostConfig.NetworkMode` becomes `vibedeploy-shared`.
- [x] Verify all five appear in `docker network inspect vibedeploy-shared`.
- [x] Verify all five containers are running, healthy where health checks exist, and have only `vibedeploy-shared` as both network mode and user-defined attachment.
- [x] Remove each old user-defined network only when its endpoint list is empty; preserve `bridge`, `host`, `none`, and `vibedeploy-shared`.
- [x] If any legacy network retains an unidentified endpoint, leave it in place and report it.

## Task 8: End-to-end verification

**Files:**
- Modify: `task_plan.md`
- Modify: `progress.md`

- [x] Run `cd server && go test ./service/system -count=1`.
- [x] Run `cd server && go test ./... -count=1` (the target package passes; the repository-wide command retains the pre-existing missing embedded frontend and upload-config build failures recorded in the verification log).
- [x] Run the shell bootstrap tests and `bash -n` on all three scripts.
- [x] Validate all seven edited Compose files.
- [x] Query `tb_project_script` and confirm all 32 active business Compose scripts structurally resolve to external-default only.
- [x] Re-run the AI aggregate deployment and confirm the missing external-network error is gone.
- [x] Verify newly created business containers join `vibedeploy-shared` and no project default network is created.
- [x] Inspect final Docker state: only built-ins plus `vibedeploy-shared` remain, unless a legacy network was deliberately retained because it still had an unidentified endpoint.
- [x] Review `git diff`, `git status`, recent commits, and scan changed files for placeholders (`TODO`, `FIXME`, temporary stubs).
- [ ] Update planning logs with exact test results and any retained network exceptions.
