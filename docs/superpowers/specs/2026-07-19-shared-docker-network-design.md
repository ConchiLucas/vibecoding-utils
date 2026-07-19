# Shared Docker Network Design

## Objective

Make `vibedeploy-shared` the only user-defined Docker network used by VibeDeploy-managed business applications and middleware. Docker's built-in `bridge`, `host`, and `none` networks remain untouched.

The design must fix the current deployment failure:

```text
network vibedeploy-shared declared as external, but could not be found
```

It must also prevent project-specific default networks from returning when containers are recreated.

## Confirmed Current State

- All 32 database-backed business Compose files reference `vibedeploy-shared` as an external network. Materialized AI Compose files specifically map Compose `default` to that network.
- The network does not currently exist, so Compose correctly refuses to create it and deployment exits with status 1.
- The current AI deployment scripts invoke `docker compose` without first creating the external network.
- Five middleware containers are running on separate networks:
  - `minio` on `minio_default`
  - `snail-job-server` on `snail-job_default`
  - `local-nginx` on `nginx_default`
  - `postgres16` on `postgresql_default`
  - `redis-7.2` on `redis_default`
- Their five Compose files have no explicit network configuration, so Compose recreates those default networks.
- VibeDeploy's two checked-in Compose files use a custom `177.7.0.0/16` network and static container IP addresses.

## Selected Architecture

VibeDeploy owns the lifecycle of one Docker bridge network named `vibedeploy-shared`.

The ownership model has three layers:

1. **Host bootstrap:** a small idempotent script creates the external network before a containerized VibeDeploy instance or middleware Compose stack is started.
2. **Runtime self-healing:** VibeDeploy verifies or creates the network at backend startup and immediately before every local deployment.
3. **Persistent Compose configuration:** every managed Compose file maps its `default` network directly to external `vibedeploy-shared`, preventing `<project>_default` networks from being created.

The network is created with the bridge driver and a VibeDeploy ownership label. Docker chooses the subnet to avoid conflicts with existing local address pools.

```bash
docker network create \
  --driver bridge \
  --label com.vibedeploy.managed=true \
  vibedeploy-shared
```

## Components

### 1. Shared-network service

Add a focused backend component responsible only for shared-network lifecycle operations.

Responsibilities:

- Inspect `vibedeploy-shared`.
- Create it when absent.
- Treat an already-created network as success, including concurrent creation races.
- Return actionable errors when Docker is unavailable or the network cannot be created.
- Emit deployment logs describing whether the network was reused or created.

Docker command execution will be injected behind a small interface so behavior can be unit-tested without requiring a live Docker daemon.

### 2. Startup reconciliation

Both HTTP and desktop backend entry points will start asynchronous network reconciliation after application initialization.

- If Docker is ready, ensure the network immediately.
- If Docker is still starting, use the existing bounded Docker-readiness behavior.
- Failure is logged but does not prevent the VibeDeploy UI from starting.

This covers normal native development startup.

### 3. Deployment guard

Before any local deployment downloads or executes scripts, VibeDeploy synchronously ensures the shared network exists.

If the guard fails:

- Stop before image build or Compose execution.
- Return a concise error identifying the network operation.
- Write the same diagnostic to the deployment log stream.

This is the authoritative self-healing path and fixes the reported failure even when the network was deleted after VibeDeploy started.

### 4. Host bootstrap script

Add an idempotent repository script for the circular bootstrap case: a containerized VibeDeploy cannot create an external network until its own container has started.

The script will:

- Return success when `vibedeploy-shared` already exists.
- Create the labeled bridge network when absent.
- Be called by repository-managed Compose/restart entry points before `docker compose up`.
- Be safe to run repeatedly.

### 5. Persistent Compose migration

The following five middleware Compose files will map `default` to the external shared network:

- `/Users/conchi/docker-compose/minio/docker-compose.yml`
- `/Users/conchi/docker-compose/snail-job/docker-compose.yml`
- `/Users/conchi/docker-compose/nginx/docker-compose.yml`
- `/Users/conchi/database/postgresql/docker-compose.yml`
- `/Users/conchi/middleware/redis/docker-compose.yml`

Each receives this top-level block:

```yaml
networks:
  default:
    name: vibedeploy-shared
    external: true
```

No per-service network list is required because services use Compose's default network.

The repository files `docker-compose.yml` and `deploy/docker-compose/docker-compose.yaml` receive the same external-default mapping. Their custom IPAM and `ipv4_address` entries are removed. Inter-container communication must use Docker DNS names rather than static IP addresses.

The 32 database-backed business Compose files already reference the shared external network. Implementation will structurally validate every service definition before migration. Any script that still permits an implicit project default or attaches another user-defined network will be normalized in the database to the external-default form above; scripts already in that form remain unchanged.

## Live Migration Sequence

The migration is ordered to avoid service interruption:

1. Capture the current Docker network and container membership inventory.
2. Create `vibedeploy-shared`.
3. Validate all edited Compose files with `docker compose config`.
4. Recreate each middleware container one at a time from its updated Compose file. This is required because `docker network connect` changes only live attachments; it does not replace the container's persisted `HostConfig.NetworkMode`.
5. After each recreation, verify the container is running, its health/port is available, and both `HostConfig.NetworkMode` and its sole user-defined attachment are `vibedeploy-shared`.
6. Verify all five containers appear in `docker network inspect vibedeploy-shared`.
7. Remove every old user-defined project network that has no remaining endpoints.

Cleanup explicitly preserves:

- `bridge`
- `host`
- `none`
- `vibedeploy-shared`

Compose replaces the five middleware container objects so their persisted network mode is correct. No image, volume, bind-mounted file, or database data is deleted.

If an old network still has an endpoint after the known containers are migrated, cleanup stops for that network and reports the endpoint instead of forcefully disconnecting an unidentified container.

## Addressing and DNS

All cross-container communication uses stable container or service DNS names on `vibedeploy-shared`.

Expected middleware names are:

- `minio`
- `snail-job-server`
- `local-nginx`
- `postgres16`
- `redis-7.2`

Business Compose service/container names must remain globally unique. This is already required by Docker when explicit `container_name` values are used. The migration does not change published host ports.

## Error Handling

- **Docker unavailable:** startup reconciliation logs a warning; deployment guard fails before executing project scripts.
- **Network create race:** re-inspect after a failed create and accept success if the network now exists.
- **Wrong existing network driver:** fail with a clear diagnostic rather than silently reusing an incompatible network.
- **Container recreation:** recreate one middleware at a time and verify service readiness before continuing.
- **Old network still in use:** retain it and report its remaining endpoints.
- **Compose validation failure:** do not recreate middleware containers; fix the file before continuing cleanup.

## Testing Strategy

### Automated tests

Use test-driven development for the backend network lifecycle component:

- Existing network is reused without creation.
- Missing network is created with the expected driver, name, and label.
- Concurrent-create race is accepted after successful re-inspection.
- Docker errors are returned with useful context.
- Local deployment invokes the network guard before project commands.
- Database-backed Compose validation detects an implicit project default or a second user-defined network.

Validate both repository Compose files using `docker compose config`.
Validate all 32 database-backed Compose scripts structurally after materialization or normalization.

### Live verification

- `docker network inspect vibedeploy-shared` succeeds.
- The shared network contains all five middleware containers.
- Each middleware container has `HostConfig.NetworkMode=vibedeploy-shared` and no old user-defined network attachment.
- Docker network inventory contains only the three built-in networks plus `vibedeploy-shared` among project/runtime networks.
- Re-run the AI database aggregate deployment and confirm the previous external-network error is gone.
- Verify newly created business containers join `vibedeploy-shared` and no `<project>_default` network is created.

## Rollback

Before removing old networks, record their names, Compose declarations, attached containers, and persisted `HostConfig.NetworkMode` values. If validation fails after any middleware container has been recreated, restore that stack's prior Compose network declaration and recreate each already-migrated container one at a time in reverse migration order. After every rollback recreation, verify the original persisted network mode and service readiness before continuing.

Once persistent Compose files are migrated and validated, rollback uses the same procedure: restore the prior files and old Compose networks, then recreate affected containers individually. Volumes and application data are unaffected by either migration or rollback.

## Out of Scope

- Deleting Docker built-in networks.
- Changing host-published ports.
- Renaming containers or services.
- Migrating Docker volumes or bind-mounted data.
- Introducing fixed IP addresses on the shared network.
