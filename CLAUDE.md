# SSXZ 项目 — 每个会话开工前必读

这个文件会被自动读进每个新会话。它只写"会让你判断错的事实"，不写别的。

## 1. 真源码不在 `backend/`

主目录 `backend/` 是空壳（0 个 `.go` 文件）。往那里 grep 会**静默返回空**，读起来像"这功能不存在"——这是本项目最容易上的当。

```
真源码：.codex-work/fix-client-brand-announcements2/
```

## 2. "生产在跑什么"去读 `DEPLOYED.md`

不要靠 grep 源码树推断。仓库里同时存在**多条平行代码线**（差异达 4720 个文件），每条都是"真的代码"，但只有一条是活的。`DEPLOYED.md` 是唯一仲裁者。

一条命令直接问生产，不需要凭证：

```bash
curl -s https://api.ssxzapi.com/api/v1/settings/public | grep -o '"version":"[^"]*"'
```

## 3. 三个会骗你的信号

| 信号 | 为什么不可信 |
|---|---|
| `VERSION` 文件 | 写的是**我们 fork 自己的**发布计数器（0.1.x），跟上游 Wei-Shaw 的计数器（0.1.17x）是两套编号撞在同一个字段里。`0.1.3` 不代表"落后 168 个版本"。 |
| `git merge-base` / `--is-ancestor` | 本仓库大量使用 squash-merge，**祖先关系已被销毁**。"不是祖先"≠"没合过这份代码"。 |
| `accounts.updated_at` | 运行时会写这一行，它不等于"配置改动时间"。 |

## 4. 判断代码血脉的正确方法：文件指纹

tag 祖先关系不可信，但**文件内容不会被 squash 抹掉**。要判断某条线有没有包含上游某版本：

```bash
# 列出上游两个 tag 之间新增的文件，再看它们在目标分支里是否存在
git diff --name-status v0.1.136 v0.1.165 | awk '$1=="A"{print $2}'
git cat-file -e <branch>:<那个文件> && echo 有 || echo 无
```

全部"无" = 那条线不含该版本。这个方法**能区分**「165 底座」和「更早底座」，而 tag 测试不能。

**更强的一招：血脉峰值法**（按 `(路径,blob)` 完全一致计数，squash 抹不掉 blob）：

```bash
# 对每个上游 tag，数生产线的 Go 文件有多少在该 tag 里字节级一致，取峰值
git ls-tree -r <tag> -- backend | ...  # 与生产线 ls-files 求交集
```

**⚠️ 已测出的结论（2026-08-07，别再重推）：生产线不是 165 底座，也不是任何上游 tag 的底座。**

| 测法 | 结果 |
|---|---|
| 136→150 新增 259 个文件，生产线里有 | **1** |
| 150→165 新增 340 个，生产线里有 | **0** |
| 165→169 / 169→171 新增，生产线里有 | **0 / 0** |
| 血脉峰值（真上游 tag 里最高的） | v0.1.136 **38.2%**，之后单调下降到 171 的 33.1% |
| `merge-base` 对 136…171 每个 tag | **全部同一个提交** `bda7c39e5`（2026-03-21）|

峰值在最老的 tag 上、且单调下降，说明**分家点比所有本地可见上游 tag 都早**，不存在"落后 6 版"。

**陷阱：`v0.1.3` 会伪装成 78.2% 的峰值——那是我们自己的 tag**（2026-06-11，woshifuweng，
距生产线 HEAD 633 个提交），拿它比就是自己跟自己比。**只用 `v0.1.1xx` 这种上游编号测。**

⚠️ **别把本节结论外推成"我们从没跟过上游"。** 本节测的只是**生产线 P**。
另一条线 `upgrade/v0.1.169` 真的合过 v0.1.171，且包含那 279 个上游文件——见第 6 节。
把 P 的测量结果说成"整个项目已脱离上游"是 2026-08-07 犯过的错。

## 5. 生产路由探测（判断线上到底部署了哪条线）

未鉴权 GET 一个 admin 路由：`401` = 路由存在，`404` = 路由不存在。用一条两条线都有的路由当控制组：

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://api.ssxzapi.com/api/v1/admin/users   # 控制组，应为 401
```

## 6. 长期方向：Upstream-first（上游当主干 + SSXZ 补丁层）— 2026-08-07 用户已确认

⚠️ **"分家"这个词是错的，已作废。** 上游 remote 从没断开（`upstream` →
`github.com/Wei-Shaw/sub2api.git`，仍可 fetch）。2026-08-05 那次合 v0.1.171
（111 提交 / 138 冲突）**是真的合过**，合并提交 `b9545e8dd` 就在 `upgrade/v0.1.169` 上。
问题不是"从没跟上游"，而是**部署是从另一条不含 171 的线出去的**。

**已定方向**：以受信上游 release 为主干，SSXZ 定制作为**边界清晰、可独立验证、可重复应用**
的补丁层叠加。目标不是零维护（只要还有品牌/计费/调度定制就不可能），而是把维护对象
从"整套分叉"缩小为"一组明确的 SSXZ 补丁"。

三条线的角色（**别混用**）：

| 线 | 角色 |
|---|---|
| **P** = `e8ef9e645`（`.codex-work/fix-client-brand-announcements2/`）| 当前生产线。迁移期**冻结为可回滚基线**，继续对客户服务 |
| **U** = `upgrade/v0.1.169`（`9e9440e35`）| **不完整合并产物，只作取证/对照，不再修补、不作未来主干** |
| **U2** = 从 `v0.1.171` = `afd154b92` 新建的干净树 | 未来主干基础。**已验证自洽**（见下）|

下面 §6 余下各节是这个方向的**证据底座**，不是相反结论——尤其"P 自洽 / U 不自洽"
是**关闭修补 U 这条路**的依据，不是否定 upstream-first。

### 两条线的实际家底（2026-08-07 实测）

| | 生产线 P (`e8ef9e645`) | 升级线 U (`upgrade/v0.1.169` = `9e9440e35`) |
|---|---|---|
| 含 v0.1.171 / `b9545e8dd` | **否 / 否** | **是 / 是** |
| 上游新增生产 Go 文件（279 个去重） | **有 0 个** | **有 279 个** |
| Reseller 经销商文件 | **0 个** | **31 个** |
| `channel_id=NULL` 计费修复 | 无 | 有（就是 U 的 tip）|
| migration 最大号 / 文件数 | 138 / 118 | 204 / 263 |
| 今天那 5 个修复 | 有 | **全无**（Turnstile 仍在闸上、无 AppSectionShell、无 parseVersion 修复）|
| `dist` 前端产物 | 已跟踪，入口 `index-Bn3BFivZ.js`，`?v=` 0 处 | **未跟踪，不可复现** |

**关键修正：U 已经包含那 279 个"我们完全没有的"上游文件，138 个冲突在 U 上已经解完了。**
所以"整版合上游 = 搬进一个陌生产品"这个论证**只对 P 成立，对 U 不成立**。
以前把它当成"永不跟上游"的理由是推错了对象。

### ⭐ 决定性事实：生产库已经是 U 的库（2026-08-07 实测，Codex 只读 SELECT）

```
total_rows=276  has_134=1  has_138=1  gt_138_rows=86
reseller: 200_reseller_roles.sql, 201_reseller_fields_hardening.sql
GT138_SET_DIFF=EMPTY   ← U 本地 >138 的 86 个文件名，生产库全都已记录
```

**风险方向跟直觉相反：不是"切到 U 有风险"，是"P 在跑一个比它自己新 86 个 migration 的库"。**

| 推论 | 依据 |
|---|---|
| 切 U 会跑几个 migration | **0 个**（文件名全部已在 `schema_migrations` 里，runner 按 filename 跳过）|
| 那 12 个换号 DDL 会不会重跑 | **不会**。P 号和 U 号**两套都已在库里**，等于重跑早就发生过且无害 |
| P 才是异常的那条 | P 只有 118 个 migration 文件，库里有 276 行 |
| Reseller 表和数据 | **已在生产库**（200/201 有留痕），但 P 有 **0 个** reseller 文件 → 那 8 条路由现在是 404 |

runner 主键是带号前缀的完整文件名（`schema_migrations.filename TEXT PRIMARY KEY`），
库里多出来的行 runner 不认识也不管，所以 P 能正常跑——**但 P 的代码没有那些表对应的功能**。

Codex 扫出的 2 条裸 `ADD COLUMN`（`189_add_group_allow_live`、`204_add_billing_model_to_usage_logs`）
和 9 处无 `ON CONFLICT` 的 INSERT：**正常部署都不会执行**（文件名已记录 → 跳过；
9 处里 8 处在 `CREATE OR REPLACE FUNCTION` 函数体内，定义时不执行）。只有**强制重跑**才会炸。

### ⛔ U 前端根本构建不出来：缺 108 个上游文件却留着 import（2026-08-07 实测）

U 后端交叉编译**成功**（162,165,705 字节，`GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 已确认），
`go test ./...` **全绿**。但前端是硬阻断：

```
修掉那 2 个语法错之后，vue-tsc 仍报 625 错 / 40 文件
pnpm exec vitest run → 60 个测试文件失败 / 133 个测试失败（184 文件 877 测试）
```

**"U 前端只坏了 2 个文件"是错的（我说错了）。真因是缺文件，不是语法：**

| 测项 | 结果 |
|---|---|
| 上游 v0.1.171 有、U 没有的前端文件（非测试） | **108 个** |
| 这 108 个里属于 169→171 新增的 | **0 个** ← 不是合 171 漏加的，本来就缺 |
| P 也缺这 108 个里的 | **107 个** |
| 自洽性判据：`import './url'` 的文件数 | **P=0，U=2**；而 `url.ts` 两条线都没有 |

**P 缺同样的文件但能构建，因为 P 连 importer 一起没有 → 自洽。
U 留着 importer 丢了被 import 的模块 → 不自洽。**
所以 U 不是"上游 171 + 我们的定制"，是一个**半成品状态**。

缺失的代表：`api/url.ts`、`api/adminUIRequest.ts`、`utils/oauthAffiliate.ts`、
`utils/safeStorage` 之外的一批 utils、`views/admin/groupsModelsList.ts` 等 5 个 Groups 拆分模块、
`features/prompt-audit/api.ts`、整套 `i18n/locales/{zh,en}/**`、Grok/批量图片/Ollama 相关组件。

要修就得把这 108 个从上游补回来 → **等于把我们从来没有的那批功能一起搬进来**
（Grok、批量图片、Ollama、prompt-audit）。这才是"搬进一个陌生产品"，
而且这次是**实测的**，不是推断的。

⚠️ **171 那次 138 个冲突是在从没构建过的情况下解的**（U 的 `dist` 未跟踪，没人发现）。
两个抽查的语法错是 `b9545e8dd` 解错的（上游自己的副本正常）：
`LinuxDoOAuthSection.vue:23` 少 `const props = withDefaults(`；
`EmailVerifyView.vue:522` 混了两边形态导致 `requestPayload` 引用 3 次却未声明。
**判据别用 `}>(), {` 计数**（P 有 27 个且构建正常，那是正常写法的尾巴）；
正确判据是「有 `}>(), {` 但整个文件没有 `withDefaults(`」→ 修复后 P=0、U=0。

### 已证伪的方案：按 `[ssxz]` 标记重放补丁

有人建议"上游底座 + 把独立的 `[ssxz]` 提交按序重放"。**这个机制没有输入**：

```
分家点以来我们侧提交 757 个，带 [ssxz] 标记的 = 0 个
```

本仓库大量 squash-merge，可重放的干净补丁序列**不存在**，也无法追溯补造。
方向（补丁层化）是对的，但不能以"重放现有 `[ssxz]` 提交"起步。

### ❌ 已作废："U 当主干只需搬 4 项"（我说错了，2026-08-07 修正）

那个结论来自**路由集 + 符号**测法。它们能证明"功能在不在"，
**证明不了"页面长得一样"**——这是这套测法的盲区，也是它漏掉的东西：

| 测项 | 结果 |
|---|---|
| 两条线共有的 `.vue`/`.css`（非测试） | 282 个，其中**内容不同 113 个** |
| 其中**客户可见**（`user/`+`auth/`+`layout/`+`common/`+`views` 根） | **57 个** |
| 仅管理端（`admin/`） | 44 个 |
| 改动最大的 | `SettingsView.vue` 7655 行差、`GroupsView.vue` 3862、`KeysView.vue` 3458 |

客户可见的 57 个里包含**整套设计系统**：`ThemeToggle.vue`、`Toggle.vue`、`Select.vue`、
`DataTable.vue`、`AppHeader.vue`、`AppSidebar.vue`、`GroupSelector.vue`、
`PaymentMethodSelector.vue`、`SubscriptionPlanCard.vue`、`HomeView.vue`、全部 `auth/` 页面。

**所以切主干真实工作量不是 4 项，是 4 项 + 57 个客户可见 UI 文件逐个核对。**

仍然成立的部分（这些测法没错，只是不够）：客户可见路由 P−U = **空集**（P=69/U=79）；
`oauth_compat` 符号 U 已齐 2/2、1/1。

❌ **已纠正：「`UserOrdersView`→`AppOrdersView` 是改名」这句话方向是反的。**
不是上游改的名，是**我们**改的，而且**没删旧的**：

```
P : views/user/AppOrdersView.vue  +  views/user/UserOrdersView.vue   ← 两个都在
U2: views/user/UserOrdersView.vue                                    ← 只有旧的
```

P 的 router 指向 `AppOrdersView`，所以 P 里的 `UserOrdersView.vue` 是死代码。
后果不是"上游缺文件"，而是**搬过去会双份**：U2 保留它自己的 `UserOrdersView`，
我们又搬进 `AppOrdersView`，两个组件同时存在，router 指哪个才是真的。

### ⭐ 盲区四：同一页面、两条线渲染不同组件（2026-08-07 实测，4 条）

`/app/available-channels` 那条警告是对的，但**它不是孤例，是一个族**。
按归一化路径把 P 与 U2 的 router 全量比对，得到 4 条：

| 页面 | P 渲染 | U2 渲染 |
|---|---|---|
| `/available-channels` | `ModelPricingView.vue`（重建的定价页）| `AvailableChannelsView.vue`（旧的）|
| `/orders` | `AppOrdersView.vue` | `UserOrdersView.vue` |
| `/purchase` | `AppPurchaseView.vue` | `PaymentView.vue` |
| `/usage` | `AppUsageView.vue` | `UsageView.vue` |

**只搬文件不重指 router → 这 4 个页面静默变回上游版本。** 文件级账本
（B2/B6 那 845 个）**结构上看不见这一类**，因为两边文件都存在、都"搬到了"。

另外：P 独有页面 **9 个**（`/`HomeView、`/chat`、`/image`、`/docs`、
`/channel-status`、`/admin/affiliates`、`/admin/api-keys` 等），
U2 独有 **16 个**（audit-logs、risk-control、prompt-audit、oidc/wechat/dingtalk
回调、model-plaza、subscriptions、batch-image 等 → 切过去白捡，但要逐个决定要不要）。

### ⚠️ 路由命名空间已分叉：`/app/*` vs 裸路径（切主干必须决定）

**我们把整个客户路由重前缀成 `/app/*`，上游还是裸路径。**
P 里有 16 处 `redirectLegacyRoute('/app/...')` 把裸路径重定向过去。

这也是为什么**第一次比对得到"0 条冲突"——join 键根本对不上**，
`/app/orders` 永远撞不上 `/orders`。**比路由必须先归一化前缀，否则结论无意义。**

切主干时二选一：保 `/app` 前缀（要把前缀层 + 16 个重定向一起搬）
或退回裸路径（客户已存的书签/外链会断）。**不能不选。**

### 主题切换（太阳/月亮）：不是回退，线上就是我们自己的（已实测结案）

怀疑"线上主题按钮变回上游出厂样式"。**实测证伪**——直接查已部署的入口 chunk：

```
入口 index-Bn3BFivZ.js（与 index.html 声明一致）
theme-toggle 类名 ×1 · 我们的 sun 路径 M12 3v2.25 ×1 · 我们的 moon 路径 M21.752 ×1
nav.lightMode ×1 · lucide 痕迹 ×0
```

| 事实 | 值 |
|---|---|
| `ThemeToggle.vue` 在上游 v0.1.171 | **不存在**（这是我们自己写的组件，无"上游出厂版"可回退）|
| P 的 `ThemeToggle.vue` 最后改动 | `2e7c8ec1f` **2026-07-13**，之后没动过 |
| 今天那个碰 `Icon.vue` 的提交 `229004d93` | **没有改 sun/moon 路径**（+1 行，与图标无关）|
| P 与 U 的 `ThemeToggle.vue` | **不同**：P 用自家 `Icon.vue`（Heroicons 路径），U 用 `@lucide/vue` 的 `Moon/Sun` |

所以**方向是反的**：切到 U 才会让主题按钮换成 lucide 版。线上现在是我们的版本。
观感变化更可能来自 `1c3bc36ca`（2026-07-16 全站单色主题统一），不是代码丢失。

### ⭐ 已定方向：Upstream-first，主干走干净的 U2（2026-08-07 用户确认）

**长期方向：以上游受信发布线为主干，SSXZ 定制作为可审计、可重放的补丁层叠加。**
P 冻结为可回滚基线继续服务；**当前那个坏掉的 U 只留作取证，不再修补、不作为主干**。

我今天写过"主干只能是 P"，那是基于「U 不自洽」这一条。这个前提**只对 U 成立，不对干净上游成立**——
下面这两条实测事实仍然正确，但它们判的是 U，不是 U2：

| 事实 | 后果 |
|---|---|
| U 缺 108 个上游前端文件，却留着 import 它们的文件 | U **不自洽**，`vue-tsc` 625 错 / 40 文件、前端测试 133 挂 |
| P 缺同样的 107 个，但 import 它们的文件数 = **0** | P **自洽**，能构建、能部署、正在服务客户 |

**U2 前置条件已实测通过**（用抓出 U 的同一个判据）：

```
U2 基线 = v0.1.171 = afd154b92aac36c6dafb1fa8e181ca827c78c465
upstream/main = 00b859617，仅领先 3 个提交，v0.1.171 是其祖先
U 里悬空的 5 个模块（api/url、api/adminUIRequest、utils/oauthAffiliate、
  features/prompt-audit/api、views/admin/groupsModelsList）在 v0.1.171 里全部存在
「有 }>(), { 但整文件无 withDefaults(」→ v0.1.171 = 0 个
```

### ✅ U2 四道闸门已实测（2026-08-07，worktree `F:\CodexTemp\upstream-v0.1.171-clean`）

`v0.1.171` 是**附注 tag**：tag 对象 `afd154b92`，指向 commit `f0e7a9c7a`。两个都对，别当成建错了树。

| 闸门 | U2 结果 | 同项 U 的结果 |
|---|---|---|
| 后端裸编译（linux/amd64/CGO=0） | ✅ 0 | ✅ |
| `go test ./...` | 46 包过 / **1 包 1 个用例失败** | 全绿 |
| 前端 typecheck | ✅ **0 error** | ❌ 625 error / 40 文件 |
| 前端 build | ✅ 15.98s，171 个 asset，入口 `?v=` 0 处 | ❌ 构建不出来 |
| 前端 vitest | 206/207 文件过，**2 个用例失败** | ❌ 60 文件 / 133 用例失败 |
| 带 `-tags embed` 编译 | ✅ 156,460,544 字节 | 未做 |
| SSXZ 改动数 | **0**（`git diff v0.1.171` 空，仅 dist 产物落在 gitignore 内）| — |

**那 3 个失败全是上游自己的测试缺陷，不碰实现代码，不阻塞换主干：**

1. `TestContentModerationRuntimeSnapshotRefreshFailureKeepsStaleConfig`（稳定复现 3/3）——
   `content_moderation.go` 在 U 与 U2 **字节级一致**，差的只有测试文件。上游靠
   `runtimeCacheTTL=1ns` 等自然过期，进不去 refresh 分支；**同文件里的兄弟用例
   `...BacksOff` 手工写 `expired.loadedAt = time.Now().Add(-time.Second)` 就过**。
   我们那 5 行（`e5c51dce9` 引入）正是这个手工过期，是**真补丁，该重放**。
   判据：`expired.loadedAt` 在 v0.1.165/169/171 都是 4 次，U 是 5 次。
2. `admin.system.rollback.spec.ts` 2 个用例——实现多传第 3 参 `{timeout:900000}`，
   测试只断言 2 参。该文件 **P 和 U 都没有**，纯上游内部不同步。
3. 10 条 `adminAPI.groups.getLiveCapability is not a function` ——**不是缺实现**
   （`api/admin/groups.ts:90` 有），是测试 mock 不全导致的 unhandled rejection，
   没让任何用例失败。

⚠️ **Codex 那份"前端环境阻塞、typecheck/build/vitest 无法执行"的结论已过期。**
`pnpm install` 其实在它停止等待之后装完了；离线复验 `Already up to date`（1.3s）。
真 store 在 `F:\CodexDev\pnpm-store\v10`（**不是**仓库里那两个 `.pnpm-store`）。
pnpm 10 会忽略 `esbuild`/`vue-demi` 的 build script（`.npmrc` 的 `ignore-scripts=false`
已不足，需 `onlyBuiltDependencies`），但实测不影响 build。
U2 与 P 的 `pnpm-lock.yaml` 不同 blob → **不能共用 node_modules**，各装各的。

**结论：U2 可以当主干基础。** 三个失败已定位到上游测试本身，不是我们的环境、
也不是 U2 不自洽。换主干的闸门（全量测试）**允许**带着这 3 个已归因的上游缺陷通过。

### 重放账本 v2（2026-08-07 实测，已带锚点自检，清单已落盘 `F:\CodexTemp\ledger\`）

**基准是「分家点 `bda7c39e5` 以来我们改过的文件」，不是「P 与 v0.1.171 的 diff」。**
后者混进了上游 3134 个提交的改动，不能当账本。

| 层 | 数 |
|---|---|
| 分家点以来我们改过（剔构建垃圾） | **1381** |
| 减去测试 | −367 |
| 减去 `backend/ent/` **生成代码**（`// Code generated by ent, DO NOT EDIT.`，靠重新生成而非手抄） | −127 |
| **→ 有效基数** | **887** |

按「P 的 blob vs U2 的 blob」逐个比，分桶（**合计精确闭合 887**）：

| 桶 | 数 | 动作 |
|---|---|---|
| B1 我们删了、上游还有 | 2 | 确认是否继续删 |
| B2 只有我们有 | **340** | 整文件直接搬 |
| B3 内容已相同 | 40 | **零工作量** |
| B6 双方都改过 | **505** | 必须语义合并，不能整文件覆盖 |
| **B7 上游删掉、我们从没改过** | **20** | ⚠️ **不在 887 里**（见下）|

**真实盘子 = 907，需动手 845（B2 340 + B6 505）+ B7 决策 20。**

⚠️ **已作废：旧的「A 类 399 / B 类 488」**。那是按"谁改过"（历史）分的，
现在这版按"内容是否真的不同 + 文件在不在 U2"分（决策口径），且过了锚点自检。
旧基数 1361 也作废——**那个过滤器已证明是坏的**（见下方方法红线），
`-z` 修好后多出的 8 个是非 ASCII 路径（`docs/教程…`）。

B6 按差异行数分层（**合计 189,897 行**）：S≤20 行 **91** · M 21–100 **151** ·
L 101–500 **164** · XL>500 **93**。最贵：`SettingsView.vue` 14377 行 ·
`gateway_service.go` 9296 · `openai_gateway_service.go` 6362 · `GroupsView.vue` 5660。

⚠️ **我"内容比对能压掉一大截"的预期是错的，只压掉 48 个。**
B4「我们其实没改过」= **0**、B5「只是吸收过上游」= **6** ——
说明我们碰过的文件几乎全是真定制，不存在"上游已经帮我们做了"的大块红利。

**B2 的 import 闭包已验（这是 U 死掉的死因，必须先验）**：85 个前端文件、
1623 条 import、可解析 201、**悬空仅 4 条且全指向同一个目标 `frontend/src/api/sora`**
（属 B7）。→ **Sora 决策一落，B2 前端闭包即干净。** Go 那 157 个的闭包
grep 验不出来，只能靠编译。

⚠️ **账本盲区（B7）：基准"我们改过的文件"对依赖不闭合。**
上游**主动删掉**的文件、我们又从没改过 → 它不在 1381 里，但我们的文件在 import 它。
`api/sora.ts` 就是这么漏的。20 个里 **19 个是 Sora**。

### Sora：一个可独立决策的 32 文件切片

| 桶 | Sora 文件 |
|---|---|
| B2 只有我们有 | 13 |
| B6 双方都改过 | **0** ← 我们从没和上游在 Sora 上冲突过 |
| B7 上游删掉 | 19 |

**上游有 `backend/migrations/090_drop_sora.sql`（P 和分家点都没有）**，内容是
`DROP TABLE sora_tasks / sora_generations / sora_accounts` + 砍 `groups`/`users`/`usage_logs`
上 8 个列。这是迁移里第一个**真正破坏性的闸门**。

⚠️ **`/api/v1/sora/models` 返回 401 只证明二进制里有路由，不证明 Sora 表还在**
（鉴权中间件在 DB 访问之前就返回了）。表在不在必须查库，未查。

⚠️ **CLAUDE.md 上文那条「切 U 会跑 0 个 migration」不能外推到 U2。**
那是对 **U** 测的（U 的 >138 集合 86 个，`GT138_SET_DIFF=EMPTY`）。
**U2 与 P 相比多 142 个 migration 文件**，其中带破坏性 DDL 的 4 个：
`090_drop_sora.sql`（DROP TABLE+DROP COLUMN）· `127_drop_channel_monitor_deleted_at.sql` ·
`136_remove_ops_retry_replay.sql`（DROP TABLE+TRUNCATE）· `180_audit_logs.sql`（TRUNCATE）。
**这 142 个在生产库里有没有留痕 = 换主干前最后一个未知项，只读 SELECT 就能答。**

### ⚠️ 方法红线（这两条今天各绊我一次，都写死在这）

1. **垃圾过滤必须在"提取出路径之后"再跑。** `git ls-tree` 每行是
   `<mode> <type> <sha>\t<path>`，对着整行做 `^backend/rust/target/` 这种锚定匹配
   **全部静默失效** → `.pnpm-store` 53834 条和 `rust/target` 2688 条会留在树里，
   桶数直接虚高。**必须先 `-z` + `core.quotePath=false` 切出路径，再过滤。**
   **锚点自检**（每次重算都要过）：`P frontend/src = 574`、`U2 frontend/src = 709`、
   垃圾残留 = 0。三条不齐，数字一律不许往外报。
2. **401/404 路由探测必须带负控。** 只看到"Sora 返回 401"不能下结论——
   万一鉴权中间件跑在路由之前，401 就什么都不证明。
   实测负控：`/api/v1/definitely-not-a-route-xyz` → **404**，探测法**有效**。
   顺带确认：`/api/v1/reseller/summary` 和 `/api/v1/admin/reseller/applications`
   都是 **404**（= CLAUDE.md 说的 8 条死路由，属实）。

### 优先项（客户正在受影响，与主干迁移解耦、可先做）

1. **Reseller** —— 生产库已有 200/201 和数据，P 有 0 个 reseller 文件 → 8 条路由现在 404。
   已实测可整块搬：15 个前端 + 5 个后端（非测试），22 个 `@/` 依赖里 P 只缺 5 个，
   其中 4 个本身就是 reseller 文件，**唯一外部依赖是 `@/components/common/LiquidButton.vue`**
2. **`channel_id=NULL` 计费修复**（`9e9440e35`）——只挑 3 个符号，
   `usageChannelMappingForAPIKey` / `channelMappingResolver` / `GroupIDForUsage`，**不要整文件覆盖**

- **唯一主干**：任何时刻只有一条线可以编译部署。**必须先立**——
  否则合过的东西会被另一条线的部署第三次顶掉
- 每次部署必须记三样：**上游底座 tag + 我们的 HEAD + 二进制 MD5**（`DEPLOYED.md`）
- 上游安全补丁不会自动到我们手上 → 必须定期看上游 release，有则单独挑
- 截至 2026-08-07，本地可见最新上游 tag 是 `v0.1.171`（**没有 172/173**）
- **比对功能差异只用符号法/路由法，不用文件名法。** 文件名法今天连续骗了三次：
  「P 独有 56549 个文件」（56515 个是 `.pnpm-store`+`rust/target` 构建垃圾）、
  「U 缺 UserOrdersView」（**当时说"改名了"，这个解释也是错的——见 §6：
  是我们加了新名字且没删旧的，P 里两个都在**）、「U 缺 oauth_compat」（符号全在）
- ⚠️ **basename 配对法是垃圾，别用。** 想查"上游是不是把文件换了路径"时，
  拿同名文件互配会得到一堆假阳性：实测 50 条命中里 0 条是真的，
  `tools/perf/README.md` 会配上 `backend/migrations/README.md`，
  `components/foundation/index.ts` 会配上 15 个不同的 `index.ts`。
  **零假阳性的判据只有一个：blob 完全一致而路径不同**（实测 B2 340 个里 0 条）。
- ⚠️ **`git diff -M` 在大树上会静默跳过改名检测。** 它只在 stderr 写一行
  `warning: exhaustive rename detection was skipped due to too many files`，
  退出码仍是 0。**丢掉 stderr 就会拿到"只有 4 条改名"这种假结论。**
  必须 `-l0` 关掉上限并且读 stderr。（关掉后实测：上游自分家点以来真改名 **1 条**，
  而那 1 条 `soraerror.go → httputil.go` 是相似度误配，不是真改名。）
- **但符号法/路由法也有盲区：它们证明不了"页面长得一样"。** 涉及 UI 时必须另外
  逐个 diff `.vue`/`.css`（今天就是这么漏掉 57 个客户可见文件的）。
  **自洽性测法**（有没有 import 却不存在的模块）比两者都强，是今天唯一一次
  直接定住"哪条线能构建"的判据
- ⚠️ **数文件前先剔垃圾，而且 `.pnpm-store` 有两个**：仓库根 `.pnpm-store/v10`（52871 个）
  和 `frontend/.pnpm-store/v10`（53833 个）。只剔一个会让计数虚高约 50 倍
  （实测：漏剔根目录那个 → 账本从 1361 虚报成 55195）。还要剔 `backend/rust/target`
- ⚠️ **`backend/ent/` 是生成代码**（文件头 `// Code generated by ent, DO NOT EDIT.`），
  127 个文件**不要手工重放**，改 schema 后重新生成。算重放工作量时必须先扣掉

## 7. 红线

- 入口 JS **禁止**加 `?v=` query（两个 URL = 两份模块 = 双 mount 黑屏，2026-08-07 黑屏根因）
- 编译生产二进制必须 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0`，漏设立刻 crash 回滚
- 往 `channel_model_pricing` 加行会**直接改扣费**，那张表是计费覆盖层
- 不要手改 `VERSION` 文件，`release.yml` 会从 tag 覆写它
- 重启后首轮请求约慢一倍（冷启动，已实测 7.8–11s vs 热态 3.4s）——
  **不要**据此判定延迟回归或触发回滚，先等温机复测
- **解完冲突必须当场构建 + 跑测试再提交。** `b9545e8dd`（合 171，138 个冲突）
  就是反面案例：它把 `LinuxDoOAuthSection.vue`（丢了 `withDefaults(` 包裹）和
  `EmailVerifyView.vue`（`requestPayload` 用了但没声明 + `const response` 声明两次）
  解坏了，**U 的前端至今 `vue-tsc` 都过不去**。因为 U 的 `dist` 未跟踪、当时也没构建，
  这两个语法错误就这么提交进去了。抽查的前 2 个冲突解法**全是错的**——
  所以 U 上剩下的 136 个冲突里很可能还有编译器查不出来的**语义**错解，
  换主干的闸门必须是**全量测试**，不能只看"能不能构建"
