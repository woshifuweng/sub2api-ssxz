# 生产真相记录（DEPLOYED）

> 这份文件只回答一个问题：**生产现在跑的是哪条代码线。**
> 它是唯一权威。与 `HANDOFF.md`、memory、`VERSION` 文件冲突时**以本文件为准**；
> 本文件与生产实测冲突时**以实测为准**，并立刻回来改这份文件。
> 最后核验：2026-08-07 部署后（核验方法见文末，任何人可自行复跑）

## 先跑这一条，别做考古

```bash
curl -s https://api.ssxzapi.com/api/v1/settings/public | grep -o '"version":"[^"]*"'
```

2026-08-07 起生产二进制带 ldflags 版本戳，这条命令直接返回代码线身份：

```
"version":"0.1.3-ssxz.20260807"
```

`0.1.3` = 我们 fork 自己的发布计数器（与上游 `0.1.17x` 是两套编号，不可比大小）；
`-ssxz.<日期>` = 构建日期戳，由构建命令注入，**认这个，不认 VERSION 文件**。

下面的"三条指纹考古法"是版本戳失效时（有人漏加 ldflags）的备用手段，平时不需要。

---

## 结论

生产跑的是 **`codex/fix-client-brand-announcements` 线**——即本地 HEAD `2202c5b60` 所在那条。

- **不是** v0.1.165 线
- **不是** 0.1.17x 线
- 这条线**没有**合入 v0.1.165 及以后的任何上游代码

真源码路径：`.codex-work/fix-client-brand-announcements2/`
（主目录 `backend/` 是空壳，0 个 `.go` 文件，往那里 grep 会静默假阴性）

---

## 仓库里同时存在三条线——这是所有混乱的唯一来源

| 线 | tip | 上游底座 | VERSION 文件 | 状态 |
|---|---|---|---|---|
| `codex/fix-client-brand-announcements` | `2202c5b60` (08-07) | 无 v0.1.165+ | `0.1.3` | **✅ 生产在跑** |
| `upgrade/v0.1.165` 后代 | `5e3aa5562` (08-03) | v0.1.165 | `0.1.164` | 曾上线，已被顶掉 |
| `fork/upgrade/v0.1.169` | `9e9440e35` (08-05) | v0.1.171 | `0.1.170` | 构建过，从未上线 |

三条线彼此**都不是**对方的祖先。后两条之间 backend 差异：**4720 文件 / +485519 / −56359**。
这不是小步升级，是换底座。

---

## 判定依据（三条独立指纹，结论一致）

**1. 版本字符串（2026-08-07 起已升级为血脉戳，这条现在是主判据）**
生产 `/api/v1/settings/public` → `0.1.3-ssxz.20260807`。
`-ssxz.<日期>` 后缀由构建时 `-ldflags -X main.Version=` 注入，**只有本线的部署流程会打这个戳**。
（历史参考：升级前生产报 `0.1.3`，165 线 VERSION 是 `0.1.164`，17x 线是 `0.1.170`。）

**2. CSP frame-src 组成（最硬的一条）**
生产实际下发（2026-08-07 起末尾多了 `'self'`，是本线的作图页修复，见部署记录）：
```
frame-src https://challenges.cloudflare.com https://js.stripe.com https://hooks.stripe.com https://pay.ldxp.cn 'self'
```
- 生产线：`js.stripe` + `hooks.stripe` 写在 `DefaultCSPPolicy` 里，`pay.ldxp.cn` 由 `ChainDianShopDomain` 动态注入 → **完全吻合**
- 165 线：`*.stripe.com` + `checkout.airwallex.com`，且 `ChainDianShopDomain` **0 命中** → 拼不出这个头
- 17x 线：还多 `turing.captcha.qcloud.com` → 拼不出这个头

`pay.ldxp.cn`（链动小铺）只存在于生产线。这一条单独就足以定案。

**3. 路由面（401=存在，404=不存在）**
```
/api/v1/admin/users                     401   两条线都有（控制组）
/api/v1/admin/channel-monitors          401   生产线有
/api/v1/admin/channel-monitor-templates 401   生产线有
/api/v1/admin/audit-logs                404   ← 165/17x 线无条件注册，若在那边必然 401
/api/v1/admin/prompt-audit/config       404
/api/v1/admin/content-moderation/config 404
/api/v1/admin/agents                    404   ← 见下方回归风险
```

---

（原"待确认的回归风险"一节已用真实路径复验完毕，结论移到文末
「已确认的功能回归：经销商」。此处不再保留推测版本，避免同一文件出现两种口径。）

---

## 不可信的信号（别再用它们判断版本）

| 信号 | 为什么不可信 |
|---|---|
| `VERSION` 文件 | fork 自己的发布计数器，四月重新起数，`release.yml` 会自动覆写。`0.1.3` 与上游 `0.1.171` 是两套编号，撞在同一字段 |
| `git merge-base` / tag 祖先 | squash-merge 抹掉祖先关系。**测不出祖先 ≠ 没合过代码** |
| `git describe` | HEAD 无任何可达 tag，永远答不出来 |
| `HANDOFF.md` 的生产 commit | 2026-07-26 写的 `21c279cff`，已过期（那是 165 线） |
| 主目录 `backend/` 下 grep | 0 个 `.go` 文件，永远返回"没找到" |

**唯一可信的判定方法：向生产发请求。** 代码树只能告诉你"某条线有什么"，不能告诉你"生产是哪条线"。

---

## 自行复验（30 秒，随时可跑）

```bash
# 0. 一条命令跑完全部核验（推荐）
bash scripts/verify-deployed.sh

# 1. 生产版本 —— 含 "-ssxz." 后缀即本线，且后缀日期就是最后一次部署日
curl -s https://api.ssxzapi.com/api/v1/settings/public | grep -o '"version":"[^"]*"'
#   期望: "version":"0.1.3-ssxz.20260807"
#   若无 -ssxz 后缀 → 部署时漏了 ldflags，或换了别的线，本文件需重新核验

# 2. 生产 CSP frame-src（看有没有 pay.ldxp.cn → 生产线特征）
curl -sI https://api.ssxzapi.com/app/image | tr ';' '\n' | grep frame-src

# 3. 路由指纹：401=存在 404=不存在
for p in /api/v1/admin/users /api/v1/admin/audit-logs /api/v1/admin/agents; do
  echo "$p -> $(curl -s -o /dev/null -w '%{http_code}' https://api.ssxzapi.com$p)"
done
# users=401 且 audit-logs=404  →  生产在生产线（老底座）
# audit-logs=401               →  生产已升到 165/17x 线，本文件作废，请重写
```

---

## 根治措施（进度）

- [x] 建立本文件作为唯一权威记录
- [x] 根目录 `CLAUDE.md`：让每个新会话第一眼就读到这里
- [x] 主目录 `backend/README.md`：空壳目录自我声明，堵住假阴性 grep
- [x] **构建时注入血脉**（2026-08-07 已生效）：
      `-ldflags "-X main.Version=0.1.3-ssxz.<日期> -X main.Commit=<sha>"`
      生产实测 `/api/v1/settings/public` → `0.1.3-ssxz.20260807`
      **这一条是循环终结器**：从此"生产是哪条线"用一个 curl 就能答，不必再考古
- [x] 版本号改 `<基线>-ssxz.<n>` 格式，编号自带血脉，不再与上游撞车
- [x] `scripts/verify-deployed.sh`：一条命令跑完全部指纹核验
- [ ] 决策：三条线并作一条（见下）
- [x] 确认 Reseller 确实从生产消失（见下节），已在 `wip/reseller-port` 分支上移植中
- [ ] **`dist/assets/*` 不进 git**（`.gitignore:105`）——「我做的东西消失了」的**结构性根因**。
      只跟踪 `index.html`、忽略全部 JS/CSS，等于"生产在跑哪份前端"既查不到也复现不出。
      最小修法：部署脚本里把 `npm run build` 和 `go build -tags embed` **绑成一步**，
      让二进制永远吃到当前源码的前端。本次部署先不动它，避免一次改两件事。

## 两个会骗你的部署陷阱（各踩过一次）

**1. systemd 覆盖 config.yaml**（2026-08-07 上游 401 故障真因）
配置文件里已经写对了域名，服务仍然用旧白名单——因为 systemd unit 里的环境变量
覆盖了 config.yaml。**改完配置只重启不够，得确认 unit 里没有同名覆盖项。**

另附一个当时差点踩的坑：`security.url_allowlist.upstream_hosts` 在 YAML 里写成显式列表会
**整体替换** `viper.SetDefault` 的 8 个默认域，不是追加。只补上出问题的两个域，
就会把 OpenAI/Anthropic 官方直连全部打掉——正是「修了这边坏那边」。补的时候必须
把默认 8 个一起写全。
（另：`allow_insecure_http` 被启动闸硬锁 false，所以 `http://` 的上游端点在
`validator.go:40` 的 scheme 检查处就被拒，**根本走不到白名单**，加域名也救不了。）

**2. `applyExecutableMiddlewares` 没有 default 分支**（`internal/server/executable_routes.go:175`）
路由注册时给一个不存在的中间件 tag（例如 `audit_log`），switch 会**静默忽略**：
不报错、不 panic、也不拦请求，函数照常返回 true。
→ 移植 Reseller 时**直接省掉 `audit_log` tag**，且不要移植
`gnet_residue_audit_test.go` 里那条断言（它断言的是"字符串在"，而行为并不在，等于断言一句假话）。

## 构建命令

`verify-deployed.sh` 第 60 行指向本段。**顺序不能颠倒**：`-tags embed` 会把当前磁盘上的
`backend/internal/web/dist` 打进二进制，所以前端必须先构建，否则会把旧前端打回去
（2026-08-07 前端"回退"事件就是这么来的）。

```bash
# 1. 先构建前端（vite 的 outDir 直接指向 embed 目录，无需再手动 copy）
#    包管理器是 pnpm，不是 npm（a11a0f289 已迁移）。用 npm 会静默换掉依赖树，见下方第 1 条。
cd frontend && pnpm install --frozen-lockfile && pnpm run build

# 2. 红线检查：入口 JS 绝不能带 ?v= query（两个 URL = 两份模块 = 双 mount 黑屏）
grep -c '?v=' ../backend/internal/web/dist/index.html   # 必须是 0

# 3. 再交叉编译后端
cd ../backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags embed -trimpath \
  -ldflags "-s -w \
    -X main.Version=0.1.3-ssxz.$(date -u +%Y%m%d) \
    -X main.Commit=$(git rev-parse --short HEAD) \
    -X main.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o sub2api_linux ./cmd/server
```

必须知道的四条：

- **只能用 pnpm，不能用 npm**。`frontend/pnpm-lock.yaml`（lockfileVersion 9.0，已跟踪）是唯一
  依赖真源。`npm ci` 会直接报 `can only install with an existing package-lock.json`（2026-08-07
  部署卡在这里）；而 `npm install` **更糟**——`package.json:62` 的 4 个 pin 写在 `pnpm.overrides`
  里，npm 不读这个键，会按 semver 范围重解出另一棵依赖树，vendor chunk 哈希跟着变，
  等于绕过了本文件第 5 项的资源基线核验。lockfile 里那几个版本号（`lodash@4.18.1`、
  `js-cookie@3.0.8`）看着像不存在，实际已在公共 registry 发布过，`--frozen-lockfile` 在
  干净服务器上能拉到，不用担心。
  - ⚠️ **未解决**：2026-08-07 在本机执行 `pnpm install --frozen-lockfile` 直接崩溃，
    退出码 `-1073740791`（`0xC0000409` STATUS_STACK_BUFFER_OVERRUN，Windows 码）。
    该次部署因产物已在磁盘上而绕过，**但下次动前端必然再撞**。未验证的怀疑（按可能性排序）：
    仓库里 **107,667 个被误提交的 `.pnpm-store/` 文件** / Windows 路径长度 / store 索引损坏。
    崩溃**没有**损坏 `node_modules`（36 个顶层包 + `.pnpm` 完好），所以当时仍能直接构建。
- **`-X main.Version` 不可省**。`backend/cmd/server/VERSION` 里只写 `0.1.3`，血脉戳完全靠
  ldflags 注入（`main.go:23` 的 `//go:embed VERSION` 只是兜底）。漏掉这个 flag，生产
  就变回裸 `0.1.3`，`verify-deployed.sh` 第 55 行会失败，考古循环重新开始。
- **不要传 `-X main.BuildType=release`**（`deploy/Dockerfile` 里那份是给 CI 用的）。
  默认值 `source` 才是生产当前状态。
- **后台"受控发布"那块灰按钮跟构建参数无关**，它由 `VersionBadge.vue` 的
  `runtimeActionsEnabled` prop 决定（默认 false），不是 `BuildType`。

## 部署记录（新的写在最上面）

### 2026-08-07（待部署）— 前端重建 + 兑换码去验证码 + 版本误报修复

**为什么这次一定要重建前端**：这是「我做的 UI 组件消失了」的真正原因，不是代码被别的线顶掉。

- `backend/internal/web/dist/assets/*` 在 `.gitignore:105` 里被忽略，**只有 `dist/index.html` 被跟踪**。
  于是"生产在跑哪份前端"在 git 里查不到，也复现不出来。
- 生产实际跑的是 **08-07 00:44–00:53** 那次构建。commit `45c1978ed`（02:12）只动了
  `index.html`（+17 行），**没有重新生成任何 JS chunk**。
- 落在构建窗口之后、从未上线的前端 commit：`03a239ff6`(00:55)、
  `adf8e487d`(00:55 重建用户模型定价表页面)、`d3c3f7495`(00:56)，外加 13:46 两个未提交的作图页文件。
  00:34–00:35 那批赶上了窗口——**所以只有一部分组件看起来"没了"**。
- 旁证：被跟踪的 `dist/index.html` 里 `<title>` 还是 `Sub2API - AI API Gateway`，
  本次重建产出 `SSXZ AI` → 该产物是**早于我们改品牌之前**构建的。
  （运行时 `injectSiteTitle` 会用 DB 里的站名覆盖 `<title>`，所以用户没看出来。）

本次一并上线：

| 内容 | 文件 |
|---|---|
| 前端重建（3 个未上线 commit + 作图页内嵌） | `dist/*`，入口 `index-CTBPrhn_.js` → `index-Bn3BFivZ.js` |
| 版本误报"有新版本 v0.1.2"修复 | `internal/service/update_service.go` |
| 兑换码取消 Turnstile 人机验证 | `internal/handler/redeem_handler.go`、`frontend/src/views/user/RedeemView.vue` |

**版本误报是有牙齿的 bug，不只是显示问题**：`parseVersion("0.1.3-ssxz.20260807")`
按 `.` 切分后第三段是 `"3-ssxz"`，`strconv.Atoi` 失败**静默留 0** → `0.1.3` 被读成 `0.1.0`
→ 判定 `< 0.1.2` → 报"有新版本"。点「立即更新」会下载 fork 的 0.1.2 覆盖生产，
**装上一个更旧的版本**——即一次真正的回退。修法是先剥掉 `-`/`+` 后缀再切分。
（当前被受控发布拦着，按钮是灰的，所以没真出事。）

**兑换码去验证码**：不是回归，是一个**一直没落地的决定**——`-S"turnstile"` 全分支只查到
07-18 那次**新增**，从来没有删除。后端闸删在 `RedeemGateway`（gin 与 gnet 两条路都汇到这里，
改一处即可），前端删掉 widget 与提交前的 token 检查。
安全兜底仍在三重：必须登录、失败 5 次锁 30 分钟（Redis，按用户）、兑换码是 UUIDv4（122 位熵）。
**登录/注册/找回密码/支付的验证码不受影响**（全局 DB 开关 `SettingKeyTurnstileEnabled` 未动
——正因为它是全局共用的，所以不能用关开关的方式解决）。

新增回归测试 `internal/handler/redeem_handler_no_turnstile_test.go`：
即使 `Turnstile.Required = true` 且不带 token，兑换请求也必须穿透到 service。
**已做变异验证**：把删掉的闸原样加回去，该测试立刻 FAIL
（`TURNSTILE_NOT_CONFIGURED`，且 service 调用次数 0）——它拦得住"验证码又被加回来"。

构建与自检（本地，已全部通过）：

```
前端: pnpm install --frozen-lockfile && pnpm run build
      # vite.config.ts:65 outDir 直接指向 backend/internal/web/dist，无需拷贝
      # 注意：本次实际是拿已有 node_modules 直接跑 `npm run build`（npm 此处只当脚本运行器，
      #       不解析依赖，所以产物正确）。但**不要**由此推出"可以用 npm"——见「构建命令」段第 1 条。
后端: GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags embed -trimpath \
      -ldflags "-s -w -X main.Version=0.1.3-ssxz.20260807 -X main.Commit=e8ef9e645 -X main.Date=<UTC>"
```

- **实际上线二进制**：`sub2api_linux_e8ef9e645`，md5 `633825a54384d11572e8f6931ec87825`，93,737,122 bytes
  - commit 戳为**干净** `e8ef9e645`（Go buildinfo `vcs.revision` 无 modified 标记）
    → 线上二进制可反查到已推送的提交，这是本轮"查不出线上跑哪份代码"的解药
  - 二进制里 `-dirty` 命中 2 处，**均为无关库字符串**（`baseline and allow-dirty are mutually exclusive`），
    `e8ef9e645-dirty` 命中 0 —— 别被这 2 处误导成脏构建
- 备用（15:21 首次构建，内容等价但戳为 `48faffaae-dirty`，无法证明对应哪次提交）：
  md5 `b35e72ddca2543fc7d0b364411ac48dc`（更早一版 `c8229da332d332428ce48b9d94abd539`）
- ELF64 x86-64 已核（`7f 45 4c 46 02 01 01` + machine `0x3e`），不是 Windows 产物
- 体积 130MB → 93.7MB，**纯粹因为加了 `-s -w`**（与 `deploy/Dockerfile:69` 一致），不是内容缺失
- 二进制内含新入口 `index-Bn3BFivZ.js`(90 处)，**旧入口 0 处** → embed 确实吃到新前端
- 版本戳 `0.1.3-ssxz.20260807` 命中 1 → `verify-deployed.sh` 第 55 行会过
- 白名单默认域 `api.openai.com`/`api.anthropic.com`/Google **仍在**，启动闸字符串仍在
  → **本次没有推翻 Codex 的白名单修复**
- 测试：handler 包全量 `ok 25.845s`，service 版本比较 `ok`，前端 typecheck EXIT=0、RedeemView 11/11
- 入口 `?v=` 红线：新产物 0 处。且全构建链（vite.config/package.json/index.html/scripts/deploy）
  **均不注入** `?v=` → 确认 08-07 黑屏那次是手工加的，重建不会复现

**部署后必须做**：`bash scripts/verify-deployed.sh --update-baseline`
（前端资源哈希这次是有意变更，不更新基线的话第 5 项会一直 FAIL）

#### 第一次部署尝试：16:35 上线后被回滚

**5 项过 4 项，唯一失败项是别人家的中转站没容量，不是我们的代码。**

| 项 | 结果 |
|---|---|
| 上传 md5 `633825a5…` | 通过 |
| 版本 `0.1.3-ssxz.20260807` | 通过 |
| 入口 `index-Bn3BFivZ.js` | 通过（embed 确实吃到新前端） |
| 白名单环境变量仍在 | 通过 |
| Kedaya `gpt-5.5` | 通过 |
| **Molifang Claude** | **约 45 秒未返回** → 触发回滚 |

已回滚至 `/opt/sub2api/backups/sub2api-pre-redeem-captcha-20260807-163527`；
当前生产 md5 `2bf007115a3706efb406c4994b11cf3d`，active/running、NRestarts=0、ExecMainStatus=0。
**门禁与基线更新均未执行。**

归因证据（全部指向「与本轮改动无关」）：

- 本轮 diff 仅 8 个文件（后端 2 个非测试），对
  `anthropic|claude|gateway|upstream|timeout|httpclient|retry|scheduler` 的 grep **全部 0 命中**
- 两处后端改动都是封闭的：`redeem_handler.go` 只动 `RedeemGateway` 内部；
  `parseVersion` 仅被 `compareVersions` 调用（`update_service.go:298` / `:506`，只算 `HasUpdate`
  一个字段），**无后台 goroutine、无启动期拉取** → 碰不到任何请求路径
- **同一轮 Kedaya `gpt-5.5` 通过** → 网关转发与调度本身正常。若转发被改坏，Kedaya 会一起挂
- 账号 38(`Kiro高缓-MoLiFang`) 是**第三方转售中转**。先例：`CLAUDE_TO_CODEX.md:2085` 记 37/38
  双双 `All available accounts exhausted`，当时结论即「两家 Kiro 转接号当前无容量，非本轮修复失败」；
  `:1534` 另有一次判为上游偶发
- 45 秒无返回是**超时**，不是白名单拦截会给的快速 400/403

**尚未证实**：没有在回滚后的旧二进制上复测 Molifang。所以「它本来就不通」目前是强推断，不是实测。

#### 归因已闭合（2026-08-07 复测，旧二进制 `2bf00711`）

复测结果：

| 项 | 结果 |
|---|---|
| 账号 32（CCMAX）Claude | **3/3 成功，约 3.4 秒** |
| 账号 38（Kiro-MoLiFang）Claude | **3/3 成功，约 3.4–6.5 秒** |
| Kedaya `gpt-5.5` | 成功，约 3.4 秒 |
| 真实网关请求 ×3 | 均 `401 Invalid API Key` → 无有效客户 Key，**不是 Claude 上游结果** |
| 账号 30/34/35 | `model_mapping` 里没有 Claude，无法测试 |

**决定性证据：`account_test_service.go` 不在本轮 diff 里**（`git diff --name-only 48faffaae..e8ef9e645`
对 `account_test` 命中 0）。即「后台账号测试」这条代码路径在新旧二进制中**逐字相同**。

由此两条结论：

1. **16:35 那次 45 秒无返回不可能由本轮改动造成。** 若当时用的是后台账号测试，跑的就是与今天
   完全相同的代码；今天同一路径 3/3 通、3.4 秒。
2. **那次回滚是无效动作。** 回滚前后该路径的代码一致，回滚不可能修复它。真实原因是上游瞬时抖动。

**更正我上一轮的判断**：我曾推断 Molifang「无容量」（依据 `CLAUDE_TO_CODEX.md:2085` 的历史先例）。
**这句是错的**——今天 6 次全通。当时属瞬时抖动，不是容量耗尽。历史先例不构成当次证据。

`401 Invalid API Key` 出现在**回滚后的旧二进制**上，因此与二进制无关，是那把 Key 本身无效
（未签发／已撤销／已过期）。**不是**新版本破坏了鉴权。

删除 `internal/pkg/ip` import 的副作用也已排除：另有 11 个文件仍 import 它，包链未断。
（该包内有 1 个 `init()`，但因仍被 `gateway_handler.go` 等引入，初始化照常执行。）

结论：**判定 A 成立，可以重新部署。**

#### 遗留风险（与本次部署无关，单独修）

- **Claude 无备胎**：当前仅账号 32 / 38 的 `model_mapping` 含 Claude；30/34/35 没配。
  两个都挂就没有兜底。DB 里另有候选（StrategyHub、211b、官方等），但**尚未确认是否已加入
  客户实际调度组**。改 `model_mapping` 后必须重启才生效。
- **真实网关路径仍未验证**：后台账号测试只证明「我们的服务器连得上上游」，
  **不等于「客户真能用上 Claude」**——它绕开了分组过滤与调度。需要一把有效客户 Key 才能验。

#### 坑：`verify-deployed.sh` 不含任何真实上游请求

脚本里 5 处 `curl --max-time 15`（第 31/51/98/127/155 行）**打的全是我们自己的域名**——
路由面、版本、CSP、前端资源、`/image/`。Molifang 那项是手加的额外验证，**不属于正式门禁**。

后果本轮已真实发生：**一次转售商容量故障，拦下了一个正式门禁本会放行的部署。**
以后真实上游冒烟测试记为「观察项」，失败先做对照组复测，不要直接触发回滚。

#### 门禁第 5 项在重新部署时**必定 FAIL**，且这是预期的

基线 `scripts/deployed-baseline.txt` 是 13:28 由脚本**自动初始化**的（不是人工挑选），
存的是**旧入口**。重新部署后第 5 项必报「前端资源与基线不一致」，预期 diff **恰好一行**：

```
- /assets/index-CTBPrhn_.js     ← 旧
+ /assets/index-Bn3BFivZ.js     ← 新
```

判读规则（**只有一行、且只有入口 chunk 变**才算正常）：

- 其余 5 个资源（`index-9_uviuae.css`、`vendor-vue-CdtVZIzQ`、`vendor-i18n-BqIlsuWk`、
  `vendor-misc-Biz7uIAo`、`vendor-misc-DB0Q8XAf.css`）**必须一字不变**。已逐一核对
  `48faffaae` 与 `e8ef9e645` 两版被跟踪的 `dist/index.html`，确实只有入口不同
  （入口变化来自 `RedeemView.vue` + `AppImageWorkbenchView.vue` 两个源码改动，属应用代码 chunk）
- **vendor chunk 若也变了 = 危险信号**：说明构建重新解析了依赖，绕过了 `package.json` 里
  `pnpm.overrides` 的四个 pin（`form-data 4.0.6` / `js-cookie 3.0.8` / `lodash 4.18.1` /
  `lodash-es 4.18.1`）。本轮 vendor 全部未变 → **pin 守住了**，那次拿 `npm run build`
  当脚本运行器没有造成依赖漂移（这是当时留下的疑点，现已排除）
- **入口若仍是 `CTBPrhn_` = embed 没吃到新前端**，那才是第 5 项本来要防的真故障

确认 diff 恰好是上面那一行之后，再跑 `--update-baseline`。

### 2026-08-07 — 作图页 CSP 修复 + 首次版本戳
- 版本：`0.1.3` → **`0.1.3-ssxz.20260807`**（首次用 ldflags 注入，生产已实测生效）
- 改动：`enhanceCSPPolicy()` 补 `'self'` 进 `frame-src`，新增 `directiveAllowsSelf()` 辅助函数
- 文件：`backend/internal/server/middleware/security_headers.go`（+29 行），测试 +43 行
- 根因：显式列出的 `frame-src` **不继承** `default-src 'self'`，所以同源 `/image/` 被拒，
  表现为"被屏蔽"的破页图标，且**无痕窗口同样复现**（CSP 是服务器下发，与缓存无关）
- 部署后核验（全部通过）：
  - `frame-src` 末尾出现 `'self'`，三方域 `challenges.cloudflare` / `js.stripe` / `hooks.stripe` / `pay.ldxp.cn` **全部仍在**
  - `frame-ancestors 'none'` **未被放宽**（外站仍无法内嵌本站）
  - 前端 6 个资源哈希与部署前**逐一相同**，未回退前端
  - 路由面 4 项全 401，无路由消失
  - `/image/` 自身 200，且不带 XFO/CSP（可被内嵌）
- 旧二进制备份：`/opt/sub2api/backups/sub2api-pre-csp-version-20260807-132113`

---

## 待决策：三条线怎么收

1. **以生产线为正本**，把 165/17x 线归档 → 放弃上游 6 个月演进，但零风险
2. **升到 17x 线**，把生产线的 ssxz 改造迁移过去 → 4720 文件差异，需专门排期与回滚预案
3. 维持三线 → 不可接受，本文件存在就是因为这个状态已经在造成损失

## 已确认的功能回归（二）：上游 v0.1.171 升级 + channel_id 计费修复

**先说清楚，避免下个会话重新考古**：171 **合过，是真的**，不是记错。
`b9545e8dd Merge upstream v0.1.171 into upgrade/v0.1.169`（2026-08-05 07:13，111 提交 /
138 冲突全解），当天上线，migration 192/193 于 16:21:44 在生产库执行过。
memory `ssxz-aug05-session-findings` 那条记录没有写错——**它只是过期了**。

现在生产上一点不剩。三条独立判据（2026-08-07 复核，全部只读）：

| 判据 | 结果 |
|---|---|
| `b9545e8dd` 是生产线 tip 的祖先吗 | **不是** |
| `v0.1.169..v0.1.171` 新增 81 个文件，生产线里有几个 | **1 个**（`AccountBulkActionsBar.spec.ts`，撞名，非同源），其余 80 全无 |
| migration 最大编号 | 生产线 `138` / 171 线 `204`；`192`/`193` 在生产线**不存在** |
| 实测生产 `/settings/public` 的 captcha 字段 | 171 线有 11 个 `tencent_*`/`aliyun_*`，生产返回 **0 个**，只有 `turnstile_enabled` |
| 待部署二进制里 `profit_control_enabled` / `tencent_captcha_enabled` | **各 0 次** |

**机制与经销商完全相同**，这是同一个病的第二个受害者：
171 合并在 `upgrade/v0.1.169` 线，经销商在 `upgrade/v0.1.165` 线，生产现在跑第三条
`codex/fix-client-brand-announcements`——它不是前两条的后代。8/6 起每次从本线构建部署，
就把那两条的成果一起盖掉。**不是谁删了代码，是另一条分支的构建覆盖了它。**

### 顺带丢失：channel_id=NULL 计费归属修复（优先级高于经销商）

`9e9440e35`（同在 169 线，8/5 上线过）。它不像经销商那样"整块消失"，
而是**同名文件内容不同**，所以只看文件是否存在会得出错误结论：
生产线里 `backend/internal/handler/api_key_group_routing.go` **存在**（8145 字节），
但里面不是那份实现。看符号才准：

| 符号 | 生产线 | 169 线 | 待部署二进制 |
|---|---|---|---|
| `usageChannelMappingForAPIKey` | 0 | 6 | **0** |
| `GroupIDForUsage` | 0 | 2 | **0** |
| `channelMappingResolver` | 0 | 1 | — |

原修复内容：账号调度后 `channelMapping` 未刷新导致 `ChannelID=0→NULL`；多分组 Key 只用第一个
GroupID。**代码层面已确认缺失（线里没有、二进制里没有）；线上使用记录当前是否真在写 NULL 尚未验证**
（需查库，非只读探测能覆盖）。

移植方式：**只搬那 3 个符号涉及的改动，禁止整文件覆盖**——169 线该文件带几百行无关漂移，
整文件搬会引入新回归（与经销商 affiliate 守卫同一教训）。

### 数据库比代码超前（低风险，但不是零）

192/193 跑过而代码只认到 `138`：

- `192_group_profit_control.sql` = `ALTER TABLE groups ADD COLUMN IF NOT EXISTS ... NOT NULL DEFAULT`，
  纯加列带默认值，老代码不碰它无影响。
- `193_...auth_cache_invalidation.sql` = `CREATE OR REPLACE FUNCTION
  enqueue_group_auth_cache_invalidation()`，**换掉了触发器函数体**。它引用的列 192 已建，
  不会报错，但这是"代码已回退、行为仍留在库里"的东西，回滚二进制**不会**把它回滚掉。
- `139–191` 之间约 50 个 migration **未逐个审计**。

### 未受影响（已复核仍在生产线）

- Luna `HasPrefix` 封禁：8 个文件命中
- 缓存 0.5x fallback：`billing_service.go` / `pricing_service.go` 均在

这解释了用户观察到的"只有一部分东西没了"——不同改动落在不同线上，命运不同。

## 已确认的功能回归：经销商（Reseller / 代理）

证据（2026-08-06 探测，全部只读）：

| 检查项 | 结果 |
|---|---|
| 上线该功能的 commit | `5e3aa5562`（2026-08-03，T5 全量上线） |
| 它是当前生产线 HEAD 的祖先吗 | **NO**（`v0.1.165` 是它的祖先，不是我们这条线） |
| 生产 `/api/v1/admin/reseller/agents` | **404**（真实路径，从 5e3aa5562 里取出，非猜测） |
| 生产 `/api/v1/admin/reseller/commission` | **404** |
| 对照组 `/api/v1/admin/users` | 401（路由存在，仅需鉴权） |
| 生产线代码里 `Reseller` handler | **0 个文件** |
| 生产线前端里 `reseller` | **0 个文件** |

结论：经销商功能（13 接口 + 6 页面）**当前不在生产上**。它曾于 8/3 在
`v0.1.165` 那条线上线，随后 8/4–8/6 从本线构建部署，把它顶掉了。
前后端一起消失，所以后台看不到入口，不是"点开报错"。

### 移植可行性评估（2026-08-07，只读分析）

结论：**可移植，比预期乐观。不是重写，是搬运 + 接线。**

规模：26 个非测试文件，**8782 行**（后端 3187 / 前端 5595），另有 5 个测试文件约 950 行。

| 关键风险点 | 实测结果 |
|---|---|
| 是否依赖 165 线独有基础设施 | **否**。三个后端文件的 internal import（`pkg/response`、`server/gatewayctx`、`server/middleware`、`service`、`pkg/errors`）**本线全部已有** |
| migration 编号是否冲突 | **不冲突**。经销商用 `200/201/202`，本线最大是 `138`，可直接叠加 |
| 数据库表是否还在 | **待查**。8/3 上线时 migration 应已跑过，若表还在则数据未丢，只是代码被顶掉 |

主要工作量在"接线"而非"写码"：路由注册、handler/service 依赖注入、前端路由与菜单项、
然后跑通编译与测试。

**先做这一步**：确认生产库里 `200/201/202` 是否已执行、相关表和数据是否还在。
若数据在，移植就是纯代码搬运；若表被回滚过，还要考虑数据补偿。

未决定前不要把它当"已上线功能"对外承诺。

### 2026-08-07 重新部署：redeem-captcha / 前端资源（A 判定）

结论：按 A 判定重新部署。16:35 的 Molifang 验证方法是后台“测试账号连接”，不是客户真实网关请求，没有使用客户 API Key。由于本轮改动未触及账号测试、HTTP client、超时、重试等路径，且旧二进制当天对账号 32/38 的 Claude 测试均为 3/3 成功，16:35 的挂起不能归因于本轮改动；Molifang 本次仅作为观察项，未因其结果自动回滚。

部署记录：

| 项目 | 结果 |
|---|---|
| 部署时间 | 2026-08-07 17:37:20 CST |
| 生产二进制 MD5 | `633825a54384d11572e8f6931ec87825` |
| 生产二进制大小 | `93,737,122 bytes` |
| 服务状态 | `active/running` |
| `NRestarts` / `ExecMainStatus` | `0 / 0` |
| 新入口资源 | `/assets/index-Bn3BFivZ.js` |
| 旧入口资源 | `/assets/index-CTBPrhn_.js`（已不再引用） |

A 判定依据：账号测试代码路径在本轮 diff 中未变；`timeout/httpclient/retry` 相关改动均为 0 命中；删除 `internal/pkg/ip` 的 import 不影响该包仍被其他文件引用及其初始化。旧二进制上账号 32/38 的 Claude 测试当日均为 3/3 成功，因此 Molifang 挂起属于观察项，不作为本次发布失败依据。

门禁结果：本机没有可执行 Linux/WSL 的 `bash` 环境，`bash scripts/verify-deployed.sh` 无法直接运行；已按脚本 0–6 项逐项执行等价只读检查。结果为：控制组 401、版本戳 `0.1.3-ssxz.20260807`、必需路由均 401、CSP 的 `frame-src` 保留 `self`/Cloudflare/Stripe/`pay.ldxp.cn`、`frame-ancestors 'none'`、`/image/ = 200`。资源基线更新前的唯一差异为：

```text
- /assets/index-CTBPrhn_.js
+ /assets/index-Bn3BFivZ.js
```

其余资源保持不变：`index-9_uviuae.css`、`vendor-i18n-BqIlsuWk.js`、`vendor-misc-Biz7uIAo.js`、`vendor-misc-DB0Q8XAf.css`、`vendor-vue-CdtVZIzQ.js`。确认差异符合预期后，已将 `scripts/deployed-baseline.txt` 中的入口从 `index-CTBPrhn_.js` 更新为 `index-Bn3BFivZ.js`。

上线后后台账号测试（新二进制）：

- `moli-ccmax-080`（账号 32，`claude-sonnet-5`）：3/3 成功，单次约 7.8–11.0 秒。
- `moli-kiro-006`（账号 38，`claude-sonnet-5`）：3/3 成功，单次约 7.8–8.0 秒。
- `kedaya-gptplus-004`（`gpt-5.5`）：1/1 成功，约 7.9 秒。

边界：后台账号测试证明账号直连测试链路和上游白名单检查通过，不等于客户真实 API Key 通过完整网关调度链路使用 Claude；Molifang 仍保留为观察项。

#### 门禁首次真实执行（2026-08-07，Git Bash）

Codex 侧无 Linux/WSL `bash`，只做了等价只读检查。事后由 Git Bash 真实执行
`bash scripts/verify-deployed.sh`：**退出码 0，7 组全绿**。这是本脚本建立以来第一次端到端跑通。

其中第 5 项意义最大：脚本自己 `curl` 生产、取回真实资源清单、与基线逐行 `diff` → PASS。
即 Codex 手写的那行基线**由实测背书**，不再是断言。第 3 项确认经销商与 `audit-logs`
仍为 404，与本文件记录的已知缺失状态一致。

**教训**：「等价检查」与「跑脚本」不可互相替代。等价检查靠人转述，无法验证基线文件本身
是否与生产一致——而基线正是唯一能发现「部署没生效」的那道闸。以后基线只允许由
`--update-baseline` 写入，不允许手改。

#### 观察：重启后延迟约翻倍，疑为冷启动（**假设，待温机复测**）

| | 旧二进制（已运行 1 小时+） | 新二进制（刚重启） |
|---|---|---|
| 账号 32 Claude | ~3.4 秒 | **7.8–11.0 秒** |
| 账号 38 Claude | ~3.4–6.5 秒 | **7.8–8.0 秒** |
| Kedaya `gpt-5.5` | ~3.4 秒 | **~7.9 秒** |

**全线一致翻倍，含互不相关的 Kedaya** → 不是 Claude 或某家上游的问题。
代码解释很弱：本轮 diff 对 `timeout|httpclient|retry` 全 0 命中，账号测试路径逐字相同。
最吻合的解释是**重启后冷启动**：连接池空、无复用 TLS 会话、DNS 未缓存。

**这条假设顺带解释了 16:35 那次 45 秒挂起**——当时同样是刚重启。45 秒即同一分布的长尾。
在此之前 45 秒无返回一直缺一个机制解释。

**运维规则（本轮已真实踩过一次）**：重启后第一轮上游冒烟测试偏慢属正常，
**不要据此判定回归、更不要自动回滚**。要判定回归，等服务温机后复测同账号同方法。

**待办**：温机后（建议部署 1 小时以上）复测账号 32/38 + Kedaya。
回到 ~3.4 秒 → 冷启动确认，本条结案；仍在 ~8 秒 → 存在真实延迟回归，需查。
