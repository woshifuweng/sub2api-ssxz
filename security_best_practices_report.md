# SSXZ API 近期整体安全检查报告

检查日期：2026-08-23  
范围：公网入口、TLS/响应头、未登录接口、生产主机监听端口与容器、近 7 天日志、当前线上二进制与源码对应关系、认证/越权/SSRF/支付/文件处理、前后端依赖及上游版本。  
执行方式：先完成只读检查，再按用户授权实施最小加固并逐项验证。全程未修改生产数据库、migration、OAuth 开关或 `channel_model_pricing`。

## 一、结论

**暂未发现网站已经被攻破、数据库被篡改或攻击者拿到免费 Token 的证据。**

截至 2026-08-24，本轮确认的直接高风险入口和可达运行时漏洞均已处置；安全加固版本已上线，完整测试与生产验收通过。剩余工作主要是上游 177–179 的常规合并评估和更长期的安全运营，不是当前正在被利用的已知漏洞。

但不能据此认定“完全安全”。本轮确认了 **2 项高风险、4 项中风险、3 项低风险/观察项**。最高优先级是：

1. 两个遗留测试容器把 PostgreSQL/Redis 发布到所有网卡；Redis 没有密码，PostgreSQL 凭据强度也不足。云安全组是否挡住它们本轮无法可靠证明。
2. 线上二进制使用 Go 1.26.5；官方工具确认有 6 个代码可达漏洞，均在 Go 1.26.6 修复。

近期确有持续的互联网扫描和大量上游 502/503，但日志中未见成功利用、程序崩溃、OOM、migration 破坏或敏感密钥泄露迹象。

## 二、风险汇总

| ID | 等级 | 发现 | 当前判断 |
|---|---|---|---|
| SEC-001 | 高 | 遗留 PostgreSQL/Redis 测试容器发布到 `0.0.0.0:32768/32769` | **已处置**：容器和匿名卷已删除，端口不再监听 |
| SEC-002 | 高 | Go 1.26.5 存在 6 个可达漏洞 | **已处置**：Go 1.26.6 重建，`govulncheck` 可达漏洞为 0 |
| SEC-003 | 中 | 注册开放、邮箱验证关闭、风控关闭 | **已加固**：保留 Turnstile，并增加全局/域名/邮箱三级限速；配置开关未擅自改变 |
| SEC-004 | 中 | 非 JSON 上游错误正文只截断、不脱敏 | **已处置**：持久化前增加敏感正文脱敏和回归测试 |
| SEC-005 | 中 | 部署记录与线上实际版本不一致，安全基线文档缺失 | **已处置**：部署账本与线上 MD5、提交、入口和回滚路径重新对齐 |
| SEC-006 | 中 | 前端生产依赖有 4 个 high advisory | **已处置**：生产依赖 high=0、critical=0，完整前端测试通过 |
| SEC-007 | 低 | 安全响应头仍可收紧 | **部分处置**：增加 `Permissions-Policy`、隐藏 nginx 版本；HSTS/CSP 长期收紧需先盘点子域与资源域名 |
| SEC-008 | 低 | SSH 保留 X11/端口转发能力 | **已处置**：关闭 X11、TCP forwarding、GatewayPorts 和 tunnel |
| SEC-009 | 观察 | WebSocket 禁用了 Origin 检查 | **已处置**：浏览器默认要求同源；无 Origin 的 API 客户端保持兼容，跨源返回 403 |

## 三、详细发现

### SEC-001：遗留数据库测试容器可能形成外网入口（高）

**证据**

- `vigilant_galileo`：`postgres:18.1-alpine3.23`，发布 `0.0.0.0/[::]:32768 -> 5432`。
- `xenodochial_mestorf`：`redis:8.4-alpine`，发布 `0.0.0.0/[::]:32769 -> 6379`。
- 两者均带 testcontainers 标签，创建于 2026-07-30，`restart=no`，但目前仍在运行。
- Redis 配置中 `requirepass` 为空，默认 ACL 为 `NOPASS`。
- PostgreSQL 使用 8 位环境变量密码；`pg_hba.conf` 的远程连接要求 SCRAM，但密码强度不足。
- UFW 未启用，主机 INPUT 默认 ACCEPT；Docker NAT/FORWARD 规则存在，`DOCKER-USER` 没有额外拦截规则。
- 工作站外测被网络代理干扰，连负控端口也显示 OPEN；服务器自身 hairpin 正控又失败。因此本轮只能得出 **“外网可达性测不出来，云安全组状态未知”**，不能伪报为已开放或已关闭。

**影响**

若云安全组允许 32768/32769 入站，Redis 可能无需认证即可访问，PostgreSQL 也面临口令猜测、数据泄露或资源滥用风险。

**建议**

先只读确认是否有业务依赖，再经授权停止并删除这两个测试容器；若必须保留，应取消公网端口发布，仅绑定 `127.0.0.1` 或放进不对外的内部网络，并旋转相关凭据。另在云安全组明确拒绝 32768/32769。

**误报边界**：没有证据证明攻击者已连入；高风险来自“暴露方式 + 弱认证 + 缺少主机防火墙兜底”的组合。

### SEC-002：线上 Go 运行时存在 6 个可达漏洞（高）

**证据**

- 线上二进制构建信息：Go 1.26.5、Linux amd64、CGO 关闭。
- `govulncheck ./...` 报告以下 6 项均有当前代码调用路径，且均在 Go 1.26.6 修复：
  - [GO-2026-6218：net/url 路径处理可造成二次复杂度消耗](https://pkg.go.dev/vuln/GO-2026-6218)
  - [GO-2026-6090：crypto/tls KeyUpdate 可造成资源消耗](https://pkg.go.dev/vuln/GO-2026-6090)
  - [GO-2026-6089：明文 HTTP/2 未正确应用 ReadHeaderTimeout](https://pkg.go.dev/vuln/GO-2026-6089)
  - [GO-2026-6088：encoding/xml 递归导致栈耗尽](https://pkg.go.dev/vuln/GO-2026-6088)
  - [GO-2026-5972：encoding/asn1 递归导致栈耗尽](https://pkg.go.dev/vuln/GO-2026-5972)
  - [GO-2026-5026：IDNA/Punycode 标签处理歧义](https://pkg.go.dev/vuln/GO-2026-5026)

**影响**

主要是拒绝服务、解析歧义和资源消耗风险。`govulncheck` 的“可达”表示程序能走到相关函数，不代表漏洞已经被利用。

**建议**

使用 Go 1.26.6 在干净环境重建当前同一源码，跑现有全量门禁后发布；不要顺带合并业务改动，以便风险归因和回滚。

### SEC-003：注册保护存在，但强度不足以覆盖分布式滥用（中）

**证据**

- 生产公开配置：注册开启、Turnstile 开启；邮箱验证关闭、风控关闭、邀请注册关闭、所有 OAuth 开关关闭。
- 注册路由为每 IP 每分钟 5 次，并在 Redis 不可用时 fail-close：`backend/internal/server/routes/auth.go:219`。
- 限速键由“动作名 + 客户端 IP”组成：`backend/internal/middleware/rate_limiter.go:149`。
- 注册处理确实调用挑战验证：`backend/internal/handler/auth_handler.go:173`。
- 当前配置下缺少挑战令牌或挑战服务异常会失败关闭。

**影响**

能拦住单一来源脚本，但轮换 IP 的分布式注册可以绕开 IP 维度限速。现有证据能证明“闸门存在并被调用”，尚未做真实挑战成功/失败/重放测试，不能夸大为“已证明绝对有效”。

**建议**

安排一次不产生赠额的受控注册测试；根据业务接受度决定是否开启邮箱验证或风控。上游 v0.1.178 含邀请注册并发竞争修复，未来启用邀请功能前应纳入该补丁；当前邀请关闭，暂不是直接可利用入口。

### SEC-004：纯文本错误正文缺少敏感信息脱敏（中）

**位置**：`backend/internal/service/ops_service.go:984-999`

**证据**

- JSON 错误会按 authorization、API key、token、password、secret、cookie、credential 等字段名脱敏。
- 非 JSON 错误正文只做长度截断，原文直接保留：`ops_service.go:995-999`。
- 生产允许用户查看属于自己的错误请求详情。

**影响**

如果上游用纯文本返回密钥、邮箱、请求片段或调试内容，可能被保存，并展示给该请求所属用户或管理员。当前未在日志中发现实际密钥泄露。

**建议**

给非 JSON 文本增加 API Key、Bearer、Cookie、邮箱、密码/密钥模式的脱敏；保留截断。增加回归测试，确保普通错误语义仍可排障。

### SEC-005：线上版本账本失真，安全治理文件缺失（中）

**证据**

- 线上实际二进制：版本 `0.1.176-ssxz.20260818`，MD5 `31b683555c1ae5f81a5fa464e0faa0a6`，内嵌提交 `5c3bc1a61`，服务 active、`NRestarts=0`、`ExecMainStatus=0`。
- 当前 `DEPLOYED.md` 最新记录仍停留在 2026-08-17 / `460c0bf24`，与线上不一致。
- 项目缺少文档约定中引用的：
  - `docs/security/launch-checklist.md`
  - `docs/security/unified-chat-workspace-v1-security-strategy.md`

**影响**

发生漏洞或回滚时，工程人员可能检查错误提交、错误二进制或错误基线，放大事故处置风险。

**建议**

先补线上真实版本、MD5、提交、入口资源和回滚路径；再建立最小发布安全清单，要求每次发布核对源码提交、二进制哈希、migration、前端入口和回滚产物。

### SEC-006：前端依赖存在高等级公告，但当前可利用性有限（中）

**证据**

`pnpm audit --prod`：0 critical、4 high、21 moderate、1 low。

- `xlsx 0.18.5`：原型污染和 ReDoS 公告。当前只用于管理员导出表格，未发现读取攻击者上传工作簿的生产路径。
- `axios 1.17.0`：high 公告影响 Node HTTP adapter；当前是浏览器 SPA 直接依赖，线上浏览器路径不使用 Node adapter。
- `nanoid 3.3.17`：经 PostCSS 间接引入；公告需特定 `customAlphabet/customRandom` 零长度调用，项目未调用，主要属于构建链风险。

**影响**

现有调用方式降低了可利用性，但长期保留已知 high 依赖会积累供应链风险，尤其 `xlsx` 版本老旧。

**建议**

优先把 axios 升到修复版本；评估替换/升级 xlsx 并跑导出回归；通过锁文件更新间接 nanoid。依赖升级应单独构建和测试，不与安全运行时升级混在同一批发布。

### SEC-007：公网安全响应头可继续收紧（低）

**现状良好**：HTTP 自动跳 HTTPS；TLS 1.0/1.1 被拒绝，TLS 1.2/1.3 可用；CSP 使用 nonce，`frame-ancestors 'none'`；有 HSTS、`nosniff`、`DENY` 和严格来源 Referrer-Policy；恶意 Origin 的 CORS 预检被拒绝。

**缺口**

- 未设置 `Permissions-Policy`。
- HSTS 没有 `includeSubDomains` / `preload`。
- CSP 的 `connect-src https:`、`img-src https:` 较宽，`style-src` 仍允许 `'unsafe-inline'`。
- 源站响应泄露 `nginx/1.18.0 (Ubuntu)`。

**建议**

先在测试环境收集真实资源域名，再逐步收紧 CSP；确认所有子域均支持 HTTPS 后再加 HSTS 子域/preload；隐藏 nginx 版本，并增加最小 `Permissions-Policy`。

### SEC-008：SSH 保留不必要能力（低）

Root 目前为密钥登录，密码登录关闭，这是正向控制；但 `X11Forwarding` 和 `AllowTcpForwarding` 仍开启。若运维不需要，可关闭以缩小被盗密钥后的横向能力。

### SEC-009：WebSocket Origin 检查关闭（观察项）

**位置**：`backend/internal/handler/openai_live.go:218`

这里的 `InsecureSkipVerify: true` 是 WebSocket 的 Origin 检查开关，不是跳过 TLS 证书验证。该路由仍需 API Key 并绑定身份，因此没有直接证据构成未授权访问。建议补一条跨站 WebSocket 负面测试，避免未来认证方式变化后变成漏洞。

## 四、近 7 天攻击与稳定性迹象

- nginx 共约 75,503 次请求；识别到约 961 次明显扫描，包括 `.env`、`.git`、WordPress、phpMyAdmin、Actuator、server-status、phpunit 和路径穿越探测。
- 401：819；403：420；404：798；429：171。
- 502：336；503：1,012，主要集中在 `/v1/responses`、`/v1/chat/completions` 和图片接口，符合上游容量/可用性问题特征，不是入侵证据。
- systemd 近 7 天无 OOM、panic、fatal 或 migration 执行失败；没有检测到日志中的 API Key/Token/密码关键词泄露。

**判断**：网站一直被自动化扫描，这是公网服务的常态；目前日志支持“防护挡住了大量无效请求”，不支持“攻击者已成功进入”。

## 五、已验证的正向安全控制

- 管理员、用户资料接口未登录返回 401；随机路由返回 404；未暴露实际 pprof/metrics。
- SSRF：上游 URL 白名单默认强制开启且禁止关闭（`backend/internal/config/config.go:2422,3360`）；拦截 localhost、私网、链路本地地址，并检查 DNS 解析结果以防重绑定（`backend/internal/util/urlvalidator/validator.go:108-110`）。
- 越权：抽查订单、API Key、错误请求详情，查询均绑定当前用户；API Key reveal 响应带 `no-store`（`backend/internal/handler/api_key_handler.go:236`）。
- 支付：回调有 1 MB 请求体上限、供应商签名、金额/币种核验、订单供应商绑定、原子状态迁移和幂等履约；未发现伪造回调直接加余额路径。
- 文件：页面 slug、路径包含关系、符号链接和体积均有限制；图片上传有 multipart/MIME 检查。
- 前端：未发现 `v-html`、`innerHTML`、`eval`、`new Function`、宽泛 postMessage 或不安全 `_blank` 用法。
- systemd：`PrivateTmp=yes`、`ProtectHome=yes`、`ProtectSystem=strict`、`NoNewPrivileges=yes`。

## 六、上游更新判断

- 当前上游已有 [v0.1.177](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.177)、[v0.1.178](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.178)、[v0.1.179](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.179)。
- v0.1.178 含邀请注册码并发注册竞争修复及多项稳定性修复，可正常纳入后续合并评估。
- v0.1.179 含 226/227/228 三条 migration，并改变长上下文计费判断，属于计费行为变更；不能因为“版本新”就直接合并，必须单独审计计费和 migration。

## 七、建议处置顺序

1. **先处理 SEC-001**：确认遗留容器无依赖，封闭 32768/32769，旋转凭据。
2. **随后处理 SEC-002**：只升级 Go 1.26.6，重建同一源码并发布。
3. 补真实部署账本与安全发布清单，避免后续检查错版本。
4. 给纯文本错误加脱敏；再做一次真实 Turnstile 负面/重放验证。
5. 分批升级 axios/xlsx/nanoid；单独评估上游 v0.1.178，谨慎处理 v0.1.179 的计费变化。
6. 最后收紧响应头和 SSH 非必要能力。

## 八、检查限制

- 未执行破坏性或高负载渗透测试，未尝试绕过登录、暴力破解、消耗真实额度或写入生产数据。
- 云安全组无法从当前只读服务器证据中可靠确认；测试容器的公网可达性因此保持“测不出来”。
- 未做真实 Turnstile 成功、失败和令牌重放实验，因此只证明代码闸门存在并被调用。
- 日志“没有发现”不等于绝对没有；结论限于本轮采集窗口与现有日志留存。

## 九、授权后的处置结果（2026-08-24）

### SEC-001 已完成：测试数据库端口与遗留资源已清理

- 已确认 `vigilant_galileo`（PostgreSQL）与 `xenodochial_mestorf`（Redis）是无生产依赖的 testcontainers 遗留实例。
- 已确认无生产进程、反向代理或业务配置依赖后，删除两个容器及其匿名卷；该删除不可恢复。
- 主机已不再监听 `32768/32769`，容器端口发布也已消失。
- 处置后生产服务保持 `active`，`NRestarts=0`，健康检查正常。

### SEC-002 已修复：线上运行时升级为 Go 1.26.6

- 使用完全相同的源码 commit `5c3bc1a61a7a1a0679042df3c4be223b210394e5` 和逐文件哈希一致的前端 `dist`，仅更换 Go 工具链重建。
- 构建目标为 `linux/amd64`、`CGO_ENABLED=0`，带 `embed`、`trimpath`、`-s -w`。
- `go test ./...`、`go vet ./...` 均通过；Go 1.26.6 下 `govulncheck ./...` 报告代码可达漏洞为 0。
- 新线上二进制：MD5 `07f29fc7d77049d8d6e483b176078f2c`，128,516,258 bytes；旧包已备份到 `/opt/sub2api/backups/sub2api-pre-go1266-20260824`。
- 上线后版本为 `0.1.176-ssxz.20260818`，健康页 200，未带密钥的模型接口 401，前端入口仍为 `/assets/index-xHKXj2HA.js` 且可访问，启动日志未见 panic、fatal 或 migration/checksum 错误。
- 本次没有改业务代码、数据库、migration、OAuth 开关、计费配置或前端资源。

### SEC-003/004/006/007/008/009 已完成最小加固

- 注册：Turnstile 之后新增全局 `100/10min`、域名 `30/10min`、邮箱 `3/hour` 三级限速；Redis 键只保存不可逆摘要，不写邮箱/IP 明文。
- WebSocket：浏览器请求默认同源，跨源握手返回 403；不带 Origin 的服务端/API 客户端仍可使用。
- 错误存储：非 JSON 上游错误在持久化前脱敏 API Key、Bearer、Cookie、邮箱、密码/secret 等模式。
- 前端依赖：axios、DOMPurify、xlsx 及间接依赖完成安全升级；`pnpm audit --prod` 为 high=0、critical=0。
- 主机：nginx 隐藏版本；SSH 关闭 X11、端口转发、GatewayPorts 与 tunnel；新连接和配置语法验证通过。
- 响应头：公网响应已带 `Permissions-Policy: camera=(), geolocation=(), microphone=(), usb=()`。

### 安全版本已上线并验真

- 远端源码：`codex/security-hardening-20260824`，HEAD `85fd1c2a8062e41646a1ff92354a508fc11b3eb3`。
- 二进制：版本 `0.1.176-ssxz.20260824-security`，MD5 `f957ad9f4897b54e1a3f9edb1f80dbb5`，128,430,242 bytes。
- 前端：`/assets/index-C1NEpc15.js`，239 文件 / 7,415,928 bytes，HTML 中 `?v=` 为 0，入口资源 HTTP 200。
- 自动门禁：`go vet`、`go test`、`govulncheck`、`vue-tsc`、224 个前端测试文件 / 1539 个用例、生产构建全部通过。
- 生产验收：版本正确、管理员控制路由 401、随机负控路由 404、服务 `active`、`NRestarts=0`、`ExecMainStatus=0`。
- 回滚：`/opt/sub2api/backups/sub2api-pre-security-20260824`，MD5 `07f29fc7d77049d8d6e483b176078f2c`。
- 未执行生产数据库写入，未运行 migration，未修改 OAuth 开关或 `channel_model_pricing`。

### 上游 177–179 处置边界

- 已单独合入上游邀请注册并发原子性修复，避免等待整版本合并。
- 其余 177–179 共约 130 个普通提交，包含 222–228 migrations；其中 225/228 涉及 `channel_model_pricing`，不可盲合。
- 后续应按正常版本升级流程做 migration 编号/语义对账和克隆库预演，再合并完整上游，不影响本次安全加固结案。
