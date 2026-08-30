# SSXZ 项目 — 每个会话开工前必读

这个文件会被自动读进每个新会话。它只写"会让你判断错的事实"，不写别的。

## 1. 真源码不在 `backend/`

主目录 `backend/` 是空壳（0 个 `.go` 文件）。往那里 grep 会**静默返回空**，读起来像"这功能不存在"——这是本项目最容易上的当。

```
真源码（U2 主干，2026-08-30 起）：
  本地工作树：F:\CodexTemp\upstream-v183-ssxz-20260830\
  分支：codex/upstream-v183-ssxz-20260830
  远端：github.com/woshifuweng/sub2api-ssxz.git
  当前生产 HEAD：cf6e05948（版本 0.1.183-ssxz.20260830.4；上游 v0.1.183 底座 + SSXZ UI/业务覆盖、企业专属分组兼容与最终页面修复；生产身份仍以 DEPLOYED.md 为准）
同名分支历史 P 线（已退役）：.codex-work/fix-client-brand-announcements2/
← backend/仍是空壳，P 线的路径规则不变；P 不再修改，只做回滚基线。
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
>⚠️ **2026-08-13 更新：以上血脉测试针对已退役的 P 线。当前生产是 U2 主干（见§6），直接基于 v0.1.171，血脉问题不再适用于日常 grep。**

## 5. 生产路由探测（判断线上到底部署了哪条线）

未鉴权 GET 一个 admin 路由：`401` = 路由存在，`404` = 路由不存在。用一条两条线都有的路由当控制组：

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://api.ssxzapi.com/api/v1/admin/users   # 控制组，应为 401
```

## 5.5 ⛔ v0.1.172 账号接管 0day — P **结构上有洞**，靠"OAuth 全关"挡着（2026-08-07 实测）

用户指令：**「必须合并 172，171 之前的版本都有大漏洞」**。上游 release 说明：
"攻击者仅凭受害者邮箱，即可通过 OAuth 登录补全流程把自己的第三方身份绑定到他人账号并直接登录"。

**修复本体只有 1 个文件 15 行**：`02e50cc22`（Puppywang，2026-08-07 03:09）改
`backend/internal/handler/auth_oauth_pending_flow.go`，加一道闸：

```go
if !canIssueTokenPair && !strings.EqualFold(strings.TrimSpace(session.Intent), oauthIntentBindCurrentUser) {
    response.Success(c, payload)
    return
}
```

位置：`ExchangePendingOAuthCompletion` 里，**紧接 `pendingSessionWantsInvitation` 那个 return 之后、
`if !adoptionDecision.hasDecision()` 之前**。

### P 的实测状态：符号全在、闸没有

| 测项 | P | 上游 172 |
|---|---|---|
| `auth_oauth_pending_flow.go` | 有，2060 行 | 2066 行（171 是 2051）|
| `ExchangePendingOAuthCompletion` / `applyPendingOAuthAdoption` / `canIssueTokenPair` / `oauthIntentBindCurrentUser` | **全部存在** | 存在 |
| `session.Intent` 字段 / `strings` import | 有 / 有（**补丁能直接编译**）| — |
| **那道闸** | **不存在** | 存在（2012 行）|
| `pendingSessionRequiresBindLogin`（172 里紧挨补丁的兄弟闸）| **0 处** | 3 处 |

攻击链在 P 里逐环都对得上：`createPendingOAuthAccount`(594) 和
`SendPendingOAuthVerifyCode`(1759/1772) 发现邮箱已存在 → `transitionPendingOAuthAccountToChoiceState`
→ `updatePendingOAuthSessionProgress`(731) `SetTargetUserID(受害者)` → exchange 带 adoption decision
→ `applyPendingOAuthAdoption`(2033) 把攻击者 identity 绑到 `session.TargetUserID`。**全程无密码、无验证码。**

### 唯一挡着的东西：生产 OAuth 全关（**不是代码修好了**）

```
"linuxdo_oauth_enabled":false   "wechat_oauth_enabled":false
"oidc_oauth_enabled":false      "wechat_oauth_open_enabled":false / mp / mobile 全 false
```

pending session **只有一个铸造点** `createOAuthPendingSessionWithContext`
（`auth_oauth_pending_flow.go:184`），调用者只有 linuxdo / oidc / wechat 三家的 callback，
三家的 config resolver 都会 `NotFound("OAUTH_DISABLED")`（linuxdo:593、oidc:744、wechat:1128/1178）。
→ **拿不到 pending cookie，链条第一环就断。**

⚠️ **但 `/oauth/pending/*` 四条路由本身是活的、免鉴权的**，已实测（生产 POST）：

| 探测 | 响应 |
|---|---|
| `/api/v1/auth/oauth/pending/exchange` | `{"code":404,...,"reason":"PENDING_AUTH_SESSION_NOT_FOUND"}` ← **handler 在跑** |
| `/api/v1/definitely-not-a-route-xyz`（负控）| `404 page not found` ← gin 的 |
| `/api/v1/auth/oauth/pending/send-verify-code` | `400 Field validation for 'Email' failed` ← **在跑** |
| `/api/v1/auth/oauth/pending/bind-login` | `400 ...` ← **在跑** |

⚠️ **红线：在补丁上线之前，后台绝对不许打开任何一个 OAuth 开关**（linuxdo / wechat / oidc / 微信支付授权）。
现在的安全性由一个**后台开关**提供，不由代码提供——**点一下开关就是真洞**。
✅ **微信支付那条已查清，不是待验**（读代码即可，不需要跑东西）：
`WeChatPaymentOAuthStart/Callback` 调的是**同一个** resolver
`getWeChatOAuthConfigGateway(ctx, "mp", c)`，闸是 `SupportsMode("mp")` → `MPEnabled`；
而 `parseWeChatConnectOAuthConfig` 先要求 `cfg.Enabled && (Open||MP||Mobile)` 才不报 OAUTH_DISABLED。
→ **微信支付授权需要 `wechat_oauth_enabled` 且 `wechat_oauth_mp_enabled` 同时为 true，
与其他微信开关同源、主开关也在闸上。** 上面那条红线是完整的。

### 172 的真实体量：真源码 122 文件 / +2384−612 行（**不是 208**）

❌ **已作废我自己报的「208 文件 / 142 非测试」**——那个数把测试和生成代码算进来了，
虚高一倍多。171→172 全区间 +6215/−820，按类别拆开才是真答案：

| 类别 | 文件 | 行 |
|---|---|---|
| **真源码** | **122** | **+2384 / −612** |
| 测试 | 66 | +3130 |
| `backend/ent/` 生成代码 | 9 | +636/−47（重新生成，不手抄）|
| README + assets + 赞助商图片 | 10 | +64 |
| `VERSION` | 1 | — |

同区间带着这些（其中两条正好打我们踩过的坑）：

| 提交 | 内容 |
|---|---|
| `db0bff82c` | **上游响应模型审计**——能识别上游偷偷替换/降级模型（对应我们查过的调度/计费疑点）|
| `e687ca3e9` | 系统日志落库失败后退避重试，**避免拖垮数据库连接池** |
| `99b357083` | 订阅每日零点配额重置修复 |
| `c33c3208e` | 流内降载错误恢复 pre-output failover |

### ✅ checksum 闸**不需要重测**（已读 diff 定案，别再写"必须重测"）

我一度写了「`migrations_runner.go` 在 171..172 被改了 → 必须重测 checksum 闸」。
**读 diff 就能定案，实测结论是不用重测：**

| 测项 | 结果 |
|---|---|
| `migrations_runner.go` 改动 | **4 行，纯新增**：2 个 const + `prepareNonTransactionalMigration` 里 1 个 `case` |
| checksum 计算块 / `migrationChecksumCompatibilityRules` | **一个字没动** |
| `backend/migrations/` 改动 | **新增 2 个，修改 0，删除 0** |

那一闸只依赖三样：算法、白名单、已有文件内容。**三样都没动 → 171 的结论直接继承。**
（新增的 4 行是自愈逻辑：`CREATE INDEX CONCURRENTLY` 失败会留下 INVALID 索引，
runner 在重跑前先 `dropInvalidIndexIfPresent` 掉。）

新增那 2 个 migration **会真的执行**，但都安全：

```sql
-- 194_add_usage_log_upstream_response_model.sql
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_response_model VARCHAR(200),
    ADD COLUMN IF NOT EXISTS upstream_model_mismatch BOOLEAN;
-- 195_..._notx.sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_upstream_model_mismatch_created_at
    ON usage_logs (created_at DESC, id DESC) WHERE upstream_model_mismatch IS TRUE;
```

可空无默认的 `ADD COLUMN` 在 PG 11+ 是纯元数据操作（`usage_logs` 再大也是瞬时）；
`CONCURRENTLY` 不锁写，且是 partial index、初始全 NULL → 索引极小。

⚠️ **但发现一个号段撞车（换主干要知道）**：194/195 这两个号在 U 上**已经被别的文件占了**——

| 号 | 上游 172 | U（生产库跑过的那条线）|
|---|---|---|
| 194 | `194_add_usage_log_upstream_response_model.sql` | `194_create_chat_workspace_tables.sql` |
| 195 | `195_..._model_mismatch_index_notx.sql` | `195_allow_workspace_image_messages.sql` |

runner 主键是**完整文件名**，所以两边不冲突、各记一行、上游那 2 个会正常执行。
**不构成闸门**，但号段从此双轨，以后自己加 migration 别再用 19x。

### ✅ P 的等价补丁已部署生产（**2026-08-08 00:58 UTC**，4988a280b）

**不整版合 172，只把那 15 行搬到 P。** 理由：整版 208 文件要过 §6 那整套账本，
而洞只有 1 个文件；先堵洞、再按 upstream-first 慢慢换主干，两件事解耦。

```
隔离 worktree: F:\CodexTemp\hotfix-oauth-takeover （从 P 的 e8ef9e645 新建，没碰 P 的树）
分支/提交:     hotfix/oauth-pending-account-takeover  →  4988a280b
改动:          auth_oauth_pending_flow.go +15 行（闸门）
               auth_oauth_pending_takeover_guard_test.go +128 行（新增回归测试）
二进制（可部署）: sub2api_linux_hotfix_4988a280b
                 93,733,026 字节 · MD5 8e24a2433d24fa3b545aace7ab67592b
                 版本戳 0.1.3-ssxz.20260807.1（与线上 …20260807 可区分，便于验证真上线）
构建命令:        与 DEPLOYED.md:260 一致（-tags embed -trimpath -s -w + 三个 -X）
```

⛔ **差点造成全站黑屏的坑（已避开，必须写死）：`dist` 只有 `index.html` 被 git 跟踪，
188 个 asset chunk 全部未跟踪。** 所以 `git worktree add` 建出来的新树里 `dist` 只有 1 个文件
（815 字节），直接 `-tags embed` 编出的二进制**只嵌了 index.html**，上线后每个 JS/CSS 都 404。

⚠️ **假绿灯判据**：我一开始用 `git diff` 比两边 `dist`，得到"完全一致"——**git 只比跟踪文件**。
**必须按文件系统比**（文件数 + 字节数）：P = **189 文件 / 5,379,192 字节**。
体量自检：正确的 embed 二进制 ≈ **93.7 MB**，只嵌 index.html 的 ≈ **88.3 MB**，差的 5.4 MB 就是它。

修法：编译前把 P 的 `backend/internal/web/dist` 整个复制进隔离树，再 `-tags embed`。
（本次前端零改动 → 用 P 的 dist 就等于线上那份，天然避开"embed 了旧前端"那个老坑。）

⚠️ **`strings`/`grep` 在这台机器上查不了大二进制**（对生产二进制查 `0.1.3-ssxz` 都返回 0，
而生产明明在发这个版本号）。**必须用 PowerShell 读字节转 Latin-1 再 `IndexOf`**，
并且**永远带一个必须命中的控制组**（拿生产二进制当控制组），否则"没找到"全是假阴性。

| 闸门 | 结果 |
|---|---|
| `go vet` 该包 | ✅ 0 |
| 生产交叉编译 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` | ✅ 0 |
| **全量 `go test ./...`**（不带 embed）| ✅ **37 包 ok / 0 FAIL** |
| **全量 `go test -tags embed ./...`**（=上线形态）| 37 ok / **1 包 panic**：`internal/server`，`embed_on.go:158` nil deref。**在 P 上跑同一条命令 panic 在同一行同一调用链**（已实测）→ **P 原有，非本补丁引入**；生产 NRestarts=0 |
| `internal/web` 测试（只在 `-tags embed` 下存在）| ✅ ok |
| 新增回归测试 11 个子用例 | ✅ 全绿 |

补丁与上游**逐字相同，只有一处必要改写**：P 的函数是 gateway-context 版
（`ExchangePendingOAuthCompletionGateway`），所以 `response.Success(c, payload)`
写成 `c.WriteJSON(http.StatusOK, payload)`。位置、条件表达式、常量全部一致。

**回归风险已实测排除（针对「修了这边别坏那边」）**：闸门拦住后 exchange 不再落
adoption decision，但**终态端点不依赖它** —— `auth_linuxdo_oauth.go:564` 和
`auth_oidc_oauth.go:715` 都是拿**自己请求体里的** `req.AdoptDisplayName`/`req.AdoptAvatar`
调 `ensurePendingOAuthAdoptionDecisionWithContext`。上游 172 同构
（`bindOptionalOAuthAdoptionDecision` 全仓只在 exchange 这一处被调用）。
另外 P 里 `step` 只有**一个**非空取值 `choose_account_action_required`，
所以被闸门影响的状态是封闭可枚举的，不存在没数到的分支。

⚠️ **测试覆盖的诚实边界**：上游那个端到端测试依赖 3501 行 / 37 测试的 harness
（`newOAuthPendingFlowTestHandler` + Affiliate/Promo/Totp/SecretEncryptor 一整套），
**P 里完全不存在**（P 的 handler 包 0 个 pending-OAuth 测试）。搬 harness 比搬补丁本身大得多，
所以我改成测**纯函数判据**：锁死 `pendingOAuthCompletionCanIssueTokenPair`
在 choice 状态返回 false（闸门左半边），加闸门真值表。
→ **没有**端到端"攻击者 identity 真的没写进库"的断言，那个等换 U2 主干时白捡。

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
| **U2** = `74178321f`（`F:\CodexTemp\upstream-v179-security-20260824\`，`codex/upstream-v179-security-20260824`）| ⭐ **当前生产线（2026-08-24 起）**。底座已选择性推进到 v0.1.179；含后台账号批量用量接口修复，保留 SSXZ 定制与安全加固，排除破坏性指纹回填及渠道倍率/分时计费改动。完整线上身份见 `DEPLOYED.md` |
| **P** = `e8ef9e645`（`.codex-work/fix-client-brand-announcements2/`）| **已退役，仅作回滚基线**。回滚备份：`/opt/sub2api/backups/sub2api-pre-u2176-20260813`（MD5 `283acdf0784aa05b6e8fd82469c51b5f`）|
| **U** = `upgrade/v0.1.169`（`9e9440e35`）| **取证/对照，不再使用** |

下面 §6 余下各节是这个方向的**证据底座**，不是相反结论——尤其"P 自洽 / U 不自洽"
是**关闭修补 U 这条路**的依据，不是否定 upstream-first。

### 两条线的实际家底（2026-08-07 实测）

| | 生产线 P (`e8ef9e645`) | 升级线 U (`upgrade/v0.1.169` = `9e9440e35`) |
|---|---|---|
| 含 v0.1.171 / `b9545e8dd` | **否 / 否** | **是 / 是** |
| 上游新增生产 Go 文件（279 个去重） | **有 0 个** | **有 279 个** |
| Reseller 经销商文件 | **已恢复并随 r2 上线**（当前已部署提交 `c346c69e3`） | **31 个** |
| `channel_id=NULL` 计费修复 | **已含 `8413e2149`**（已确认是已部署提交 `c346c69e3` 的祖先） | 172 缺失，后续方向为 **P→172** |
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

### ⭐ B6 真实成本 = 63,166 行，不是 189,897（2026-08-07 修正 3.0×）

❌ **作废：「B6 合计 189,897 行，`SettingsView.vue` 14377、`GroupsView.vue` 5660」。**
那个数是 **P↔U2 的 diff**，其中 50.4% 是**上游 3134 个提交自己的改动**，不是我们的工作量。
`rep`（P↔U2）能告诉你"合并时会撞多少行"，**不能**告诉你"我们改了多少" —— 别混用。

三个口径必须分开：`ours` = fork→P（我们改的）· `theirs` = fork→U2（上游改的）· `rep` = P↔U2。

⚠️ **但 `ours` 对"导入文件"也是错的**：2026-04-25 有一次上游同步
（`412a04e2b` "Synchronize upstream features and native gnet auth routes" 加 22 个 +
`8c7798584` "优化" 加 14 个），把上游代码**当成我们的新增**算进了 `ours`。
141 个 B6 文件在分家点**不存在**，其中 129 个是这两个提交带进来的。
这类文件的基线必须**重建为最近的上游 blob**，不能用分家点。

按重建基线重算，505 个 B6 的真实结构：

| 桶 | 数 | 真实行数 | 旧 `ours` 报的 |
|---|---|---|---|
| 纯上游副本（blob 与某上游 tag 字节一致）| 26 | **0**（直接取 U2）| — |
| 导入时就改过（非纯副本）| 51 | **1,795**（均 35）| 12,491（**虚高 7×**）|
| 导入后又改过 | 64 | **5,602** | 上两桶合计 40,082 |
| 真 SSXZ 自研 | 364 | **55,769** | — |
| **合计** | **505** | **63,166** | 189,897 |

修正后最贵的不再是那几个（`SettingsView.vue` 真实 `ours` = **654** 行，
`GroupsView.vue` = **85** 行，两个都掉出前十）。51 个导入文件里最贵的：
`gateway_tool_rewrite.go` 186 · `channel_monitor_checker.go` 182 · `payment_order.go` 121 ·
`auth_oauth_email_flow.go` 114 · `payment_webhook_handler.go` 103 · `MonitorFormDialog.vue` 103。

⚠️ **「导入后没再改过」≠「纯上游副本」**：那次同步是**在同一个提交里**边导入边适配的，
所以"导入后 diff 为空"的文件里仍然藏着真定制。**唯一可靠的纯净判据是
blob 与上游 tag 字节一致**（实测 77 个里只有 26 个是）。

**独立佐证血脉**：51 个导入文件的最近基线 **全部 = v0.1.136**，
与 §4 血脉峰值法（v0.1.136，38.2%）用**完全不同的方法**得到同一个答案。

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

### ⭐ 破坏性闸门已关（这条已定案，2026-08-07 只读 SELECT 实测）

U2 的 138 个 SQL migration **全部已在 `schema_migrations` 里**（未记录 = 0）→
**换主干不会跑任何一条 migration，4 个破坏性 DDL 一条都不触发。闸门关闭。**

库里现在的 Sora 实况（090 要删的东西基本都还在）：

| 090 要删的 | 现在库里 |
|---|---|
| `sora_generations` | **还在，8 条真数据** |
| `sora_accounts` | 还在，0 条 |
| `groups` 5 列 + `users` 2 列 + `usage_logs.media_type` | **8 个列全在** |
| `sora_tasks` | 不存在 —— 但**两条线都没有 migration 建过它**（`063` 只建了 `sora_generations`），所以它的缺席不是 090 跑过的证据 |

### ❌ 已撤回：「090 已记录但从未执行」（我下早了，2026-08-07 同日修正）

发现 **`backend/migrations/198_restore_sora_customizations.sql`（只在 U 上，P 和 U2 都没有）**，
文件头第一行就是：

```
-- Restore the production Sora customization after upstream migration 090 removes it.
```

→ 同一份证据有**两个都成立的历史**，不能只挑一个：

| | 历史 A：090 从未执行 | 历史 B：090 执行了，198 又建回来 |
|---|---|---|
| 8 个列还在 | ✅ 没被删过 | ✅ 198 用 `ADD COLUMN IF NOT EXISTS` 加回 |
| `sora_accounts` 0 行 | ✅ 一直没配账号 | ✅ 被 drop 后 198 重建成空表 |
| `sora_generations` 8 行 | ✅ 正常累积 | ✅ 重建后新写入的 |

**证据现在明显偏向 B**：P 的 checksum 白名单里 `001_init.sql`/`131`/`132`/`133` 四条，
`acceptedDBChecksum` 记的是 **U2（上游）那一侧的哈希** → 说明**上游那一版文件真的在这个库上跑过**，
而 090 就在同一批里。B 还不需要任何人工干预就能自洽（runner 按序跑：090 删 → 198 建回）。

**判据（已验证可用，单次 `information_schema` 查询定案）**：

| | `sora_accounts` 的列 |
|---|---|
| P 的 `046` 形状 | `account_id, access_token(NOT NULL), refresh_token(NOT NULL), session_token, created_at, updated_at` |
| U 的 `198` 形状 | `account_id, credentials(JSONB), extra(JSONB), created_at, updated_at` |

两者在 `access_token` / `credentials` 上**互斥** → 查到哪个就是哪段历史。

⚠️ **`sora_generations` 不能当判据**（我一度想用，已验证不行）：P 的 `063` 本来就有
`api_key_id`/`media_urls`/`s3_object_keys`/`completed_at`，两种历史下形状一样。

**不管是 A 还是 B，闸门都是关的**（已记录 → runner 跳过 → 换主干不跑 DDL）。
但如果是 B，多一条后果：**Sora schema 是靠 198 活着的，而 U2 有 090、没有 198。**
→ 要留 Sora 就必须把 `198` 一起搬到 U2，否则将来在干净库上重建会重新删掉。

⚠️ **通用红线：「migration 已记录」≠「库已经是这条 migration 之后的样子」。**
runner 按完整文件名跳过，只要那一行在，DDL 到底生效没有无从得知。
凡是依赖"某个 schema 变更已生效"的判断，必须直接查 `information_schema`，
**不能查 `schema_migrations`**。

⚠️ **反向红线：绝不能删 `schema_migrations` 里 090 那一行，也不能强制重跑 migration。**
那 8 条 `sora_generations` 数据和 8 个列现在是靠"记录挡着"才活下来的。

### ⭐ checksum 闸门：已测，是关的（差点漏掉的第三个换主干闸门）

`migrations_runner.go:187-212` —— runner 对**每个已记录的 migration** 重算
`sha256(TrimSpace(文件内容))` 并与库里的 `checksum` 比，**不一致就拒绝启动**
（不是警告，是 `return fmt.Errorf`）。只有硬编码白名单
`migrationChecksumCompatibilityRules` 能豁免，且要求文件哈希和库哈希**同时**命中。

→ 所以"同名文件两条线内容不同"是一个**独立于 migration 记录的启动级闸门**。实测：

| 比对 | 同名 | checksum 不同 |
|---|---|---|
| **U vs U2** | 239 | **0** ← 决定性：U2 的文件与库里（U 跑出来的）记录天然一致 |
| P vs U2 | 101 | 4（`001_init`/`131`/`132`/`133`）——且**这 4 个正好全在 P 的白名单里** |

**结论：U2 不会因 checksum 拒绝启动。** P 之所以需要那 4 条白名单，正是因为库里存的是
上游那一版的哈希——这反过来成了"上游 migration 真跑过"的证据。

方法自检（必须做）：我重算的 P `001_init.sql` = `17d187d5de98…87fe`，与白名单里硬编码的
`fileChecksum` **完全一致** → 证明我算的就是 runner 算的那个东西。

migration 条目数也全部对上：同名 103 = 101 `.sql` + 2 `.go`；
U2 独有 142 = 138 `.sql` + 4 `.go`（与 Codex 报的 138 不矛盾）；
U 比 U2 多的 19 个 `.sql`（`186`–`204`）**正好等于 B8a 里那 19 个 migration**。

**计费风险已排除**（曾担心 U2 不写这些列会 INSERT 失败，实测不会）：
`usage_logs.media_type` 是 `VARCHAR(16)` **可空无默认**，`users`/`groups` 上的 sora 列是
`NOT NULL DEFAULT 0` → U2 的 INSERT 不提这些列也能过。

⚠️ **`/api/v1/sora/models` 返回 401 只证明二进制里有路由，不证明 Sora 表还在**
（鉴权中间件在 DB 访问之前就返回了）。已改用只读 SELECT 定案。

⚠️ **上文那条「切 U 会跑 0 个 migration」不能外推到 U2**——但 U2 已单独测过，结论相同。
U2 比 P 多 **142** 个条目 = 138 个 `.sql` + 4 个 `.go` 回归测试
（`auth_identity_payment_migrations_regression_test.go` 等），与 Codex 报的 138 不矛盾。
带破坏性 DDL 的 4 个：`090_drop_sora.sql` · `127_drop_channel_monitor_deleted_at.sql` ·
`136_remove_ops_retry_replay.sql`（DROP TABLE+TRUNCATE）· `180_audit_logs.sql`（TRUNCATE）。

### ⚠️ 账本盲区二（B8）：历史盲区记录（含已纠正的误判）

账本基数 1381 是拿 **fork / P / U2 三棵树**算的，**没有 U**。所以任何"只在 U 上做过、
P 没有"的定制，结构上不可能出现在 B1–B7 里。已实测漏掉的：

| B8 记录 | 实测 |
|---|---|
| Reseller 经销商代码 | U **31 个**文件；U2 **0**、P **0** |
| Reseller migration `200`/`201` | 生产库**已记录**、表和数据都在；U2 有 **0 个** reseller migration |
| **已作废：曾误判为只活在 U** — `channel_id=NULL` 计费修复 | **P 已含 `8413e2149`**（本轮已用 `merge-base --is-ancestor` 确认它是已部署提交 `c346c69e3` 的祖先）；缺口在 172，方向固定为 **P→172** |

→ **账本总盘已对齐为 1023：B1–B7 共 907，另加 B8a 的 116。**

**B8a 已量出 = 116 个**（用「存在性」测法：**U 有 + U2 无 + P 无**，零误判，
不用去减上游那 3134 个提交——因为上游改过的文件必然在 U2 里存在，天然被排除掉）：

| 目录 | 数 |
|---|---|
| `backend/internal/service` | 36 |
| `backend/migrations` | **19**（`186`–`204`）|
| `backend/internal/handler/admin` | 11 |
| `backend/internal/handler` | 11 |
| `frontend/src/views/reseller` | 6 |
| `frontend/src/views/admin/reseller` | 5 |
| `backend/scripts` | 4 |
| 其余（middleware/repository/components…）| 24 |

含测试 153，剔测试 + `ent` 后 **116**。

锚点自检的坑：**reseller 31 个里有 10 个是测试**（`reseller_repo_test.go`、
`__tests__/AdminAgents.spec.ts` 等），所以剔测试后是 **21**，不是 31。
21 个全部 `U2=0|P=0` → B8a 管线正确。**别拿 31 当锚点**。

**→ 账本总盘：907（B1–B7）+ 116（B8a）= 1023，需动手 845 + 116 = 961。**

B8b（U/U2/P 三方都有、但 U 与另两个都不同）**先不量**：那一桶必然混进上游改动，
且价值低于先把 B8a 这 116 个（reseller 全在里面）落地。

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
3. ⛔ **禁止用「必须重测」「归为待验」代替一次实际检查。**（2026-08-07 用户当场纠正）
   我写过「`migrations_runner.go` 被改了 → 换 172 必须重测 checksum 闸」和
   「微信支付读哪个 config key → 待验」。两条**都是 `git diff` / 读代码 5 分钟能定案的**，
   实际答案还都是"不用重测 / 已同源"。**先查再下结论，查不了才写待验，并写明为什么查不了。**
   同一次还踩了**用文件数虚报改动量**：172 我说"208 文件"，真源码只有 **122 个**，
   其余是 66 个测试 + 9 个 `ent/` 生成代码 + README/赞助商图片。
   **报改动量必须按类别拆，并且剔掉测试和生成代码。**

### 优先项（客户正在受影响，与主干迁移解耦、可先做）

1. **Reseller（已完成）** —— 已随 r2 上线，当前已部署提交为 `c346c69e3`；不再列为 P 的待修项。
2. **`channel_id=NULL` 计费修复（P 已完成，172 待补）** —— `8413e2149` 已确认是
   已部署提交 `c346c69e3` 的祖先。缺口在 172，方向固定为 **P→172**；后续只重放
   `usageChannelMappingForAPIKey` / `channelMappingResolver` / `GroupIDForUsage` 三个符号的语义，
   **不要整文件覆盖**。

- **唯一主干**：任何时刻只有一条线可以编译部署。**必须先立**——
  否则合过的东西会被另一条线的部署第三次顶掉
- 每次部署必须记三样：**上游底座 tag + 我们的 HEAD + 二进制 MD5**（`DEPLOYED.md`）
- 上游安全补丁不会自动到我们手上 → 必须定期看上游 release，有则单独挑
- 截至 2026-08-07 晚，本地可见最新上游 tag 是 `v0.1.172` = `155c49496`（**没有 173**），
  `upstream/main` = `68d8f122e`。**v0.1.172 含账号接管 0day 修复，见 §5.5。**
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
