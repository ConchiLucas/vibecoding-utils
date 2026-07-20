# Context Router Docker Build Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve context-router's isolated host build while giving its Docker image build a package command that creates `/app/.next`, persist that command in VibeDeploy, and prove the aggregate deployment succeeds.

**Architecture:** The frontend package exposes two explicit contracts: `build` remains the temporary-directory verification build, and `build:docker` runs `next build` in the container working directory. The tested source commit is fast-forwarded into the repository used by VibeDeploy before the active PostgreSQL Dockerfile template is changed under an optimistic MD5 guard. A real aggregate SSE deployment then rematerializes the template, builds the image, and verifies runtime state.

**Tech Stack:** Next.js 15, Node.js 22, Node test runner through `tsx`, Docker BuildKit/Compose, PostgreSQL 16, VibeDeploy Go SSE API, shell verification.

---

## File Structure

- Modify: `/Users/conchi/workforce/python_workforce/agent-context-router/frontend/lib/build-output.test.ts` — package build-contract regression test.
- Modify: `/Users/conchi/workforce/python_workforce/agent-context-router/frontend/package.json` — add the Docker-specific build command while preserving the isolated host command.
- Create temporarily: `/private/tmp/context-router-Dockerfile.before` — exact rollback copy of database script 902.
- Create temporarily: `/private/tmp/context-router-Dockerfile.expected` — exact Dockerfile with the Docker-specific build command.
- Modify persistently: PostgreSQL `easy_deploy.tb_project_script`, row ID 902 — authoritative VibeDeploy template for project 126 / route 202.
- Materialized by VibeDeploy: `/Users/conchi/workforce/python_workforce/agent-context-router/deploy/frontend/web_next/local_full/Dockerfile` — generated output; never edit it as the source of truth.

### Task 1: Create the isolated source worktree and establish its baseline

**Files:**
- Create worktree: `/Users/conchi/workforce/go_workforce/vibecoding-utils/.worktrees/agent-context-router-docker-build-contract`
- Read: `/Users/conchi/workforce/python_workforce/agent-context-router/frontend/package-lock.json`
- Install ignored dependencies: `frontend/node_modules/`

- [ ] **Step 1: Create the source branch in the existing ignored worktree host**

Run:

```bash
git -C /Users/conchi/workforce/python_workforce/agent-context-router worktree add \
  /Users/conchi/workforce/go_workforce/vibecoding-utils/.worktrees/agent-context-router-docker-build-contract \
  -b codex/context-router-docker-build-contract
```

Expected: the worktree is created from the current context-router `main`, while the original checkout retains only its pre-existing untracked `deploy/` directory.

- [ ] **Step 2: Install the exact locked frontend dependencies**

Run:

```bash
npm ci
```

Working directory:

```text
/Users/conchi/workforce/go_workforce/vibecoding-utils/.worktrees/agent-context-router-docker-build-contract/frontend
```

Expected: exit status 0 and no tracked-file changes.

- [ ] **Step 3: Run the focused and full source baseline**

Run:

```bash
npm test
npm run lint
```

Expected: both commands exit 0. If a real test or lint assertion fails after dependencies are installed, stop and report the baseline failure before modifying source.

### Task 2: Add the Docker build contract using red-green TDD

**Files:**
- Modify: `frontend/lib/build-output.test.ts:7-19`
- Modify: `frontend/package.json:5-10`

- [ ] **Step 1: Write the failing Docker contract test**

Append this test to `frontend/lib/build-output.test.ts`:

```typescript
test("Docker build writes Next output in its working directory", () => {
  assert.equal(packageJson.scripts["build:docker"], "next build");
});
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
npm exec -- tsx --test lib/build-output.test.ts
```

Expected: one failing assertion reports actual `undefined` versus expected `next build`; the existing isolated-build assertion still passes.

- [ ] **Step 3: Add the minimal package implementation**

Change the scripts object in `frontend/package.json` to exactly:

```json
"scripts": {
  "dev": "next dev",
  "lint": "eslint . --max-warnings=0",
  "build": "sh scripts/build-isolated.sh",
  "build:docker": "next build",
  "test": "tsx --test lib/*.test.ts"
}
```

- [ ] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
npm exec -- tsx --test lib/build-output.test.ts
```

Expected: both build-output tests pass.

- [ ] **Step 5: Prove both build contracts with real Next builds**

Run:

```bash
test ! -e .next
npm run build:docker
test -f .next/BUILD_ID
before_mtime=$(stat -f '%m' .next)
npm run build
after_mtime=$(stat -f '%m' .next)
test "$before_mtime" = "$after_mtime"
```

Expected: `build:docker` creates `.next/BUILD_ID`; the normal isolated build succeeds without changing the source `.next` directory timestamp.

- [ ] **Step 6: Run the complete frontend verification**

Run:

```bash
npm test
npm run lint
git status --short
```

Expected: tests and lint pass; only `frontend/package.json` and `frontend/lib/build-output.test.ts` are tracked modifications because `.next` and `node_modules` are ignored.

- [ ] **Step 7: Commit the tested source contract**

Run:

```bash
git add frontend/package.json frontend/lib/build-output.test.ts
git commit -m "fix: add context router Docker build contract"
```

Expected: one commit containing only the two source files.

### Task 3: Integrate the tested source commit into the deployment checkout

**Files:**
- Modify through fast-forward: `/Users/conchi/workforce/python_workforce/agent-context-router/frontend/package.json`
- Modify through fast-forward: `/Users/conchi/workforce/python_workforce/agent-context-router/frontend/lib/build-output.test.ts`
- Preserve: `/Users/conchi/workforce/python_workforce/agent-context-router/deploy/`

- [ ] **Step 1: Recheck that the deployment checkout has no tracked changes**

Run:

```bash
git -C /Users/conchi/workforce/python_workforce/agent-context-router status --short
```

Expected: only `?? deploy/`. Stop if any tracked change appears.

- [ ] **Step 2: Fast-forward `main` to the tested source branch**

Run:

```bash
git -C /Users/conchi/workforce/python_workforce/agent-context-router merge \
  --ff-only codex/context-router-docker-build-contract
```

Expected: `main` advances by exactly the Task 2 commit; the untracked deployment directory remains present.

- [ ] **Step 3: Verify the deployed checkout exposes both contracts**

Run:

```bash
node -e 'const p=require("./frontend/package.json"); if (p.scripts.build !== "sh scripts/build-isolated.sh" || p.scripts["build:docker"] !== "next build") process.exit(1)'
git status --short
```

Expected: the Node assertion exits 0 and Git still reports only `?? deploy/`.

### Task 4: Prepare and validate the persistent Dockerfile replacement

**Files:**
- Create: `/private/tmp/context-router-Dockerfile.before`
- Create: `/private/tmp/context-router-Dockerfile.expected`
- Read: PostgreSQL `easy_deploy.tb_project_script`, ID 902

- [ ] **Step 1: Create the exact rollback Dockerfile using `apply_patch`**

Create `/private/tmp/context-router-Dockerfile.before` with:

```dockerfile
FROM node:22-bookworm AS builder

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./

ARG NEXT_PUBLIC_CONTEXT_ROUTER_API_URL=http://127.0.0.1:10061
ENV NEXT_PUBLIC_CONTEXT_ROUTER_API_URL=${NEXT_PUBLIC_CONTEXT_ROUTER_API_URL}

RUN npm run build

FROM node:22-bookworm-slim AS runner

WORKDIR /app
ENV NODE_ENV=production

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --omit=dev

COPY --from=builder /app/.next ./.next
COPY --from=builder /app/next.config.ts ./next.config.ts

EXPOSE 3000

CMD ["./node_modules/.bin/next", "start", "--hostname", "0.0.0.0", "--port", "3000"]
```

- [ ] **Step 2: Create the exact replacement Dockerfile using `apply_patch`**

Create `/private/tmp/context-router-Dockerfile.expected` with the same content except the builder command is:

```dockerfile
RUN npm run build:docker
```

- [ ] **Step 3: Verify database identity and old hash immediately before mutation**

Run:

```bash
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select id,project_id,route_id,file_name,script_type,md5(content),length(content)
   from tb_project_script
   where id=902 and project_id=126 and route_id=202
     and file_name='Dockerfile' and script_type=1 and deleted_at is null"
```

Expected exactly:

```text
902|126|202|Dockerfile|1|c340d78e95386b2ae71ec5caa7ec630e|646
```

- [ ] **Step 4: Prove the rollback file is byte-identical to the database**

Run:

```bash
test "$(md5 -q /private/tmp/context-router-Dockerfile.before)" = \
  "$(docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
    "select md5(content) from tb_project_script where id=902 and deleted_at is null")"
rg -n '^RUN npm run build$' /private/tmp/context-router-Dockerfile.before
rg -n '^RUN npm run build:docker$' /private/tmp/context-router-Dockerfile.expected
```

Expected: hashes match; each exact command matches only its intended file.

### Task 5: Update script 902 with an optimistic guard

**Files:**
- Read: `/private/tmp/context-router-Dockerfile.before`
- Read: `/private/tmp/context-router-Dockerfile.expected`
- Modify: PostgreSQL `easy_deploy.tb_project_script`, ID 902

- [ ] **Step 1: Update exactly the diagnosed active row**

Run in one shell so encoded content is not printed:

```bash
new_content_b64=$(base64 -i /private/tmp/context-router-Dockerfile.expected | tr -d '\n')
docker exec postgres16 psql -U conchi -d easy_deploy \
  -v new_content_b64="$new_content_b64" \
  -c "update tb_project_script
      set content=convert_from(decode(:'new_content_b64','base64'),'UTF8'), updated_at=now()
      where id=902
        and project_id=126
        and route_id=202
        and file_name='Dockerfile'
        and script_type=1
        and deleted_at is null
        and md5(content)='c340d78e95386b2ae71ec5caa7ec630e';"
```

Expected: `UPDATE 1`. Stop without deploying for any other row count.

- [ ] **Step 2: Read back and validate the persistent template**

Run:

```bash
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select content from tb_project_script where id=902 and deleted_at is null" \
  | rg -n '^RUN npm run build:docker$'
! docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select content from tb_project_script where id=902 and deleted_at is null" \
  | rg -n '^RUN npm run build$'
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select id,md5(content),length(content) from tb_project_script where id=902 and deleted_at is null"
```

Expected: the exact Docker command is present, the legacy complete instruction is absent, and one new hash is reported for ID 902.

- [ ] **Step 3: Retain the exact guarded rollback procedure**

If any later failure is attributable to this change, run:

```bash
current_hash=$(docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select md5(content) from tb_project_script where id=902 and deleted_at is null")
old_content_b64=$(base64 -i /private/tmp/context-router-Dockerfile.before | tr -d '\n')
docker exec postgres16 psql -U conchi -d easy_deploy \
  -v old_content_b64="$old_content_b64" \
  -v current_hash="$current_hash" \
  -c "update tb_project_script
      set content=convert_from(decode(:'old_content_b64','base64'),'UTF8'), updated_at=now()
      where id=902 and project_id=126 and route_id=202
        and file_name='Dockerfile' and script_type=1 and deleted_at is null
        and md5(content)=:'current_hash';"
```

Expected when used: `UPDATE 1`, followed by old MD5 `c340d78e95386b2ae71ec5caa7ec630e`. Restore the old package manifest and regression test in a new source commit rather than resetting Git history.

### Task 6: Run the real aggregate deployment and verify the result

**Files:**
- Materialize: `/Users/conchi/workforce/python_workforce/agent-context-router/deploy/**`
- Inspect: Docker images, containers, logs, networks, and VibeDeploy SSE output

- [ ] **Step 1: Trigger project 125 / route 200 through the UI-equivalent API**

Run in one shell so the token is never printed:

```bash
token=$(curl -fsS http://127.0.0.1:23638/base/login \
  -H 'Content-Type: application/json' \
  --data '{"username":"admin","password":"123456"}' \
  | jq -er '.data.token')
curl -N --fail-with-body --max-time 1800 \
  "http://127.0.0.1:23638/project/deployStream/125?env=200&token=${token}" \
  | tee /private/tmp/context-router-deploy-stream.log
```

Expected: the stream ends with `event: done` and `部署完成`; it contains no `event: error` or `/app/.next: not found`.

- [ ] **Step 2: Verify the generated Dockerfile came from the updated template**

Run:

```bash
rg -n '^RUN npm run build:docker$' \
  /Users/conchi/workforce/python_workforce/agent-context-router/deploy/frontend/web_next/local_full/Dockerfile
! rg -n '^RUN npm run build$' \
  /Users/conchi/workforce/python_workforce/agent-context-router/deploy/frontend/web_next/local_full/Dockerfile
```

Expected: only the Docker-specific complete instruction matches.

- [ ] **Step 3: Verify frontend and backend runtime health**

Run:

```bash
curl -fsS http://127.0.0.1:10061/health
curl -fsSI http://127.0.0.1:6061/
docker inspect agent-context-router-backend agent-context-router-web \
  --format '{{.Name}} running={{.State.Running}} exit={{.State.ExitCode}} restart={{.HostConfig.RestartPolicy.Name}} networks={{range $key, $value := .NetworkSettings.Networks}}{{$key}} {{end}}'
```

Expected: both HTTP requests succeed; both containers are running with exit code 0, `restart=no`, and only `vibedeploy-shared`.

- [ ] **Step 4: Inspect build/runtime logs and network inventory**

Run:

```bash
docker logs --tail 160 agent-context-router-web
docker logs --tail 160 agent-context-router-backend
docker network ls --format '{{.Name}}\t{{.Driver}}'
```

Expected: no Next artifact, crash-loop, bind, or fatal backend connection error. Record any pre-existing project-specific network separately; do not treat an unrelated `agent-context-router-postgres-1` lifecycle issue as evidence that the Docker build contract failed.

- [ ] **Step 5: Record final source and persistent-template evidence**

Run:

```bash
git -C /Users/conchi/workforce/python_workforce/agent-context-router log -1 --oneline
git -C /Users/conchi/workforce/python_workforce/agent-context-router status --short
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select id,project_id,route_id,md5(content),length(content)
   from tb_project_script where id=902 and deleted_at is null"
```

Expected: `main` contains the tested build-contract commit, only generated `deploy/` remains untracked, and active script 902 retains the new hash.

### Task 7: Roll back only if the new contract itself fails

**Files:**
- Restore if needed: PostgreSQL `easy_deploy.tb_project_script`, ID 902
- Restore if needed: context-router frontend package manifest and regression test

- [ ] **Step 1: Classify any deployment failure before rollback**

The change is invalid only if `npm run build:docker` fails, `/app/.next` is still absent, the updated Dockerfile cannot reach the runner stage, or the resulting web container cannot start because of the new build command. Registry, APT/NPM mirror, unrelated backend, existing database-container network, and port failures are diagnosed separately and do not silently rewrite verified source or database state.

- [ ] **Step 2: Restore both persistence layers when the contract is invalid**

Run the guarded SQL rollback retained in Task 5. Then restore `frontend/package.json` and `frontend/lib/build-output.test.ts` to their pre-change content with `apply_patch`, run `npm test` and `npm run lint`, and commit:

```bash
git add frontend/package.json frontend/lib/build-output.test.ts
git commit -m "revert: restore context router build contract"
```

Expected: database script 902 has MD5 `c340d78e95386b2ae71ec5caa7ec630e`; source again exposes only the isolated `build` script; no destructive reset or unrelated file removal occurs.
