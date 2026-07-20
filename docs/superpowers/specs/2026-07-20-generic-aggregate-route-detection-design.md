# Generic Aggregate Route Detection Design

## Context

VibeDeploy materializes the deployment scripts referenced by an aggregate
`start.sh` before it executes that script. This makes a fresh checkout behave
the same as a computer that already has generated `deploy/` files.

The preflight currently runs only when `TbProject.ComputerLanguage` is exactly
`前后端 docker-compose`. The project editor and other deployment code also
support `docker-compose`, and task center and context router use that valid
value. Those aggregate deployments therefore skip dependency materialization
silently and fail later with `start.sh: No such file or directory`.

The fix must recognize aggregate deployment routes by supported metadata, not
by project IDs, project names, group names, or repository-specific paths. It
must preserve the existing all-or-nothing materialization behavior and must not
run aggregate parsing for an ordinary single-project Compose route.

## Goals

- Treat supported Docker Compose project-type aliases consistently.
- Materialize referenced child-route scripts for every local aggregate route.
- Keep ordinary Compose project routes outside the aggregate preflight.
- Keep aggregate discovery independent of project names, numeric IDs, and
  fixed directory layouts.
- Preserve existing path-safety, uniqueness, preflight, atomic publication,
  and rollback guarantees.
- Verify the behavior with task-center-shaped mixed-language data and existing
  English/stock-style data.

## Non-goals

- Rewriting aggregate shell scripts or changing their supported command syntax.
- Automatically treating every `start.sh` that invokes another script as an
  aggregate route.
- Migrating individual database records to a single display value.
- Changing Docker networking, restart policies, port mappings, or project
  source paths.
- Generalizing Python dependency-image build policy. Python wheel and compiler
  decisions remain a separate concern because they depend on each project's
  dependency graph.

## Considered Approaches

### 1. Semantic project type plus aggregate route metadata (selected)

Normalize supported Docker Compose labels and require a local route whose key
starts with `frontend_backend_`. This uses stable application metadata already
present in the editor, route generator, and database. It fixes both known
affected project groups and prevents ordinary `local_full` Compose routes from
entering the aggregate parser.

### 2. Inspect every local `start.sh`

Parse any start script that appears to reference child scripts. This is more
permissive but can reinterpret an ordinary project's internal bootstrap steps
as cross-project dependencies. It also makes an empty or dynamically generated
script fail the aggregate contract unexpectedly.

### 3. Normalize existing database rows only

Rewrite `docker-compose` to `前后端 docker-compose`. This can unblock current
rows but leaves the editor capable of recreating the mismatch and leaves the
narrow code gate unchanged. It is a data patch, not a general fix.

## Design

### Canonical Compose classification

Add one package-level predicate for Docker Compose project types. It trims and
lowercases the stored value, collapses repeated whitespace, and accepts an
explicit supported-value set. This covers the currently generated and
historical forms:

- `docker-compose`
- `docker compose`
- `前后端 docker-compose`
- `前后端 docker compose`

Values merely containing those tokens, such as `not-docker-compose`, are not
accepted. This avoids turning malformed metadata into an orchestration entry.
The predicate must not fall back to `ProjectName` containing `compose`.
Project names are mutable labels and are not a reliable deployment contract.

All aggregate materialization checks use this predicate. In particular,
child-route resolution also uses it when excluding aggregate Compose projects
from the candidate child set, replacing its current exact string comparison.

### Aggregate route classification

Add a focused predicate equivalent to:

```text
isDockerComposeProjectType(project.ComputerLanguage)
AND route.ServerId == 0
AND lower(trim(route.RouteKey)) starts with "frontend_backend_"
```

The three conditions have distinct roles:

- Compose project type states that the project is an orchestration entry.
- `ServerId == 0` restricts materialization to local deployment routes.
- The `frontend_backend_` prefix distinguishes aggregate routes from ordinary
  Compose routes such as `local_full` and `local_incremental`.

Both `frontend_backend_full` and `frontend_backend_incremental` are eligible.
Unknown or empty route keys are not inferred from route names or shell content.
This is deliberate: ambiguous metadata should not silently change behavior.

### Deployment flow

`prepareAggregateChildDeployScripts` remains the single pre-execution entry
point. Its initial exact-language guard is replaced by the aggregate-route
predicate.

For a non-aggregate route, it returns without filesystem or database changes.
For an aggregate route, the existing flow remains unchanged:

1. Load the selected aggregate route's unique active local `start.sh`.
2. Parse ordered `$ROOT_DIR/.../start.sh` references.
3. Resolve every reference to exactly one active local route in the same group.
4. Validate that each child route has exactly one non-empty active `start.sh`.
5. Load the aggregate route and all referenced child scripts before publishing.
6. Publish all prepared scripts atomically, retaining database/filesystem
   rollback behavior on failure.
7. Execute the aggregate route only after successful publication.

No project-specific paths or language-specific child branches are added.
Python, Java, Go, Vue, and React children continue through the same route and
script abstractions.

### Call-site consistency

Manual deployment, group-wide deployment, and startup linkage already converge
on the same deployment service. They therefore receive the corrected preflight
without separate project-specific handling.

Existing compose-related helpers outside materialization may continue to use
broader rules for their own purposes, such as choosing a log command. This
change does not globally redefine every use of the word "aggregate". The new
route predicate is intentionally scoped to dependency materialization.

## Error Handling and Safety

- Supported alias values no longer cause a silent preflight skip.
- A Compose project with an ordinary route key remains a no-op for aggregate
  preflight, even if its script has internal `sh` or `bash` calls.
- Aggregate routes keep the existing explicit failures for missing, ambiguous,
  remote-only, empty, duplicate, absolute, traversal, unsupported, and
  self-referential child entries.
- All dependency validation completes before any target script is published.
- Existing generated files and script-content state retain their current
  rollback guarantees when publication fails.
- Deployment logs continue to emit dependency parsing, child count, per-child
  preparation, and successful materialization messages.

## Test Strategy

### Unit and service tests

Use test-driven changes in `aggregate_route_materializer_test.go`:

1. Table-test Compose classification for all supported aliases, whitespace,
   casing, and negative ordinary-language values.
2. Table-test aggregate-route classification:
   - both canonical and alias Compose values are accepted;
   - full and incremental aggregate keys are accepted;
   - remote routes are rejected;
   - ordinary `local_full` Compose routes are rejected;
   - a project name containing `compose` without Compose type is rejected.
3. Add a task-center-shaped materialization test using stored type
   `docker-compose` and referenced Python, Java, and React child routes. Begin
   with no generated child directories and assert all referenced scripts are
   published before aggregate execution can begin.
4. Preserve the existing seven-child English-style mixed-language test using
   `前后端 docker-compose`.
5. Add coverage that a child project stored under a supported Compose alias is
   excluded from child candidates.
6. Preserve all existing unsafe-reference, ambiguity, missing-script,
   preflight-before-write, and rollback tests.

Run at minimum:

```sh
go test ./server/service/system -run 'Aggregate|DockerCompose' -count=1
go test ./server/service/system -count=1
go test -race ./server/service/system -count=1
go vet ./server/service/system
```

If the repository's actual Go module root requires commands from `server/`, use
the equivalent `go test ./service/system` and `go vet ./service/system` forms.
Any known repository-wide failure unrelated to this package must be reproduced
on the unchanged baseline and reported separately rather than hidden.

### Live acceptance

After automated tests pass:

1. Ensure the three task-center child deployment directories referenced by the
   aggregate script are absent, or move them to a recoverable temporary backup.
2. Trigger task center's local `frontend_backend_full` route through the same
   VibeDeploy endpoint used by the UI.
3. Confirm the log reports three discovered dependencies before the first child
   `start.sh` runs.
4. Confirm the Python Worker, Java server, and React frontend scripts are
   materialized and the expected containers start successfully.
5. Verify each resulting business container uses restart policy `no` and only
   the shared `vibedeploy-shared` network.
6. Run representative aggregate preflights/deployments for the canonical
   `前后端 docker-compose` form (English or stock) and the alias form (context
   router when safe) to guard both classification paths.
7. Verify an ordinary single-project Compose `local_full` route still bypasses
   aggregate dependency parsing.

Any temporarily moved generated directories are restored on test failure. No
database project-type rewrite is part of acceptance or rollback.

## Acceptance Criteria

- A fresh task-center deployment no longer depends on generated files left by
  another computer.
- Both supported Compose naming families enter aggregate preflight when paired
  with a local `frontend_backend_*` route.
- Ordinary Compose routes do not enter aggregate parsing.
- The solution contains no task-center, English, stock, or context-router
  project IDs, names, group names, or fixed paths.
- Existing atomic publication and rollback tests remain green.
- Scoped service tests, race tests, vet, and live task-center deployment pass.
- Task-center containers run with restart policy `no` on only
  `vibedeploy-shared`.
