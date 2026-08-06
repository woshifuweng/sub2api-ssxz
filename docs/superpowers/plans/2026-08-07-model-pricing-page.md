# 模型定价页重建 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将认证用户的 `/app/available-channels` 改造成按模型展示的 Anthropic/OpenAI 定价表，并以隔离资源方式发布到生产。

**Architecture:** 复用现有 `/api/v1/channels/available` 和 `/api/v1/groups/rates`，在前端将渠道平台下的模型扁平化为行；路由改为懒加载 `ModelPricingView.vue`，旧 `AvailableChannelsView.vue` 保留。生产构建只替换新路由 chunk 及其依赖，并在稳定入口中替换对应动态 import 映射。

**Tech Stack:** Vue 3、TypeScript、Vue Router、Vitest、Vite、Go `go:embed`、PowerShell、SSH。

---

### Task 1: 固化 API 数据契约和页面辅助逻辑

**Files:**
- Create: `frontend/src/views/user/ModelPricingView.vue`
- Modify: `frontend/src/api/channels.ts` only if the existing model type needs `context_length`
- Test: `frontend/src/views/user/__tests__/ModelPricingView.spec.ts`

- [ ] **Step 1: Add the normalized view-model types and pure helpers in the view module**

Use the existing `UserAvailableChannel`, `UserChannelPlatformSection`, `UserSupportedModel`, and `UserAvailableGroup` types. Add local types for a flattened row and implement pure functions with these signatures:

```ts
type PricingRow = {
  key: string
  model: UserSupportedModel
  platform: 'anthropic' | 'openai'
  groups: UserAvailableGroup[]
  defaultGroup: UserAvailableGroup | null
  inputPrice: number | null
  outputPrice: number | null
  cacheWritePrice: number | null
  cacheReadPrice: number | null
  contextLength: number | null
}

function classifyPlatform(model: UserSupportedModel, sectionPlatform: string): 'anthropic' | 'openai' | null
function flattenChannels(channels: UserAvailableChannel[]): PricingRow[]
function pricePerMillion(base: number | null, multiplier: number | undefined): number | null
function formatPrice(base: number | null, multiplier: number | undefined): string
```

`classifyPlatform` must prefer canonical API platform, then map `claude-` to Anthropic and `gpt-`, `o1`, `o3`, `deepseek-` to OpenAI. Unknown models return `null` and are not shown in either tab.

- [ ] **Step 2: Write failing tests for the data contract**

Cover: one channel with two platform sections becomes one row per model; Claude and GPT prefixes classify correctly; a group multiplier multiplies input/output prices; null prices format as `-`; duplicate model names from different platforms remain separate using `platform:model` as the key; rows without a group still render with a null default group.

- [ ] **Step 3: Run the focused test and confirm the new helpers fail before implementation**

Run from `frontend`:

```powershell
npm run test -- src/views/user/__tests__/ModelPricingView.spec.ts --run
```

Expected: FAIL because `ModelPricingView.vue` and its helpers do not exist yet.

- [ ] **Step 4: Implement the helpers and keep the API response unchanged**

Use `userChannelsAPI.getAvailable()` and `userGroupsAPI.getUserGroupRates()` in the page loader. Do not create `frontend/src/api/pricing.ts`, because the production probe found no `/api/v1/pricing` endpoint and the existing model objects already contain pricing fields.

- [ ] **Step 5: Run the focused test and commit the data layer**

Run the same focused Vitest command; expected PASS. Commit only the new page/test files with:

```powershell
git add frontend/src/views/user/ModelPricingView.vue frontend/src/views/user/__tests__/ModelPricingView.spec.ts
git commit -m "新增模型定价页数据整理"
```

### Task 2: Build the model pricing UI

**Files:**
- Modify: `frontend/src/views/user/ModelPricingView.vue`

- [ ] **Step 1: Add the page shell and controls**

Use `AppSectionShell`, `TablePageLayout`, `Icon`, `PlatformIcon`, `GroupBadge`, and `GroupOptionItem` where their props match. Render two buttons for `anthropic` and `openai`, a search input, a refresh button, and `显示 X 个共 X 个` count text. Keep all controls keyboard accessible and disable refresh while loading.

- [ ] **Step 2: Add the table and group popover**

Render columns `模型名 | 供应商 | 分组 | 输入 | 输出 | 缓存 | 上下文`. The supplier cell uses a colored badge plus `PlatformIcon`. The group cell shows the default `GroupBadge`; if `groups.length > 1`, show a button that opens a positioned list with every group name, multiplier, and description. Close the list on outside click and Escape.

- [ ] **Step 3: Add filtering, sorting, and empty/error states**

Search against model name and provider label. Sort by model name on header click, toggling ascending/descending. Keep the selected platform tab while searching. Show a loading row, an empty-state row, and route errors through `appStore.showError(extractApiErrorMessage(...))`.

- [ ] **Step 4: Run focused tests and type-check**

Run:

```powershell
npm run test -- src/views/user/__tests__/ModelPricingView.spec.ts --run
npm run type-check
```

Expected: focused tests PASS and type-check exits 0.

- [ ] **Step 5: Commit the UI**

```powershell
git add frontend/src/views/user/ModelPricingView.vue
git commit -m "重建按模型展示的定价表"
```

### Task 3: Switch the route and verify the frontend build

**Files:**
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: Replace only the available-channels lazy component**

Change the route component from:

```ts
component: () => import('@/views/user/AvailableChannelsView.vue')
```

to:

```ts
component: () => import('@/views/user/ModelPricingView.vue')
```

Do not delete or rename `AvailableChannelsView.vue`.

- [ ] **Step 2: Build and inspect the new asset map**

Run from `frontend`:

```powershell
npm run type-check
npm run build
```

Expected: both pass; the output contains a new `ModelPricingView-*.js` chunk and no import error for the preserved old view.

- [ ] **Step 3: Commit the route**

```powershell
git add frontend/src/router/index.ts
git commit -m "切换可用模型页到定价视图"
```

### Task 4: Isolated production composition and verification

**Files:**
- Create: `F:\CodexTemp\compose-model-pricing-isolated-20260807.ps1`
- Create: `F:\CodexTemp\deploy-model-pricing-isolated-20260807.sh`
- Modify: `F:\CodexTemp\codex-to-claude-handoff.md`

- [ ] **Step 1: Compose an isolated dist from the known production snapshot**

Copy the production frontend snapshot to a new temporary output, copy only the local `ModelPricingView` JS/CSS and direct changed dependencies, and patch the stable entry's old AvailableChannels route strings to the new chunk names. Validate every relative import in the new chunk exists in the composed asset directory.

- [ ] **Step 2: Build an embedded Linux binary**

From `backend` run:

```powershell
$env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -tags embed -trimpath -ldflags '-s -w' -o sub2api-model-pricing-linux ./cmd/server
```

Verify the first four bytes are `7F 45 4C 46` and record SHA256. Do not deploy a non-ELF build.

- [ ] **Step 3: Deploy with a pre-deploy backup and health gate**

Upload to `/opt/sub2api/.incoming/`, back up `/opt/sub2api/sub2api`, atomically replace it, restart `sub2api`, and require: systemd active, `/health` 200, `/v1/models` without a key 401, and SHA256 match. Restore the backup automatically if any gate fails.

- [ ] **Step 4: Verify production resources and behavior**

Check the public entry contains the new route query/version and that ModelPricing JS/CSS plus changed dependencies return HTTP 200. In an authenticated browser verify Anthropic/OpenAI tabs, filtering, icons, group badges, multipliers, and search. If no authenticated browser is available, record that limitation instead of claiming visual acceptance.

- [ ] **Step 5: Update handoff and commit the deployment record**

Append Part 1 SQL/channel results, API probe results, chosen pricing-field strategy, build/deploy SHA, verification results, and any incomplete item to `F:\CodexTemp\codex-to-claude-handoff.md`.
