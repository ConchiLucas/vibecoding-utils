# Remove AI Assistant Implementation Plan

> **For Codex:** REQUIRED SUB-SKILL: Use executing-plans to implement this plan task-by-task.

**Goal:** Remove the AI chat UI, chat/history/provider discovery endpoints, and AI deployment-configuration generation code while preserving AI provider configuration and table-sample AI data generation.

**Architecture:** Split the reusable non-stream AI completion client from the assistant-specific chat/tool service. Keep only three `/ai/config` routes and route table-data generation through the new completion client. Remove assistant UI/assets/routes/services/tools, and run an idempotent startup cleanup for the retired chat-history table.

**Tech Stack:** Go, Gin, GORM, PostgreSQL/SQLite tests, React, TypeScript, Vite, Node test runner.

---

## Task 1: Add backend preservation contract tests

**Files:**
- Create: `server/service/system/sys_ai_completion_test.go`
- Create: `server/service/system/sys_ai_config_test.go`

1. Add `TestAICompletionServiceCompleteJSONOnceUsesConfiguredProvider` using `httptest.Server`. The test must set `global.GVA_DB = nil`, install one temporary OpenAI-compatible provider in `global.GVA_CONFIG.AI`, call `AICompletionService.CompleteJSONOnce`, and assert:
   - request path is `/v1/chat/completions`;
   - request contains the supplied `AIMessage`;
   - `response_format.type` is `json_object`;
   - returned content and provider name match the server response/configuration.
   Restore all globals with `t.Cleanup`.
2. Add `TestAIConfigServicePersistsAndSwitchesActiveProvider` using an isolated in-memory SQLite database migrated with `system.TbAIConfig`. Save a two-provider configuration, reload it, switch the active provider, and assert both providers remain unchanged and the selected provider changes.
3. Run:

   ```bash
   cd server
   go test ./service/system -run 'TestAI(Completion|Config)Service' -count=1
   ```

   Expected RED result: compile errors for the not-yet-created `AICompletionService`, `AIMessage`, and `AIConfigService` symbols.

4. Commit the red tests:

   ```bash
   git add server/service/system/sys_ai_completion_test.go server/service/system/sys_ai_config_test.go
   git commit -m "test: define retained AI service contracts"
   ```

## Task 2: Extract reusable AI configuration and completion services

**Files:**
- Create: `server/service/system/sys_ai_completion.go`
- Modify: `server/service/system/sys_ai_config.go`
- Modify: `server/service/system/sys_ai_chat.go`
- Modify: `server/service/system/tb_connection.go`
- Modify: `server/service/system/enter.go`
- Modify: `server/api/v1/system/sys_ai_chat.go`

1. In `sys_ai_completion.go`, define `AIMessage` with only `Role` and `Content`, plus `AICompletionService`. Move/copy only the non-stream provider client behavior required by table-data generation:
   - provider endpoint normalization;
   - `CompleteOnce` and `CompleteJSONOnce`;
   - OpenAI-compatible and Anthropic request/response handling;
   - max-token normalization, system-message splitting, and HTTP request execution.
   Do not include tool definitions, SSE, chat IDs, deployment prompts, or history behavior.
2. Rename the receiver in `sys_ai_config.go` from `AIChatService` to `AIConfigService` for `CurrentAIConfig`, `SaveAIConfig`, `SaveActiveAIConfig`, and DB-loading helpers.
3. While the old assistant service still exists, change its configuration lookups to `(&AIConfigService{}).CurrentAIConfig()` so the package keeps compiling during extraction.
4. Replace `AIChatService`/`ChatMessage` usages in `tb_connection.go` with `AICompletionService`/`AIMessage`, including initial generation and JSON-repair retries.
5. Add `AIConfigService` and `AICompletionService` to the service group. Update the existing API file temporarily to call `AIConfigService` for the three config handlers.
6. Run the focused tests and the existing system-service suite:

   ```bash
   cd server
   go test ./service/system -run 'TestAI(Completion|Config)Service' -count=1
   go test ./service/system -count=1
   ```

   Expected GREEN result: both commands pass.
7. Commit:

   ```bash
   git add server/service/system server/api/v1/system/sys_ai_chat.go
   git commit -m "refactor: extract reusable AI completion service"
   ```

## Task 3: Add and test retired-table cleanup

**Files:**
- Create: `server/initialize/removed_ai_assistant_tables_test.go`
- Create: `server/initialize/removed_ai_assistant_tables.go`
- Modify: `server/initialize/gorm.go`
- Modify: `server/initialize/ensure_tables.go`

1. Write `TestDropRemovedAIAssistantTablesDropsHistoryOnly` using SQLite. Create dummy `tb_ai_chat_history` and `tb_ai_config` tables, insert one AI-config row, invoke `dropRemovedAIAssistantTables`, then assert history is absent and AI config plus its row remain.
2. Run:

   ```bash
   cd server
   go test ./initialize -run TestDropRemovedAIAssistantTablesDropsHistoryOnly -count=1
   ```

   Expected RED result: `dropRemovedAIAssistantTables` is undefined.
3. Implement:

   ```go
   func dropRemovedAIAssistantTables(db *gorm.DB) error {
       return db.Exec("DROP TABLE IF EXISTS tb_ai_chat_history").Error
   }
   ```

4. Call this cleanup from both `RegisterTables` and the ensure-tables migration path, returning/logging an initialization error rather than silently ignoring failure. Remove `TbAIChatHistory` from both migration and table-existence lists. Keep `TbAIConfig` registration unchanged.
5. Re-run:

   ```bash
   cd server
   go test ./initialize -run TestDropRemovedAIAssistantTablesDropsHistoryOnly -count=1
   ```

   Expected GREEN result: pass.
6. Commit:

   ```bash
   git add server/initialize
   git commit -m "chore: retire AI chat history table"
   ```

## Task 4: Remove backend assistant routes, history, and deployment tools

**Files:**
- Create: `server/router/system/sys_ai_config.go`
- Create: `server/api/v1/system/sys_ai_config.go`
- Create: `server/router/system/sys_ai_config_test.go`
- Modify: `server/initialize/router.go`
- Modify: `server/router/system/enter.go`
- Modify: `server/api/v1/system/enter.go`
- Modify: `server/service/system/enter.go`
- Delete: `server/router/system/sys_ai_chat.go`
- Delete: `server/api/v1/system/sys_ai_chat.go`
- Delete: `server/api/v1/system/sys_ai_chat_history.go`
- Delete: `server/service/system/sys_ai_chat.go`
- Delete: `server/service/system/sys_ai_chat_history.go`
- Delete: `server/service/system/sys_deploy_tools.go`
- Delete: `server/service/system/project_type_detector.go`
- Delete: `server/model/system/sys_ai_chat_history.go`

1. Add a router test that initializes `AIConfigRouter` on a Gin group and asserts its only routes are:
   - `GET /ai/config`
   - `POST /ai/config`
   - `POST /ai/config/active`
2. Run:

   ```bash
   cd server
   go test ./router/system -run TestAIConfigRouterRegistersOnlyConfigRoutes -count=1
   ```

   Expected RED result: `AIConfigRouter` is undefined.
3. Move only `GetAIConfig`, `SaveAIConfig`, and `SaveAIActiveProvider` into `sys_ai_config.go` and make them use `AIConfigService`. Create the matching config-only router.
4. Register only `InitAIConfigRouter` in `initialize/router.go`. Remove public provider/model registration and private chat/history registration.
5. Remove assistant/history members from API, router, and service groups. Retain `AIConfigApi`, `AIConfigRouter`, `AIConfigService`, and `AICompletionService`.
6. Delete all listed assistant/history/tool files. Do not delete normal project, route, script, deployment, AI-config, or table-generation services.
7. Run:

   ```bash
   cd server
   gofmt -w api/v1/system/sys_ai_config.go api/v1/system/enter.go router/system/sys_ai_config.go router/system/sys_ai_config_test.go router/system/enter.go service/system/enter.go initialize/router.go
   go test ./router/system ./api/v1/system ./initialize ./service/system -count=1
   ```

   Expected GREEN result: all selected packages compile and tests pass.
8. Verify removed backend surface:

   ```bash
   rg -n 'InitAIChat|AIChatService|AIChatHistory|generate_deploy_info|DetectProjectType|ScanProject|/ai/chat|/ai/providers|/ai/models' server
   ```

   Expected result: no matches, except explicit assertions in removal tests if any.
9. Commit:

   ```bash
   git add -A server
   git commit -m "refactor: remove AI assistant backend"
   ```

## Task 5: Add frontend removal/retention contract test

**Files:**
- Create: `web-react/tests/ai-assistant-removal.test.mjs`
- Modify: `web-react/package.json`

1. Add a Node test that reads the source tree and asserts:
   - `Header.tsx` does not contain `toggle-ai-chat`;
   - `Layout.tsx` does not contain `AIChatWidget`;
   - `src/components/AIChatWidget` and `src/api/aiChat.ts` do not exist;
   - `src/api/aiConfig.ts` exports `getAIConfig`, `saveAIConfig`, and `saveAIActiveProvider`;
   - `AIConfigManager.tsx` imports `../../api/aiConfig`;
   - `TableDataPreview.tsx` still contains `generateRemoteTableData` and `AI 造数`.
2. Add `"test": "node --test tests/*.test.mjs"` to `package.json`.
3. Run:

   ```bash
   cd web-react
   npm test
   ```

   Expected RED result: the assistant UI/API still exists and `aiConfig.ts` does not.
4. Commit the red contract test:

   ```bash
   git add web-react/tests/ai-assistant-removal.test.mjs web-react/package.json
   git commit -m "test: define frontend AI removal contract"
   ```

## Task 6: Remove frontend assistant UI and split retained config API

**Files:**
- Create: `web-react/src/api/aiConfig.ts`
- Modify: `web-react/src/views/layout/Header.tsx`
- Modify: `web-react/src/views/layout/Layout.tsx`
- Modify: `web-react/src/views/config-manager/AIConfigManager.tsx`
- Delete: `web-react/src/api/aiChat.ts`
- Delete: `web-react/src/components/AIChatWidget/AIChatWidget.tsx`
- Delete: `web-react/src/components/AIChatWidget/AIChatWidget.css`
- Delete: `web-react/src/components/AIChatWidget/AIChatWidgetHistory.ts`
- Delete: `web-react/src/components/AIChatWidget/AIChatWidgetIntent.ts`

1. Create `aiConfig.ts` by moving only `AIProviderConfigItem`, `AIConfigResponse`, `getAIConfig`, `saveAIConfig`, and `saveAIActiveProvider` from `aiChat.ts`. Keep endpoint paths and payload shapes unchanged.
2. Update `AIConfigManager.tsx` to import from `../../api/aiConfig`.
3. Remove the purple AI button, `Sparkles` import if unused, and `toggle-ai-chat` dispatch from `Header.tsx`.
4. Remove `AIChatWidget` import and rendering from `Layout.tsx`.
5. Delete the assistant component directory and old mixed `aiChat.ts` API module.
6. Run:

   ```bash
   cd web-react
   npm test
   npm run build
   ```

   Expected GREEN result: contract test and Vite production build pass.
7. Run the existing lint command only as a baseline comparison:

   ```bash
   npm run lint
   ```

   Expected known baseline result: ESLint fails in `eslint.config.js` because `reactHooks.configs.flat.recommended` is undefined. Do not broaden this change to repair unrelated lint configuration; record the unchanged baseline failure.
8. Commit:

   ```bash
   git add -A web-react
   git commit -m "refactor: remove AI assistant frontend"
   ```

## Task 7: Full static, build, and database verification

**Files:**
- Modify only if a verification failure exposes a defect in the scoped implementation.

1. Run backend verification:

   ```bash
   cd server
   go test ./service/system ./initialize ./router/system ./api/v1/system -count=1
   ```

2. Run frontend verification:

   ```bash
   cd web-react
   npm test
   npm run build
   ```

3. Run repository searches:

   ```bash
   rg -n 'AIChatWidget|toggle-ai-chat|AIChatService|AIChatHistory|generate_deploy_info|sys_deploy_tools|project_type_detector|/ai/chat|/ai/providers|/ai/models' web-react/src server
   rg -n '/ai/config|AIConfigManager|generateRemoteTableData|AI 造数|AICompletionService' web-react/src server
   ```

   The first search must have no production-code matches. The second must show retained config and table-generation paths.
4. Before restarting, snapshot the exact `tb_ai_config` rows and a deterministic hash. Restart the VibeDeploy backend/frontend from this worktree using the existing development script. Then confirm:
   - `GET /ai/config` returns 200 with an authenticated request;
   - `/ai/chat`, `/ai/providers`, `/ai/models`, and `/ai/chat/history*` return 404;
   - `tb_ai_chat_history` does not exist;
   - `tb_ai_config` contents and hash are unchanged;
   - project-list and project-deployment pages still load;
   - the top-right purple AI entry and chat panel are absent;
   - the table preview still shows `AI 造数`.
5. If live verification fails, stop and revert only the failing implementation commit(s), then diagnose before retrying. The chat-history table deletion is intentionally irreversible once applied to a database, as approved.
6. Run `git status --short` and review `git diff 761baf6...HEAD`. Confirm there are no unrelated files or generated build artifacts.

