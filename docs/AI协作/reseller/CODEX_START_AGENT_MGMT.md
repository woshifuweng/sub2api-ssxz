# Codex 开工指令：Agent 管理 v1 实现

**状态：规格已锁定，可以开始编码。**

---

## 读取顺序

1. `AGENT_MGMT_V1_FINAL.md` — 完整规格（修订1–4全部已纳入）
2. `AGENT_MGMT_V1_FINAL_CODEX_SIGNOFF.md` — Codex 自己的审查原文（供参考）
3. `ADMIN_AGENTS_CLAUDE_RESPONSE.md` — Claude 对6个 blocker 的回答

---

## 实施顺序（按 AGENT_MGMT_V1_FINAL.md 第十节）

```
1. migration 202 → staging 验证无报错
2. 后端：reseller_repo.go（查询扩展 + 新增写方法）
3. 后端：reseller_service.go（Manager 降级保护 + 停用/启用逻辑 + 返佣检查修改）
4. 后端：reseller_handler.go（新增 PATCH / POST disable / POST enable / GET detail）
5. 前端：AdminAgents.vue（操作列重构 + 状态 Tab + 筛选栏 + 返佣列）
6. 前端：AdminAgentEditDrawer.vue（新建）
7. 前端：reseller.ts（新增 updateAgent、disableAgent、enableAgent 方法）
8. 测试：pnpm test:run + go test ./...
9. 部署：deploy_gate.sh 标准流程
```

---

## 关键约束（高优先，不可违背）

- **PATCH /agents/:id 必须是单一 PostgreSQL 事务**，不可调用 HTTP Affiliate API
- **aff_rebate_rate_percent** 是唯一返佣数据源（DECIMAL 5,2，0–100），commission_rate 已 deprecated
- **manager_id** 用于所有直属关系判断，granted_by 只读审计
- **降级保护**：包含 disabled 直属 Agent 也返回 409（停用≠解除归属）
- **最终撤销**：有直属 Agent 或待审批兑换 → 409
- **Step-up 敏感操作**：返佣比例/角色切换/manager_id 修改/最终撤销
- **Gateway + Gin 双注册**，路由集合必须一致
- **migration 202**：回填 manager_id 仅填可确认关系，不可确认保持 NULL

---

## 编译命令

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags embed ./cmd/server
```

**不要**执行 `go generate`（wire_gen.go 手动维护）。

---

## 完成后

完成后在本文件底部追加一行：

```
CODEX_STATUS: DONE | <commit_hash> | <日期>
```

然后更新 `docs/AI协作/HANDOFF.md` 说明当前已完成的节点。

CODEX_STATUS: DONE | 3c5db4ba6 | 2026-07-30
