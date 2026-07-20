# Context Router Docker Build Contract Design

## Context

The context-router frontend package intentionally changed its normal `build`
script from `next build` to `sh scripts/build-isolated.sh`. The isolation script
builds in `/tmp/agent-context-router-next-build.*` so verification builds do not
overwrite the `.next` directory used by a development server. Its EXIT trap
deletes the temporary build directory.

The active VibeDeploy frontend Dockerfile still runs `npm run build` and then
copies `/app/.next` into the runtime stage. The isolated build succeeds but
never publishes an artifact at `/app/.next`, so BuildKit fails at the runner
stage with `/app/.next: not found`.

The earlier aggregate-route correction is working: VibeDeploy materializes and
executes the frontend child route before this failure. This design corrects the
frontend image build contract and does not change aggregate route detection,
Docker networking, or restart policy handling.

## Goals

- Preserve isolated host verification builds.
- Give Docker builds an explicit command that produces `/app/.next`.
- Keep the package-level build contract visible in `package.json`.
- Update the persistent VibeDeploy database template, not only generated files.
- Protect the database update with optimistic locking and a recorded rollback.
- Prove the correction using a clean image build and real aggregate deployment.

## Non-goals

- Changing the Next.js application, routes, or runtime API configuration.
- Changing `build-isolated.sh` or making it publish artifacts back into the
  source checkout.
- Generalizing VibeDeploy's Dockerfile generation for every Next.js project.
- Changing container ports, shared-network membership, or restart policy.
- Editing historical or soft-deleted deployment scripts.

## Considered Approaches

### 1. Dedicated Docker build script (selected)

Add `"build:docker": "next build"` alongside the existing isolated `build`
script. Change the active database-backed Dockerfile to run
`npm run build:docker`.

This preserves both contracts: host verification stays isolated, while Docker
builds in `/app` and produces the artifact consumed by the runtime stage. It is
also discoverable to developers through the package manifest.

### 2. Copy isolated output back to the project directory

Extend `build-isolated.sh` with an option that copies `.next` back before
cleanup. This couples two different use cases and risks reintroducing the dev
server artifact corruption that the isolation change was created to prevent.

### 3. Invoke the Next binary directly in the Dockerfile

Replace the Dockerfile command with
`RUN ./node_modules/.bin/next build`. This would work, but it hides the Docker
build contract outside `package.json` and makes local reproduction less clear.

## Design

### Source package contract

Modify the context-router frontend `package.json` scripts to retain:

```json
"build": "sh scripts/build-isolated.sh"
```

and add:

```json
"build:docker": "next build"
```

The existing `build-output.test.ts` contract test will assert both values. It
will continue to verify that the normal build changes into a temporary build
directory. This makes accidental removal or redirection of either contract a
test failure.

No source component or Next configuration change is required.

### Persistent deployment template

Update only active script ID 902 for project 126 / route 202, after verifying
that all identifiers, active-state predicates, and the current content hash
still match the diagnosed record.

The Dockerfile changes exactly one command:

```dockerfile
RUN npm run build:docker
```

The runner stage remains unchanged and continues to copy `/app/.next`, install
production dependencies, and run `next start` on port 3000.

The generated file under
`deploy/frontend/web_next/local_full/Dockerfile` is not edited as an authority.
VibeDeploy rematerializes it from PostgreSQL during the next deployment.

### Optimistic database update and rollback

Before updating the database:

1. Read script ID 902 with `deleted_at IS NULL`.
2. Record its complete content and MD5 hash in the implementation evidence.
3. Update only when the row ID, project ID, route ID, active state, and old MD5
   all match.
4. Require exactly one affected row.
5. Read the row back and verify the new content and hash.

If source tests, clean Docker build, aggregate deployment, or runtime checks
fail because of this change, restore the old database content using the new MD5
as the optimistic guard and restore the source manifest from its recorded
pre-change content. Generated files and failed containers may then be
regenerated from the restored template.

## Data Flow

After the correction, the deployment path is:

1. VibeDeploy materializes project 126 / route 202 from PostgreSQL.
2. Compose builds the frontend image using the repository root as context.
3. The builder installs dependencies and runs `npm run build:docker` in `/app`.
4. Next.js writes `/app/.next`.
5. The runtime stage copies `/app/.next` and the Next configuration.
6. Compose starts `agent-context-router-web` on the external
   `vibedeploy-shared` network.
7. The aggregate start script observes HTTP 200 on port 6061 and completes.

## Testing

### Source contract tests

- First change `build-output.test.ts` to require `build:docker` and observe it
  fail before editing `package.json`.
- Add the script and rerun the focused test.
- Run the complete frontend test suite and lint.
- Run the isolated `npm run build` and verify it still leaves the source `.next`
  unchanged.

### Docker build tests

- Materialize the updated active route through VibeDeploy or an equivalent
  read-only script download path.
- Verify the generated Dockerfile contains the exact instruction
  `RUN npm run build:docker` and contains no line whose complete instruction is
  `RUN npm run build`.
- Run a no-cache Compose build for the frontend route. The builder must produce
  `/app/.next`, and the runner-stage copy must complete.

### End-to-end acceptance

- Trigger project 125 route 200 through the same deployment stream used by the
  UI.
- Confirm the backend route and frontend route both complete.
- Confirm the stream ends with the aggregate completion event.
- Verify backend health on port 10061 and frontend HTTP 200 on port 6061.
- Verify `agent-context-router-backend` and `agent-context-router-web` are
  running with exit code 0, restart policy `no`, and only
  `vibedeploy-shared`.
- Confirm Docker still has no project-specific user-defined networks.

## Acceptance Criteria

- `npm run build` remains isolated and does not overwrite the source `.next`.
- `npm run build:docker` creates `.next` in its working directory.
- Active database script ID 902 persists the Docker-specific build command.
- A no-cache frontend image build completes through the runner-stage COPY.
- The real context-router aggregate deployment ends successfully.
- Both runtime services are healthy and follow the shared-network and
  `restart=no` invariants.
- On any failure attributable to the change, both the source manifest and
  database template are restored to their recorded previous state.
