# Stock Python Wheel-Only Build Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist a reliable ARM64 wheel-only dependency build for `stock-python-back`, then prove the complete stock history backtest deployment starts successfully through VibeDeploy.

**Architecture:** VibeDeploy materializes deployment files from PostgreSQL on every run, so script ID 541 is the source of truth. Replace only that script's unnecessary Debian compiler layer using an optimistic MD5 guard, validate the replacement against the real stock repository, then run project 88 route 159 and inspect the resulting four containers.

**Tech Stack:** PostgreSQL 16, Docker Desktop/BuildKit, Docker Compose, Python 3.11 slim ARM64, pip binary wheels, VibeDeploy Go SSE API, shell verification.

---

## File Structure

- Create temporarily: `/private/tmp/stock-Dockerfile.project.before` — exact rollback content for database script 541.
- Create temporarily: `/private/tmp/stock-Dockerfile.project.expected` — exact wheel-only replacement used for preflight and database update.
- Modify persistently: PostgreSQL `easy_deploy.tb_project_script`, row ID 541 — source of the generated `deploy/backend/stock_python_back/build_project/Dockerfile.project`.
- Materialized by VibeDeploy: `/Users/conchi/workforce/stock_workforce/deploy/**` — generated deployment scripts; never edit these as the source of truth.

### Task 1: Establish the template contract and rollback artifact

**Files:**
- Create: `/private/tmp/stock-Dockerfile.project.before`
- Create: `/private/tmp/stock-Dockerfile.project.expected`
- Inspect: PostgreSQL `easy_deploy.tb_project_script`, ID 541

- [ ] **Step 1: Save the exact current template with `apply_patch`**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

RUN if command -v apk >/dev/null 2>&1; then apk add --no-cache build-base linux-headers; fi

ENV PYTHONUNBUFFERED=1
ENV PIP_NO_CACHE_DIR=1

RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential gcc \
    && rm -rf /var/lib/apt/lists/*

COPY stock_view/stock_python_back/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
```

- [ ] **Step 2: Run the pre-change contract and verify it fails**

Run:

```bash
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select content from tb_project_script where id=541 and deleted_at is null" \
  | rg -n 'apt-get|build-essential|(^|[^[:alnum:]_])gcc([^[:alnum:]_]|$)|apk add'
```

Expected: matches the Alpine probe and Debian compiler installation, proving the current persistent template violates the wheel-only contract.

- [ ] **Step 3: Create the exact replacement with `apply_patch`**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

ENV PYTHONUNBUFFERED=1
ENV PIP_NO_CACHE_DIR=1

COPY stock_view/stock_python_back/requirements.txt .
RUN python -m pip install \
    --only-binary=:all: \
    --retries 8 \
    --timeout 60 \
    --no-cache-dir \
    -r requirements.txt \
    -i https://pypi.tuna.tsinghua.edu.cn/simple
```

- [ ] **Step 4: Verify the replacement passes the structural contract**

Run:

```bash
! rg -n 'apt-get|build-essential|(^|[^[:alnum:]_])gcc([^[:alnum:]_]|$)|apk add' /private/tmp/stock-Dockerfile.project.expected
rg -n -- '--only-binary=:all:|--retries 8|--timeout 60' /private/tmp/stock-Dockerfile.project.expected
```

Expected: the forbidden-package search has no matches; all three pip reliability flags match.

### Task 2: Prove the replacement builds on the real host

**Files:**
- Read: `/private/tmp/stock-Dockerfile.project.expected`
- Read: `/Users/conchi/workforce/stock_workforce/stock_view/stock_python_back/requirements.txt`

- [ ] **Step 1: Build the dependency image from the real repository context**

Run:

```bash
docker build \
  --network vibedeploy-shared \
  -f /private/tmp/stock-Dockerfile.project.expected \
  -t stock-python-back:wheel-preflight \
  /Users/conchi/workforce/stock_workforce
```

Expected: `Successfully tagged stock-python-back:wheel-preflight` or BuildKit's equivalent successful export message, with no APT step.

- [ ] **Step 2: Import representative compiled dependencies from the image**

Run:

```bash
docker run --rm --network vibedeploy-shared stock-python-back:wheel-preflight \
  python -c 'import clickhouse_connect, cryptography, lz4, numpy, pandas, uvloop, zstandard; print("wheel imports ok")'
```

Expected: `wheel imports ok` and exit status 0.

### Task 3: Update the persistent template with optimistic locking

**Files:**
- Read: `/private/tmp/stock-Dockerfile.project.before`
- Read: `/private/tmp/stock-Dockerfile.project.expected`
- Modify: PostgreSQL `easy_deploy.tb_project_script`, ID 541

- [ ] **Step 1: Recheck the old content hash immediately before mutation**

Run:

```bash
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select md5(content) from tb_project_script where id=541 and deleted_at is null"
```

Expected: `437f104d1ef760b9f277a978b785c7b4`. Stop without modifying anything if it differs.

- [ ] **Step 2: Update exactly one row using the guarded old hash**

Run in one shell so the base64 value is not printed:

```bash
NEW_CONTENT_B64=$(base64 -i /private/tmp/stock-Dockerfile.project.expected | tr -d '\n')
docker exec postgres16 psql -U conchi -d easy_deploy \
  -v new_content_b64="$NEW_CONTENT_B64" \
  -c "update tb_project_script
      set content=convert_from(decode(:'new_content_b64','base64'),'UTF8'), updated_at=now()
      where id=541
        and deleted_at is null
        and md5(content)='437f104d1ef760b9f277a978b785c7b4';"
```

Expected: `UPDATE 1`. Any other row count is a failure and deployment must not begin.

- [ ] **Step 3: Run the post-change persistent contract**

Run:

```bash
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select content from tb_project_script where id=541 and deleted_at is null" \
  | rg -n -- '--only-binary=:all:|--retries 8|--timeout 60'
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select content from tb_project_script where id=541 and deleted_at is null" \
  | rg -n 'apt-get|build-essential|(^|[^[:alnum:]_])gcc([^[:alnum:]_]|$)|apk add'
```

Expected: the first command reports all reliability flags; the second command reports no matches and exits 1.

- [ ] **Step 4: Retain the exact rollback command**

If Task 4 proves the wheel-only change itself is invalid, run:

```bash
CURRENT_HASH=$(docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select md5(content) from tb_project_script where id=541 and deleted_at is null")
OLD_CONTENT_B64=$(base64 -i /private/tmp/stock-Dockerfile.project.before | tr -d '\n')
docker exec postgres16 psql -U conchi -d easy_deploy \
  -v old_content_b64="$OLD_CONTENT_B64" \
  -v current_hash="$CURRENT_HASH" \
  -c "update tb_project_script
      set content=convert_from(decode(:'old_content_b64','base64'),'UTF8'), updated_at=now()
      where id=541 and deleted_at is null and md5(content)=:'current_hash';"
```

Expected when used: `UPDATE 1`, followed by MD5 `437f104d1ef760b9f277a978b785c7b4`.

### Task 4: Run and verify the complete stock deployment

**Files:**
- Materialize: `/Users/conchi/workforce/stock_workforce/deploy/**`
- Inspect: Docker containers and VibeDeploy deployment output

- [ ] **Step 1: Trigger the real aggregate route through VibeDeploy**

Run in one shell so the JWT is never printed:

```bash
TOKEN=$(curl -fsS http://127.0.0.1:23638/base/login \
  -H 'Content-Type: application/json' \
  --data '{"username":"admin","password":"123456"}' \
  | jq -er '.data.token')
curl -N --fail-with-body --max-time 1800 \
  "http://127.0.0.1:23638/project/deployStream/88?env=159&token=${TOKEN}"
```

Expected: the stream ends with `event: done` and `部署完成`, and contains no `event: error`.

- [ ] **Step 2: Verify all four expected containers and ports**

Run:

```bash
docker ps --filter name='^/stock-python-back$' \
  --filter name='^/stock-schedule-dashboard$' \
  --filter name='^/stock-vue-front-web$' \
  --filter name='^/stock-schedule-dashboard-web$' \
  --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

Expected running containers and host ports:

- `stock-python-back` — `10021`
- `stock-schedule-dashboard` — `10022`
- `stock-vue-front-web` — `6021`
- `stock-schedule-dashboard-web` — `6022`

- [ ] **Step 3: Verify restart policy and network isolation**

Run:

```bash
docker inspect \
  stock-python-back stock-schedule-dashboard stock-vue-front-web stock-schedule-dashboard-web \
  --format '{{.Name}} restart={{.HostConfig.RestartPolicy.Name}} networks={{range $key, $value := .NetworkSettings.Networks}}{{$key}} {{end}}'
```

Expected: every container has `restart=no` and the only listed network is `vibedeploy-shared`.

- [ ] **Step 4: Exercise backend health and frontend HTTP endpoints**

Run:

```bash
curl -fsS http://127.0.0.1:10021/health
curl -fsS http://127.0.0.1:10022/health
curl -fsSI http://127.0.0.1:6021/
curl -fsSI http://127.0.0.1:6022/
```

Expected: both health endpoints return successful JSON; both frontend roots return HTTP 200.

- [ ] **Step 5: Inspect recent container logs for startup failures**

Run:

```bash
docker logs --tail 120 stock-python-back
docker logs --tail 120 stock-schedule-dashboard
docker logs --tail 80 stock-vue-front-web
docker logs --tail 80 stock-schedule-dashboard-web
```

Expected: no crash loop, missing dependency, bind conflict, or fatal database/cache connection error.

- [ ] **Step 6: Record the final persistent template hash and deployment evidence**

Run:

```bash
docker exec postgres16 psql -U conchi -d easy_deploy -Atqc \
  "select id, md5(content), length(content) from tb_project_script where id=541 and deleted_at is null"
```

Expected: one row for ID 541 with a new MD5 and shorter content than the original 454-byte compiler-based template.
