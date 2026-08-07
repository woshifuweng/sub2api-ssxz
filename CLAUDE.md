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

## 6. 真问题是"哪条线当主干"，不是"要不要跟上游"（2026-08-07 修正）

⚠️ **"分家"这个词是错的，已作废。** 上游 remote 从没断开（`upstream` →
`github.com/Wei-Shaw/sub2api.git`，仍可 fetch）。2026-08-05 那次合 v0.1.171
（111 提交 / 138 冲突）**是真的合过**，合并提交 `b9545e8dd` 就在 `upgrade/v0.1.169` 上。
问题不是"从没跟上游"，而是**部署是从另一条不含 171 的线出去的**。

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

### U 前端构建失败 = 合并解错，不是上游缺陷（已定位到行）

U 后端交叉编译**成功**（162,165,705 字节，`GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 已确认）。
前端 `vue-tsc` 挂在 **2 个文件**，都是 `b9545e8dd`（合 171 那次）最后改的，**上游自己的副本是好的**：

| 文件 | 缺陷 | 上游对照 |
|---|---|---|
| `LinuxDoOAuthSection.vue:23` | `defineProps<{` **没有 `withDefaults(` 包裹**，但尾部留着 `}>(), {` | v0.1.169/171 都有 `withDefaults` |
| `EmailVerifyView.vue:522` | 保留了 P 的 `const response = await sendVerifyCode({`，但函数体和收尾来自上游的 `const requestPayload = {` 形式 → `requestPayload` 被引用 3 次却**从未声明**，且 `const response` 声明两次 | v0.1.171 有 `const requestPayload = {`（531 行）|

**判据（别用 `}>(), {` 计数，会误报）**：`有 }>(), { 但整个文件没有 withDefaults(` → P=0 个，U=1 个。
P 里有 27 个 `}>(), {` 且构建正常，那是正常写法的尾巴。

⚠️ **这暴露一个更大的信号：171 那次 138 个冲突是在从没构建过的情况下解的**（U 的 `dist` 未跟踪）。
抽查的头 2 个都解错了 → 换主干前**必须跑全套测试**，不能只看编译过。
编译能过不代表语义对：这 2 个是语法错所以 tsc 抓到了，解错成"语法对但语义错"的不会报错。

### 已证伪的方案：按 `[ssxz]` 标记重放补丁

有人建议"上游底座 + 把独立的 `[ssxz]` 提交按序重放"。**这个机制没有输入**：

```
分家点以来我们侧提交 757 个，带 [ssxz] 标记的 = 0 个
```

本仓库大量 squash-merge，可重放的干净补丁序列**不存在**，也无法追溯补造。
方向（补丁层化）是对的，但不能以"重放现有 `[ssxz]` 提交"起步。

### U 当主干需要搬的东西（已用符号/路由法测清，不是文件名法）

**功能层面 U 几乎是 P 的超集**，"P 独有 33 个源码文件"里绝大部分是文档/截图/skill：

| 测项 | 结果 |
|---|---|
| 客户可见路由 P−U | **空集**（P=69 条，U=79 条）|
| `oauth_compat` 符号在 U | **2/2、1/1 全齐** → 不用搬 |
| `UserOrdersView` | U 的 `AppOrdersView` 是**改名**（`getMyOrders`/`getRefundEligibleProviders` 都在）→ 不用搬 |
| U 白捡的 Reseller 路由 | **8 条**（`/app/reseller/*` + `/admin/reseller/*`）|

真正要搬的只有 4 项：

1. 今天 5 个修复：兑换码去 Turnstile / `parseVersion` / 作图页 `AppSectionShell` / CSP / dist 重建
2. `ModelPricingView.vue` + `modelPricing.ts` + **router 把 `/app/available-channels` 指回去**
3. `frontend/src/api/affiliate.ts`（4 个符号在 U 整棵树 0 命中，真缺）
4. `frontend/public/logo.png`（U 里没这个文件）

**⚠️ 第 2 项里的 router 重指是最容易漏的一步**：`/app/available-channels` 这条路由
**两条线都有**，所以路由集比对是空集查不出来。但它渲染的组件不同——
P 是 `ModelPricingView.vue`（今天重建的定价页），U 是 `AvailableChannelsView.vue`（旧的）。
**只切主干不重指，今天的定价页会静默消失。**

### 站得住的规则

- **唯一主干**：任何时刻只有一条线可以编译部署。这条规则跟"选 P 还是选 U"无关，
  但**必须先立**——否则合过的东西会被另一条线的部署第三次顶掉
- 每次部署必须记三样：**上游底座 tag + 我们的 HEAD + 二进制 MD5**（`DEPLOYED.md`）
- 上游安全补丁不会自动到我们手上 → 必须定期看上游 release，有则单独挑
- 截至 2026-08-07，本地可见最新上游 tag 是 `v0.1.171`（**没有 172/173**）
- **比对功能差异只用符号法/路由法，不用文件名法。** 文件名法今天连续骗了三次：
  「P 独有 56549 个文件」（56515 个是 `.pnpm-store`+`rust/target` 构建垃圾）、
  「U 缺 UserOrdersView」（改名了）、「U 缺 oauth_compat」（符号全在）

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
