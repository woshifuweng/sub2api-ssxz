-- ============================================================
-- 生产点修：user 1 返利历史底账补记 + 误写比例复位（2026-07-27 裁决）
--
-- 事实（2026-07-26 只读审计，全库唯一偏差账号）：
--   user 1: aff_history_quota=100，台账 accrue=0、transfer=100，
--           aff_quota=0、aff_frozen_quota=0，aff_rebate_rate_percent=0.00（显式）
--   → A vs B 差 100；守恒式差 100；「比例 0% vs 累计 $100」矛盾源头。
--
-- 裁决甲：补一条 action='adjust' 台账（+100，备注注明由来），不伪造 accrue。
-- 裁决二：aff_rebate_rate_percent 0.00 判定为误写（AffiliatesView v-model.number
--         清空输入框时 Number('')===0 静默提交所致），复位为 NULL（跟随全局）。
--         仅限 user 1；其他账号显式 0 是合法业务值，一律不动。
--
-- 用法：
--   dry-run（只读，展示将写入/修改的行）:
--     sudo -u postgres psql -d sub2api -f affiliate-rebate-fix-user1.sql
--   执行（需人工确认 dry-run 输出后）:
--     sudo -u postgres psql -d sub2api -v apply=1 -f affiliate-rebate-fix-user1.sql
--
-- 前置条件在执行段内硬校验，任一不符即 RAISE EXCEPTION 整体回滚。
-- 依赖 notes 列：migration 199 或本脚本内等价幂等 DDL（二者收敛）。
-- ============================================================

\set QUIET off
\set ON_ERROR_STOP on
\pset footer off

\echo ''
\echo '================ 当前状态（修复前） ================'
SELECT ua.user_id,
       ua.aff_rebate_rate_percent,
       ua.aff_history_quota::numeric(20,8) AS history_col,
       ua.aff_quota::numeric(20,8)         AS quota_col,
       ua.aff_frozen_quota::numeric(20,8)  AS frozen_col
FROM user_affiliates ua
WHERE ua.user_id = 1;

SELECT action,
       COUNT(*) AS rows,
       COALESCE(SUM(amount),0)::numeric(20,8) AS amount_total
FROM user_affiliate_ledger
WHERE user_id = 1
GROUP BY action
ORDER BY action;

\echo ''
\echo '================ [写入预览 1/2] 将 INSERT 的 adjust 台账行 ================'
SELECT 1                                    AS user_id,
       'adjust'                             AS action,
       100.00000000                         AS amount,
       NULL::bigint                         AS source_user_id,
       NULL::timestamptz                    AS frozen_until,
       '迁移131之前直写 aff_history_quota 列遗留，补记以恢复 quota+frozen = accrue−transfer 恒等式（2026-07-27 裁决甲；adjust 计入累计与守恒口径）' AS notes;

\echo ''
\echo '================ [写入预览 2/2] 将 UPDATE 的比例复位行 ================'
SELECT user_id,
       aff_rebate_rate_percent              AS current_value,
       'NULL（未设置，跟随全局默认）'        AS new_value
FROM user_affiliates
WHERE user_id = 1 AND aff_rebate_rate_percent = 0;

\echo ''
\echo '================ 恒等式修复前后对照（user 1） ================'
WITH led AS (
    SELECT COALESCE(SUM(amount) FILTER (WHERE action='accrue'),0)   AS accrue,
           COALESCE(SUM(amount) FILTER (WHERE action='adjust'),0)   AS adjust,
           COALESCE(SUM(amount) FILTER (WHERE action='transfer'),0) AS transfer
    FROM user_affiliate_ledger WHERE user_id = 1
), ua AS (
    SELECT aff_history_quota AS history, aff_quota + aff_frozen_quota AS settleable
    FROM user_affiliates WHERE user_id = 1
)
SELECT (ua.history  - (led.accrue + led.adjust))::numeric(20,8)                    AS history_residual_now,
       (ua.history  - (led.accrue + led.adjust + 100))::numeric(20,8)              AS history_residual_after_fix,
       (ua.settleable - (led.accrue + led.adjust - led.transfer))::numeric(20,8)   AS conserv_residual_now,
       (ua.settleable - (led.accrue + led.adjust + 100 - led.transfer))::numeric(20,8) AS conserv_residual_after_fix
FROM ua, led;

\if :{?apply}
\echo ''
\echo '################ APPLY 模式：开始写入 ################'
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

-- 与 migration 199 等价的幂等 DDL（先于迁移直连生产时收敛）
ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS notes TEXT NULL;
COMMENT ON COLUMN user_affiliate_ledger.notes IS '调整类流水的备注说明；常规 accrue/transfer 流水为 NULL';
COMMENT ON COLUMN user_affiliate_ledger.action IS 'accrue|transfer|adjust';

DO $$
DECLARE
    v_history  numeric;
    v_rate     numeric;
    v_accrue   numeric;
    v_transfer numeric;
    v_adjust_rows int;
    v_updated  int;
BEGIN
    SELECT aff_history_quota, aff_rebate_rate_percent
      INTO v_history, v_rate
      FROM user_affiliates WHERE user_id = 1 FOR UPDATE;

    SELECT COALESCE(SUM(amount) FILTER (WHERE action='accrue'),0),
           COALESCE(SUM(amount) FILTER (WHERE action='transfer'),0),
           COUNT(*) FILTER (WHERE action='adjust')
      INTO v_accrue, v_transfer, v_adjust_rows
      FROM user_affiliate_ledger WHERE user_id = 1;

    -- 前置条件：与 2026-07-26 审计事实完全一致才允许写入（防重复执行/防状态漂移）
    IF v_history IS NULL THEN
        RAISE EXCEPTION 'ABORT: user 1 无 user_affiliates 行';
    END IF;
    IF v_history <> 100 THEN
        RAISE EXCEPTION 'ABORT: aff_history_quota=% ≠ 100，状态已变化，人工复核', v_history;
    END IF;
    IF v_accrue <> 0 THEN
        RAISE EXCEPTION 'ABORT: SUM(accrue)=% ≠ 0，状态已变化，人工复核', v_accrue;
    END IF;
    IF v_transfer <> 100 THEN
        RAISE EXCEPTION 'ABORT: SUM(transfer)=% ≠ 100，状态已变化，人工复核', v_transfer;
    END IF;
    IF v_adjust_rows <> 0 THEN
        RAISE EXCEPTION 'ABORT: 已存在 % 条 adjust 流水，疑似重复执行', v_adjust_rows;
    END IF;
    IF v_rate IS DISTINCT FROM 0 THEN
        RAISE EXCEPTION 'ABORT: aff_rebate_rate_percent=% ≠ 0，状态已变化，人工复核', v_rate;
    END IF;

    INSERT INTO user_affiliate_ledger
        (user_id, action, amount, source_user_id, notes, created_at, updated_at)
    VALUES
        (1, 'adjust', 100.00000000, NULL,
         '迁移131之前直写 aff_history_quota 列遗留，补记以恢复 quota+frozen = accrue−transfer 恒等式（2026-07-27 裁决甲；adjust 计入累计与守恒口径）',
         NOW(), NOW());

    UPDATE user_affiliates
       SET aff_rebate_rate_percent = NULL,
           updated_at = NOW()
     WHERE user_id = 1 AND aff_rebate_rate_percent = 0;
    GET DIAGNOSTICS v_updated = ROW_COUNT;
    IF v_updated <> 1 THEN
        RAISE EXCEPTION 'ABORT: 比例复位影响行数=% ≠ 1', v_updated;
    END IF;

    RAISE NOTICE '写入完成：adjust +100 台账 1 条；user 1 比例已复位为 NULL';
END $$;

COMMIT;

\echo ''
\echo '================ 修复后校验 ================'
WITH led AS (
    SELECT COALESCE(SUM(amount) FILTER (WHERE action='accrue'),0)   AS accrue,
           COALESCE(SUM(amount) FILTER (WHERE action='adjust'),0)   AS adjust,
           COALESCE(SUM(amount) FILTER (WHERE action='transfer'),0) AS transfer
    FROM user_affiliate_ledger WHERE user_id = 1
), ua AS (
    SELECT aff_history_quota AS history, aff_quota + aff_frozen_quota AS settleable,
           aff_rebate_rate_percent AS rate
    FROM user_affiliates WHERE user_id = 1
)
SELECT (ua.history - (led.accrue + led.adjust))::numeric(20,8)                  AS history_residual,
       (ua.settleable - (led.accrue + led.adjust - led.transfer))::numeric(20,8) AS conserv_residual,
       ua.rate                                                                   AS rate_should_be_null
FROM ua, led;
\else
\echo ''
\echo '（dry-run 结束：未写入任何数据。确认以上预览后加 -v apply=1 执行。）'
\endif
