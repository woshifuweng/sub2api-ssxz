# Usage Density and Admin Header Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将客户使用记录从 13 个平铺列重组为 8 个双层组合列，并修正管理后台顶部标题、版本和分隔线比例，同时保留全部信息和现有业务行为。

**Architecture:** 在共享 `UsageTable` 增加仅由客户页开启的 `groupedDetails` 展示模式，让现有模型、IP、分组、Token、费用、耗时和定位码逻辑继续作为唯一渲染来源；客户页只调整列定义和宽度。管理页顶部仅修改共享 `AppHeader` 的结构类与确定性尺寸，不触碰路由、数据或版本查询。

**Tech Stack:** Vue 3、TypeScript、Tailwind CSS、Vitest、Vite、Go 单二进制发布、Docker Compose。

---

## File Map

- Modify: `frontend/src/components/admin/usage/UsageTable.vue` — 增加可选组合详情模式，并复用现有单元格逻辑。
- Modify: `frontend/src/views/user/AppUsageView.vue` — 启用组合模式、改为 8 列并更新客户页列宽。
- Modify: `frontend/src/components/admin/usage/__tests__/UsageTable.spec.ts` — 验证组合字段完整、定位码复制和 IP 工具栏不丢。
- Modify: `frontend/src/views/user/__tests__/AppUsageView.spec.ts` — 验证 8 列、组合模式和横向容器边界。
- Modify: `frontend/src/components/layout/AppHeader.vue` — 统一顶部标题块、版本标识和分隔线基线。
- Modify: `frontend/src/components/layout/__tests__/AppSidebar.spec.ts` — 添加顶部比例和对齐源代码门禁。
- Modify: `docs/AI协作/PROJECT_HANDOFF.md` — 记录最终版本、提交、验证和生产状态。

### Task 1: 锁定组合列行为

- [ ] **Step 1: 在 `UsageTable.spec.ts` 写失败测试**

添加一条包含模型、推理强度、接口、IP、分组、类型、计费方式、时间和定位码的数据，挂载时传入：

```ts
props: {
  data: [groupedUsageRow],
  columns: [
    { key: 'api_key', label: 'API Key' },
    { key: 'model', label: 'Model / Reasoning' },
    { key: 'endpoint', label: 'Endpoint / IP' },
    { key: 'group', label: 'Group / Billing' },
    { key: 'tokens', label: 'Token' },
    { key: 'cost', label: 'Cost' },
    { key: 'latency', label: 'Latency' },
    { key: 'created_at', label: 'Time / Support code' }
  ],
  groupedDetails: true
}
```

断言页面同时包含 `high`、`/v1/responses`、测试 IP、分组名、流式、按量、定位码，并且 IP 批量获取工具栏仍出现。

- [ ] **Step 2: 运行测试并确认失败**

Run: `pnpm vitest run src/components/admin/usage/__tests__/UsageTable.spec.ts`

Expected: FAIL，原因是 `groupedDetails` 尚未渲染推理、IP、类型、计费和定位码。

- [ ] **Step 3: 在 `UsageTable.vue` 增加最小组合模式**

Props 增加：

```ts
groupedDetails?: boolean
```

默认值：

```ts
groupedDetails: false
```

在现有单元格中按 `groupedDetails` 增加第二层，不复制格式化函数：

```vue
<div v-if="groupedDetails" class="usage-cell-detail">
  {{ formatReasoningEffort(row.reasoning_effort) }}
</div>
```

`endpoint` 第二层复用 `row.ip_address` 与 `IpGeoCell`；`group` 第二层复用 `getRequestTypeLabel`、`getRequestTypeBadgeClass`、`getBillingModeLabel` 和 `getBillingModeBadgeClass`；`created_at` 第二层复用现有定位码截断、标题和 `copyRequestId`。`showIpGeoToolbar` 改为在组合模式且数据含 IP 时也启用：

```ts
const showIpGeoToolbar = computed(() =>
  props.columns.some((col) => col.key === 'ip_address') ||
  (props.groupedDetails && props.data.some((row) => Boolean(row.ip_address)))
)
```

- [ ] **Step 4: 运行组件测试并确认通过**

Run: `pnpm vitest run src/components/admin/usage/__tests__/UsageTable.spec.ts`

Expected: PASS，既有管理端单列模式测试保持通过。

- [ ] **Step 5: 提交共享表格改动**

```bash
git add frontend/src/components/admin/usage/UsageTable.vue frontend/src/components/admin/usage/__tests__/UsageTable.spec.ts
git commit -m "feat(usage): add grouped detail cells"
```

### Task 2: 客户页切换为 8 个组合列

- [ ] **Step 1: 在 `AppUsageView.spec.ts` 写失败测试**

把 13 列断言改为精确 8 列：

```ts
expect(usageColumnKeys).toEqual([
  'api_key', 'model', 'endpoint', 'group',
  'tokens', 'cost', 'latency', 'created_at'
])
expect(source).toContain(':grouped-details="true"')
expect(source).toContain('min-width: 86.5rem')
```

同时断言源码不再定义独立 `reasoning_effort`、`ip_address`、`stream`、`billing_mode`、`request_id` 列。

- [ ] **Step 2: 运行测试并确认失败**

Run: `pnpm vitest run src/views/user/__tests__/AppUsageView.spec.ts`

Expected: FAIL，当前仍为 13 列且最小宽度为 `105.5rem`。

- [ ] **Step 3: 修改 `AppUsageView.vue` 列定义与样式**

启用：

```vue
:grouped-details="true"
```

列定义改为 8 列，组合标题使用现有翻译拼接：

```ts
const usageTableColumns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), class: 'usage-col-api-key' },
  { key: 'model', label: `${t('usage.model')} / ${t('usage.reasoningEffort')}`, sortable: true, class: 'usage-col-model-context' },
  { key: 'endpoint', label: `${t('usage.endpoint')} / ${t('admin.usage.ipAddress')}`, class: 'usage-col-route' },
  { key: 'group', label: `${t('admin.usage.group')} / ${t('admin.usage.billingMode')}`, class: 'usage-col-group-context' },
  { key: 'tokens', label: t('usage.tokens'), class: 'usage-col-tokens' },
  { key: 'cost', label: t('usage.cost'), class: 'usage-col-cost' },
  { key: 'latency', label: t('usage.duration'), class: 'usage-col-latency' },
  { key: 'created_at', label: `${t('usage.time')} / ${t('usage.workbench.supportCode')}`, sortable: true, class: 'usage-col-activity' }
])
```

表格最小宽度改为 `86.5rem`；列宽分别为 `10rem / 8.5rem / 16rem / 14rem / 10.5rem / 7.5rem / 8.5rem / 11.5rem`。组合单元格使用 0.25–0.375rem 层间距，行内主信息不缩小，次信息使用 11–12px 和 muted 色。

- [ ] **Step 4: 运行客户页测试**

Run: `pnpm vitest run src/views/user/__tests__/AppUsageView.spec.ts src/components/admin/usage/__tests__/UsageTable.spec.ts`

Expected: PASS。

- [ ] **Step 5: 提交客户页改动**

```bash
git add frontend/src/views/user/AppUsageView.vue frontend/src/views/user/__tests__/AppUsageView.spec.ts
git commit -m "fix(usage): reduce table density without hiding details"
```

### Task 3: 修正管理后台顶部比例

- [ ] **Step 1: 在 `AppSidebar.spec.ts` 写失败测试**

断言 `AppHeader.vue` 使用明确结构类与尺寸：

```ts
expect(headerSource).toContain('class="app-header-title-cluster"')
expect(headerSource).toContain('class="app-header-title-copy"')
expect(headerSource).toContain('class="app-header-version"')
expect(headerSource).toContain('height: var(--ssxz-header-height, 56px);')
expect(headerSource).toContain('align-items: center;')
expect(headerSource).not.toContain('class="mt-0.5"')
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `pnpm vitest run src/components/layout/__tests__/AppSidebar.spec.ts`

Expected: FAIL，当前标题与版本仍依赖 Tailwind `items-start` 和 `mt-0.5`。

- [ ] **Step 3: 修改 `AppHeader.vue`**

把标题区域改为具名结构类：

```vue
<div class="app-header-title-cluster">
  <div class="app-header-title-copy hidden lg:block">
    <h1>{{ pageTitle }}</h1>
    <p v-if="pageDescription">{{ pageDescription }}</p>
  </div>
  <VersionBadge
    v-if="authStore.isAdmin"
    :runtime-actions-enabled="false"
    class="app-header-version"
  />
</div>
```

使用确定性 CSS：

```css
.app-header-shell,
.app-header-inner { height: var(--ssxz-header-height, 56px); }
.app-header-title-cluster { display: flex; align-items: center; gap: 12px; min-width: 0; }
.app-header-title-copy { min-width: 0; }
.app-header-title-copy h1 { margin: 0; font-size: 18px; font-weight: 600; line-height: 20px; }
.app-header-title-copy p { margin: 2px 0 0; font-size: 12px; line-height: 16px; }
.app-header-version { flex: none; align-self: center; }
```

保留现有颜色 token 和顶部 `border-b`，使主栏底线继续与同一 `--ssxz-header-height` 的侧栏头部对齐。

- [ ] **Step 4: 运行布局测试并确认通过**

Run: `pnpm vitest run src/components/layout/__tests__/AppSidebar.spec.ts src/components/layout/__tests__/docUrlSanitization.spec.ts`

Expected: PASS。

- [ ] **Step 5: 提交管理页顶部改动**

```bash
git add frontend/src/components/layout/AppHeader.vue frontend/src/components/layout/__tests__/AppSidebar.spec.ts
git commit -m "fix(admin): align header proportions and divider"
```

### Task 4: 全量验证与回归检查

- [ ] **Step 1: 运行相关测试**

Run: `pnpm vitest run src/components/admin/usage/__tests__/UsageTable.spec.ts src/views/user/__tests__/AppUsageView.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts src/components/common/__tests__/DataTable.spec.ts`

Expected: 全部 PASS。

- [ ] **Step 2: 运行类型检查与 lint**

Run: `pnpm typecheck && pnpm lint`

Expected: exit 0，无新增警告或错误。

- [ ] **Step 3: 运行完整前端测试与构建**

Run: `pnpm vitest run && pnpm build`

Expected: 完整测试 PASS，Vite production build 成功。

- [ ] **Step 4: 检查改动范围**

Run: `git diff --check HEAD~3..HEAD && git status --short`

Expected: 无空白错误；只有历史候选二进制保持未跟踪，不纳入提交。

### Task 5: 候选发布、正式部署与真人验收

- [ ] **Step 1: 生成 `.9` 候选二进制**

Run:

```powershell
Set-Location frontend
pnpm build
Set-Location ..\backend
$env:GOOS='linux'
$env:GOARCH='amd64'
$env:CGO_ENABLED='0'
go build -tags embed -trimpath -ldflags='-s -w -X main.Version=0.1.183-ssxz.20260830.9' -o ..\sub2api-linux-amd64-v183.9 .\cmd\server
Get-FileHash ..\sub2api-linux-amd64-v183.9 -Algorithm SHA256
Get-FileHash ..\sub2api-linux-amd64-v183.9 -Algorithm MD5
(Get-Item ..\sub2api-linux-amd64-v183.9).Length
```

Expected: Linux ELF 二进制生成成功，三个结果记录到发布汇报。

- [ ] **Step 2: 创建测试与正式环境共同回滚点**

先上传候选：

```powershell
scp .\sub2api-linux-amd64-v183.9 ssxz-server:/opt/sub2api/.incoming/sub2api-v183.9-linux-amd64
```

再在服务器建立 `/opt/sub2api/backups/v183r9-before-<时间>`，复制 `sub2api`、`sub2api-staging`、`config.yaml`、systemd 单元与 drop-in，执行 `sudo -u postgres pg_dump -Fc sub2api > database.dump`，生成并校验 `SHA256SUMS`。校验候选 SHA256 与本地一致；任一备份或校验失败则停止发布。

- [ ] **Step 3: 发布测试环境并验收**

使用原子替换：先 `install -o sub2api -g sub2api -m 0755` 到 `sub2api-staging.new`，再 `mv` 为 `/opt/sub2api/sub2api-staging` 并 `systemctl restart sub2api-staging`。轮询 `http://127.0.0.1:18080/health`，并检查 `/home`、`/login`、`/app/usage`、`/admin/dashboard` 为 200、无凭据 `/v1/models` 为 401、运行二进制 SHA256 与候选一致。任一门禁失败，从快照恢复 `sub2api-staging` 并重启。

- [ ] **Step 4: 发布正式环境并验收**

先确认测试环境运行二进制 SHA256 与候选一致且健康，再以相同 `install` + `mv` 原子替换 `/opt/sub2api/sub2api`，执行 `systemctl restart sub2api`。轮询 `http://127.0.0.1:8080/health`，重复路由门禁，确认 `systemctl is-active sub2api`、`NRestarts=0`、配置文件与快照一致，并检查 `journalctl -u sub2api --since '-10 min'` 无持续 error/panic。任一门禁失败，从快照恢复生产二进制并重启。

- [ ] **Step 5: 浏览器尺寸验收**

在 1920、1440、1280、1024 和 390px 下实际点击并测量：页面 `scrollWidth <= clientWidth`；表格滚动只发生在内部容器；8 个标题存在；组合字段完整；管理顶部标题、版本和侧栏分隔线同高。

- [ ] **Step 6: 更新交接文档并提交**

在 `docs/AI协作/PROJECT_HANDOFF.md` 写入 `.9` 版本、代码提交、哈希、备份目录、测试数量和线上验收结论；提交并推送代码与文档分支。
