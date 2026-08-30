-- 邀请返利台账支持 'adjust' 调整类流水（历史遗留数据补记）。
--
-- 背景：migration 131 建台账之前，存在直写 user_affiliates 冗余列、无台账底账的
-- 历史数据（生产 user 1 的 aff_history_quota=100 即此类：列上有 100，台账无 accrue，
-- 后续 transfer 100 使守恒式 aff_quota+aff_frozen_quota = SUM(accrue)-SUM(transfer)
-- 对该用户永久破坏）。裁决（2026-07-27）：以 action='adjust' 补记底账，不伪造 accrue
-- （accrue 语义是真实产生的返利收益，报表不得出现从未发生的推荐收入）。
--
-- 本迁移只做 schema：
--   1) action 语义扩展为 'accrue' | 'transfer' | 'adjust'；
--   2) 新增 notes 列存放调整备注（常规 accrue/transfer 流水保持 NULL）。
-- 数据补记本身是生产点修，不在迁移内（见 backend/scripts/affiliate-rebate-fix-user1.sql）。
--
-- 口径影响（一致性守卫 affiliate-reconcile-guard.sh 与后续读路径统一采用）：
--   终身累计返利 = SUM(accrue) + SUM(adjust)
--   可结算守恒   aff_quota + aff_frozen_quota = SUM(accrue) + SUM(adjust) - SUM(transfer)

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS notes TEXT NULL;

COMMENT ON COLUMN user_affiliate_ledger.notes IS '调整类流水的备注说明；常规 accrue/transfer 流水为 NULL';
COMMENT ON COLUMN user_affiliate_ledger.action IS 'accrue|transfer|adjust';
