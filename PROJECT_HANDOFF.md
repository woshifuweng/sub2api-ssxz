# SSXZ AI 项目交接

> 更新时间：2026-08-03  
> 本文用途：记录本轮只读核验结论，作为下一次协作的唯一恢复入口。  
> 安全边界：不记录密码、API Key、Token、JWT、Cookie、数据库口令或完整服务器凭据。

## 1. 核验范围与结论

- 本轮只读核验了四个本地工作树、`ssxz-server` 上的生产运行状态、生产 `schema_migrations`，以及 T5 Reseller 的文件、路由、接口和 migration。
- 本轮唯一写入目标是本文件；未修改业务代码，未运行部署，未执行生产写操作。
- 唯一后续开发基线是 `F:\CodexWork\_upgrade-v0.1.169-cherry-pick`。其他三个目录只用于历史差异追溯，不得混合作为开发基线。
- 生产服务当前正常，但生产二进制没有可核验的 Git revision；不得把本地 HEAD 写成生产 HEAD。

## 2. 唯一工作树与本地 Git 状态

### 唯一工作树

- 路径：`F:\CodexWork\_upgrade-v0.1.169-cherry-pick`
- 分支：`upgrade/v0.1.169`
- 本地 HEAD：`2ca51f952f4b4b171b0156ab97fe2d59774209c6`
- 最近提交：`2026-08-02T19:10:10+08:00 feat(reseller): deploy reseller frontend views`
- dirty 状态：dirty；当前完整未跟踪文件清单为：
  - `PROJECT_HANDOFF.md`
  - `backend/sub2api_linux`
  - `backend/sub2api_linux_ldflags`
  - `backend/sub2api_reseller_prod`
- 未发现 T5 源码或 migration 的未提交改动。上述二进制均不得混入 T5 代码提交。

### 其他工作树（只读历史目录）

| 路径 | 分支 | HEAD | 最近提交 | dirty 状态 |
|---|---|---|---|---|
| `F:\CodexWork\_upgrade-v0.1.169-merged` | `upgrade/v0.1.169-merged` | `682c4fe0e61b851508fa976ac693e0f68a0639eb` | `2026-07-31T23:24:44+08:00 Merge pull request #5147 from Wei-Shaw/feat/moderation-proxy-and-smtp-starttls` | dirty；1645 个 index 路径，12 个同时存在工作区改动 |
| `F:\CodexWork\_deploy-t1t2-reseller-recharge-20260801` | `codex/fix-logo-crossfade-timing` | `2b944e5738f4bae4aa825ea4394bdef302683c2a` | `2026-08-01T06:57:41+08:00 fix(brand): synchronize logo crossfade` | dirty；2 个未跟踪二进制 |
| `F:\CodexWork\_upgrade-v0.1.168` | `codex/fix-admin-route-parity` | `cf48fc971bb223a14d48671a6fb5b339acc99d25` | `2026-08-01T04:55:31+08:00 fix(home): align hero copy and logo animation` | dirty；11 个已修改路径、5 个未跟踪路径 |

上述 HEAD 只代表各自本地工作树，不能互相替代，也不能代表生产 HEAD。

## 3. 本地已完成内容

以唯一工作树 HEAD 为准，已确认存在以下已提交内容：

- `7bc055d52`：Reseller 作用域、角色、基础 handler/service/repository、路由、API/store、初始 migration `200_reseller_roles.sql`。
- `50ec6d044`：余额兑换/提现流程加固、migration `201_reseller_fields_hardening.sql`、用户与 Admin 路由、提现历史和状态组件。
- `3c5db4ba6`：Agent 生命周期管理、管理层级、角色授权/撤销、Admin Agent 管理、migration `202_agent_lifecycle.sql` 及相关测试。
- `2ca51f952`：用户侧 Reseller 概览、下线、提现、Manager 页面和对应路由/导航调整。
- 本地目录还存在 `203_account_balance_ledger.sql`、`204_add_billing_model_to_usage_logs.sql`；它们不是 T5 Reseller 专属 migration。

已确认的 Reseller 核心文件包括：

- 后端：`reseller_handler.go`、`reseller_service.go`、`reseller_repo.go`、`routes/user.go`、`routes/admin.go`、`wire_gen.go`。
- 数据库：`200_reseller_roles.sql`、`201_reseller_fields_hardening.sql`、`202_agent_lifecycle.sql`。
- 前端：`api/reseller.ts`、`stores/reseller.ts`、`AgentDashboard.vue`、`AgentRecruits.vue`、`AgentWithdrawals.vue`、`ManagerDashboard.vue`。
- Admin 前端：`AdminAgents.vue`、`AdminWithdrawals.vue`、`AdminAgentDetailDrawer.vue`、`AdminAgentEditDrawer.vue`、`AdminAgentGrantDialog.vue`。
- 测试：handler、service、repository、migration、API、store 和 Admin 页面均有对应测试文件。

`admin_reseller_handler.go` 不存在不是漏项：当前 Admin Reseller handler 与用户侧 handler 合并在 `backend/internal/handler/reseller_handler.go` 中。

## 4. 生产只读核验

核验对象：`ssxz-server` 上的 `/opt/sub2api/sub2api` 和 `sub2api` 服务。

- Go build info：`go1.26.5`；主模块为 `github.com/Wei-Shaw/sub2api`，构建包为 `github.com/Wei-Shaw/sub2api/cmd/server`。
- 生产二进制 SHA-256：`87fcb0cabbbd02fcfb410989bba41b5d7088a46956a47f5e026b40be359e32e0`。
- 服务状态：`ActiveState=active`。
- 重启次数：`NRestarts=0`。
- 健康检查：`http://127.0.0.1:8080/health` 返回 `{"status":"ok"}`。
- 生产 HEAD：不可核验。`go version -m` 未输出 `vcs.revision`、`vcs.time` 或 `vcs.modified`；`/opt/sub2api` 也不是可读取 Git 源码工作树。因此不能从现有证据推断生产 commit，更不能写成 `2ca51f952f4b4b171b0156ab97fe2d59774209c6`。

生产 `sub2api` 数据库 `public.schema_migrations` 最新记录：

| 文件 | applied_at |
|---|---|
| `204_add_billing_model_to_usage_logs.sql` | `2026-07-31 03:55:20.88492+08` |
| `203_account_balance_ledger.sql` | `2026-07-31 00:00:25.795444+08` |
| `202_agent_lifecycle.sql` | `2026-07-30 21:56:20.193215+08` |
| `201_reseller_fields_hardening.sql` | `2026-07-30 15:40:15.247504+08` |
| `200_reseller_roles.sql` | `2026-07-30 04:31:56.436787+08` |

以上表格是按生产数据库实际查询返回的最新 5 条记录（按 `applied_at` 倒序），只能证明这些记录存在；“最高编号为 204”本身不等于自动证明 200–204 的每一条 migration 都已执行。当前已逐条记录 200、201、202、203、204 的文件名和时间。

补充只读观察：服务器上的 `sub2api_staging_0168` 数据库最新为 `201_reseller_fields_hardening.sql`；这不证明 staging 与生产连接关系，后续操作前仍需单独确认数据库隔离。

## 5. T5 Reseller 实际完成度

### 已实现

- 后端核心链路已实现：角色读取、Agent/Manager/Admin 权限检查、Agent dashboard、下线列表、Manager Agent 管理、授权/撤销、Admin Agent 管理、提现申请/取消/列表/审批。
- 当前实际用户路由包含：
  - `GET /api/v1/user/reseller/role`
  - `GET /api/v1/user/reseller/dashboard`
  - `GET /api/v1/user/reseller/recruits`
  - `GET /api/v1/user/reseller/withdrawals`
  - `POST /api/v1/user/reseller/withdraw`
  - `POST /api/v1/user/reseller/withdrawals/:id/cancel`
  - `GET /api/v1/user/reseller/manager/dashboard`
  - `GET /api/v1/user/reseller/manager/agents`
  - `GET /api/v1/user/reseller/manager/agents/:id`
  - `POST /api/v1/user/reseller/manager/agents/:id/grant`
  - `DELETE /api/v1/user/reseller/manager/agents/:id/role`
  - `GET /api/v1/user/reseller/manager/withdrawals`
- 当前实际 Admin 路由包含 Agent 列表/详情/更新/启停/角色授权撤销，以及提现列表和审批。
- migration `200`、`201`、`202` 已存在于唯一工作树，也已在生产 `sub2api` 数据库中执行。
- 当前实现使用 `user_reseller_roles` 与 `affiliate_withdraw_requests`；T5 规格中提出的 `reseller_withdrawals` 不是当前实际表名，不能未经产品确认直接改表或重复建表。

### 未完成或与 T5 规格不一致

当前不能称为“T5 完整重做”，因为以下缺口已被核验：

- 缺失前端文件：
  - `frontend/src/views/reseller/AgentCommission.vue`
  - `frontend/src/views/reseller/AgentInvite.vue`
  - `frontend/src/components/reseller/RecruitDetailDrawer.vue`
  - `frontend/src/components/reseller/WithdrawModal.vue`
- 缺失或未按规格提供的接口：
  - `GET /api/v1/user/reseller/recruits/:userId/detail`
  - `GET /api/v1/user/reseller/commission`
  - `GET /api/v1/user/reseller/invite`
  - `GET /api/v1/user/reseller/manager/agents/:agentId/recruits`
- 提现接口契约不同：当前为 `POST /user/reseller/withdraw`，规格要求 `POST /user/reseller/withdrawals`；当前 Admin 审批为 `POST /admin/reseller/withdrawals/:id/review`，规格要求 `PUT /admin/reseller/withdrawals/:id`，且规格包含 `mark_paid` 状态流转，当前实现未提供该状态。
- 当前 Reseller 用户端实际有概览、下线、提现和 Manager 页面共 4 个页面；佣金明细、推广工具和两个专用弹窗/抽屉未完成。
- 因此实际完成度结论为：**后端核心与现有 V1 管理链路已完成，T5 规格要求的完整用户端和接口契约尚未完成；不得按“全量完成”处理。**

## 6. 下一步唯一任务

**在获得用户确认后，只在 `F:\CodexWork\_upgrade-v0.1.169-cherry-pick` 对照 T5 规格补齐上述缺失接口、接口契约和前端页面，并做定向本地验证；当前不自动开始实现、不提交、不部署。**

补齐前必须先对产品确认两处差异：提现状态模型（`approved/rejected/cancelled` 与规格的 `paid_out`）以及现有 `affiliate_withdraw_requests` 是否继续作为 T5 数据源。

## 7. 禁止修改范围与操作边界

- 本轮及未获明确授权前：不修改业务代码、不修改数据库、不执行 migration、不部署、不重启生产服务。
- T5 不得修改：
  - `backend/internal/service/billing_service.go`
  - `backend/internal/service/openai_gateway_service.go`
- 不得顺带修改图片计费、支付、充值、渠道路由、账户清理或其他无关业务。
- 不运行 `go generate` 或 Wire 命令；依赖注入如获授权，只能手动维护并核验 `wire_gen.go`。
- 不执行 `git reset --hard`、`git clean`、强制覆盖、批量回滚或删除未知改动；不得使用 `git add -A` 混入无关文件。
- 生产操作必须另行获得明确授权，并经过备份、构建、验证、回滚准备和 deploy gate；本交接文档不授权任何生产操作。
- 不在 staging 数据库身份未确认时运行 migration；不得假设 staging 与生产隔离。
- 文档、日志、截图和回复中不得出现密码、API Key、Token、JWT、Cookie、数据库口令或完整服务器凭据。

## 8. 关键路径

### 唯一工作树

- `F:\CodexWork\_upgrade-v0.1.169-cherry-pick`
- `F:\CodexWork\_upgrade-v0.1.169-cherry-pick\PROJECT_HANDOFF.md`

### T5 规格

- `C:\Users\24091\Documents\中转站项目\docs\AI协作\T5_RESELLER_REBUILD_CODEX.md`

### Reseller 后端

- `backend/internal/handler/reseller_handler.go`
- `backend/internal/service/reseller_service.go`
- `backend/internal/repository/reseller_repo.go`
- `backend/internal/server/routes/user.go`
- `backend/internal/server/routes/admin.go`
- `backend/cmd/server/wire_gen.go`
- `backend/migrations/200_reseller_roles.sql`
- `backend/migrations/201_reseller_fields_hardening.sql`
- `backend/migrations/202_agent_lifecycle.sql`

### Reseller 前端

- `frontend/src/api/reseller.ts`
- `frontend/src/stores/reseller.ts`
- `frontend/src/views/reseller/`
- `frontend/src/views/admin/reseller/`
- `frontend/src/components/reseller/`
- `frontend/src/components/user/AppSectionShell.vue`
- `frontend/src/router/index.ts`

### 生产只读位置

- SSH 主机别名：`ssxz-server`
- 服务名：`sub2api`
- 二进制：`/opt/sub2api/sub2api`
- 健康端点：`http://127.0.0.1:8080/health`
- schema 表：`public.schema_migrations`

## 9. 已知风险

- 生产 HEAD 不在 Go build info 中，生产二进制 SHA-256 只能作为二进制身份，不足以证明来源 commit。
- 四个本地目录均 dirty；尤其 `_upgrade-v0.1.169-merged` 有大量 index 改动，不能从其工作区整体复制文件。
- 本地唯一工作树还有未跟踪二进制产物；不得把它们纳入 T5 代码提交或文档以外的变更。
- T5 规格与现有 Reseller V1 的表名、提现状态和 API 路径存在差异；直接补实现可能造成重复表、重复 handler 或破坏现有接口。
- migration 文件存在不等于任意数据库已执行；本次只确认了生产 `sub2api` 和服务器上一个 staging 数据库的最新记录。
- Reseller 涉及角色权限、邀请关系、佣金累计、审批状态和余额并发一致性；后续实现必须用后端权限、事务/锁和集成测试验证，不能只靠前端隐藏菜单。
- 生产服务健康不代表本地 HEAD 与生产二进制一致；任何部署判断都必须重新取得可追溯构建证据。

