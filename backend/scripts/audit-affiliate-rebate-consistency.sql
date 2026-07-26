-- =============================================================================
-- 邀请返利数据一致性审计（只读，不改任何数据）
--
-- 背景：仓库中「累计返利」等聚合值存在多套互不相干的口径：
--   A = user_affiliates 冗余列（aff_history_quota / aff_quota / aff_frozen_quota / aff_count）
--   B = user_affiliate_ledger 按 action='accrue' 的 SUM/COUNT（管理端口径）
--   C = 用户页邀请记录：ledger 按「当前仍存在且仍绑定」的被邀人 join（受删号/解绑与 LIMIT 100 影响）
-- 本脚本量化 A/B/C 与派生口径的现存偏差，为单一事实源迁移提供基线数据。
--
-- 运行（生产，只读事务兜底）：
--   sudo -u postgres psql -d sub2api -f audit-affiliate-rebate-consistency.sql
-- =============================================================================

\set QUIET off
\pset footer off
BEGIN TRANSACTION READ ONLY;

\echo ''
\echo '================ [0] 盘面基础量 ================'
SELECT (SELECT COUNT(*) FROM user_affiliates)                                   AS ua_rows,
       (SELECT COUNT(*) FROM user_affiliates WHERE inviter_id IS NOT NULL)      AS bound_invitees,
       (SELECT COUNT(*) FROM user_affiliate_ledger)                             AS ledger_rows,
       (SELECT MIN(created_at) FROM user_affiliate_ledger)                      AS ledger_first,
       (SELECT MAX(created_at) FROM user_affiliate_ledger)                      AS ledger_last;

\echo '--- ledger 实际存在的 action 口径（若出现 accrue/transfer 之外的值，守恒公式需扩展） ---'
SELECT action,
       COUNT(*)                                  AS rows,
       COALESCE(SUM(amount), 0)::numeric(20, 8)  AS amount_total,
       COUNT(*) FILTER (WHERE amount < 0)        AS negative_rows,
       COUNT(*) FILTER (WHERE source_user_id IS NULL) AS null_source_rows
FROM user_affiliate_ledger
GROUP BY action
ORDER BY action;

\echo ''
\echo '================ [1] A vs B：aff_history_quota（终身累计列）对 SUM(accrue) ================'
WITH ledger_b AS (
    SELECT user_id, COALESCE(SUM(amount), 0) AS accrue_total
    FROM user_affiliate_ledger
    WHERE action = 'accrue'
    GROUP BY user_id
),
drift AS (
    SELECT ua.user_id,
           ua.aff_history_quota,
           COALESCE(lb.accrue_total, 0) AS accrue_total,
           (ua.aff_history_quota - COALESCE(lb.accrue_total, 0)) AS diff
    FROM user_affiliates ua
    LEFT JOIN ledger_b lb ON lb.user_id = ua.user_id
    WHERE ua.aff_history_quota <> 0 OR COALESCE(lb.accrue_total, 0) <> 0
)
SELECT COUNT(*)                                          AS users_with_rebate,
       COUNT(*) FILTER (WHERE ABS(diff) > 1e-8)          AS mismatched_users,
       COALESCE(SUM(diff), 0)::numeric(20, 8)            AS net_drift_a_minus_b,
       COALESCE(SUM(ABS(diff)), 0)::numeric(20, 8)       AS total_abs_drift
FROM drift;

\echo '--- A vs B 偏差 Top 20 明细 ---'
WITH ledger_b AS (
    SELECT user_id, COALESCE(SUM(amount), 0) AS accrue_total
    FROM user_affiliate_ledger
    WHERE action = 'accrue'
    GROUP BY user_id
)
SELECT ua.user_id,
       ua.aff_history_quota::numeric(20, 8)              AS a_history_quota,
       COALESCE(lb.accrue_total, 0)::numeric(20, 8)      AS b_ledger_accrue,
       (ua.aff_history_quota - COALESCE(lb.accrue_total, 0))::numeric(20, 8) AS diff
FROM user_affiliates ua
LEFT JOIN ledger_b lb ON lb.user_id = ua.user_id
WHERE ABS(ua.aff_history_quota - COALESCE(lb.accrue_total, 0)) > 1e-8
ORDER BY ABS(ua.aff_history_quota - COALESCE(lb.accrue_total, 0)) DESC
LIMIT 20;

\echo ''
\echo '================ [2] B vs C：用户页口径缺口（被邀人删号/解绑 + LIMIT 100 截断） ================'
\echo '--- 2.1 join 缺口：accrue 流水中已无法通过「现存且仍绑定的被邀人」join 出来的部分 ---'
WITH b AS (
    SELECT user_id, SUM(amount) AS b_total
    FROM user_affiliate_ledger
    WHERE action = 'accrue'
    GROUP BY user_id
),
c AS (
    SELECT ual.user_id, SUM(ual.amount) AS c_total
    FROM user_affiliate_ledger ual
    JOIN user_affiliates inv
      ON inv.user_id = ual.source_user_id
     AND inv.inviter_id = ual.user_id
    WHERE ual.action = 'accrue'
    GROUP BY ual.user_id
)
SELECT COUNT(*)                                                    AS inviters_with_accrue,
       COUNT(*) FILTER (WHERE b.b_total - COALESCE(c.c_total, 0) > 1e-8) AS inviters_losing_in_c,
       COALESCE(SUM(b.b_total - COALESCE(c.c_total, 0)), 0)::numeric(20, 8) AS total_amount_invisible_in_c
FROM b
LEFT JOIN c ON c.user_id = b.user_id;

\echo '--- 2.2 缺口归因：孤儿 accrue 流水分类 ---'
SELECT CASE
         WHEN ual.source_user_id IS NULL                        THEN 'source_user_id 为 NULL（被邀人已删号，FK SET NULL）'
         WHEN inv.user_id IS NULL                               THEN '被邀人 user_affiliates 行已不存在（删号 CASCADE）'
         WHEN inv.inviter_id IS DISTINCT FROM ual.user_id       THEN '被邀人仍在但 inviter 绑定已变更/置空'
         ELSE '正常可 join'
       END                                                      AS category,
       COUNT(*)                                                 AS rows,
       COALESCE(SUM(ual.amount), 0)::numeric(20, 8)             AS amount_total
FROM user_affiliate_ledger ual
LEFT JOIN user_affiliates inv ON inv.user_id = ual.source_user_id
WHERE ual.action = 'accrue'
GROUP BY 1
ORDER BY amount_total DESC;

\echo '--- 2.3 LIMIT 100 截断：现存绑定被邀人数超过 100 的邀请人 ---'
SELECT inviter_id,
       COUNT(*) AS bound_invitees,
       COUNT(*) - 100 AS truncated_rows
FROM user_affiliates
WHERE inviter_id IS NOT NULL
GROUP BY inviter_id
HAVING COUNT(*) > 100
ORDER BY COUNT(*) DESC;

\echo ''
\echo '================ [3] 邀请人数三口径：aff_count 列 vs 现存绑定数 vs 产生过返利的被邀人数 ================'
WITH bind AS (
    SELECT inviter_id AS user_id, COUNT(*) AS bound_cnt
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
),
led AS (
    SELECT user_id, COUNT(DISTINCT source_user_id) AS rebated_cnt
    FROM user_affiliate_ledger
    WHERE action = 'accrue' AND source_user_id IS NOT NULL
    GROUP BY user_id
),
merged AS (
    SELECT ua.user_id,
           ua.aff_count,
           COALESCE(b.bound_cnt, 0)  AS bound_cnt,
           COALESCE(l.rebated_cnt, 0) AS rebated_cnt
    FROM user_affiliates ua
    LEFT JOIN bind b ON b.user_id = ua.user_id
    LEFT JOIN led  l ON l.user_id = ua.user_id
    WHERE ua.aff_count <> 0 OR b.bound_cnt IS NOT NULL OR l.rebated_cnt IS NOT NULL
)
SELECT COUNT(*)                                                AS inviters,
       COUNT(*) FILTER (WHERE aff_count <> bound_cnt)          AS affcount_vs_bound_mismatch,
       COUNT(*) FILTER (WHERE aff_count < bound_cnt)           AS affcount_lt_bound,
       COUNT(*) FILTER (WHERE aff_count > bound_cnt)           AS affcount_gt_bound,
       COALESCE(SUM(aff_count - bound_cnt), 0)                 AS net_diff_affcount_minus_bound
FROM merged;

\echo '--- 邀请人数偏差 Top 20 明细 ---'
WITH bind AS (
    SELECT inviter_id AS user_id, COUNT(*) AS bound_cnt
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
)
SELECT ua.user_id,
       ua.aff_count,
       COALESCE(b.bound_cnt, 0) AS bound_cnt,
       ua.aff_count - COALESCE(b.bound_cnt, 0) AS diff
FROM user_affiliates ua
LEFT JOIN bind b ON b.user_id = ua.user_id
WHERE ua.aff_count <> COALESCE(b.bound_cnt, 0)
ORDER BY ABS(ua.aff_count - COALESCE(b.bound_cnt, 0)) DESC
LIMIT 20;

\echo ''
\echo '================ [4] 可结算额度守恒：列值 vs 由 ledger 推导 ================'
\echo '--- 4.1 守恒检查：aff_quota + aff_frozen_quota 应等于 SUM(accrue) - SUM(transfer) ---'
WITH led AS (
    SELECT user_id,
           COALESCE(SUM(amount) FILTER (WHERE action = 'accrue'), 0)   AS accrue_total,
           COALESCE(SUM(amount) FILTER (WHERE action = 'transfer'), 0) AS transfer_total,
           COALESCE(SUM(amount) FILTER (WHERE action = 'accrue' AND frozen_until IS NOT NULL AND frozen_until >  NOW()), 0) AS still_frozen,
           COALESCE(SUM(amount) FILTER (WHERE action = 'accrue' AND frozen_until IS NOT NULL AND frozen_until <= NOW()), 0) AS matured_pending_thaw
    FROM user_affiliate_ledger
    GROUP BY user_id
),
merged AS (
    SELECT ua.user_id,
           ua.aff_quota,
           ua.aff_frozen_quota,
           COALESCE(l.accrue_total, 0)          AS accrue_total,
           COALESCE(l.transfer_total, 0)        AS transfer_total,
           COALESCE(l.still_frozen, 0)          AS still_frozen,
           COALESCE(l.matured_pending_thaw, 0)  AS matured_pending_thaw,
           (ua.aff_quota + ua.aff_frozen_quota)
             - (COALESCE(l.accrue_total, 0) - COALESCE(l.transfer_total, 0)) AS conservation_diff
    FROM user_affiliates ua
    LEFT JOIN led l ON l.user_id = ua.user_id
    WHERE ua.aff_quota <> 0 OR ua.aff_frozen_quota <> 0 OR l.user_id IS NOT NULL
)
SELECT COUNT(*)                                                   AS users_checked,
       COUNT(*) FILTER (WHERE ABS(conservation_diff) > 1e-8)      AS conservation_broken_users,
       COALESCE(SUM(conservation_diff), 0)::numeric(20, 8)        AS net_conservation_diff,
       COUNT(*) FILTER (WHERE matured_pending_thaw > 1e-8)        AS users_with_matured_unthawed,
       COALESCE(SUM(matured_pending_thaw), 0)::numeric(20, 8)     AS matured_unthawed_total
FROM merged;

\echo '--- 4.2 守恒破坏 Top 20 明细（diff = 列值 - ledger 推导值） ---'
WITH led AS (
    SELECT user_id,
           COALESCE(SUM(amount) FILTER (WHERE action = 'accrue'), 0)   AS accrue_total,
           COALESCE(SUM(amount) FILTER (WHERE action = 'transfer'), 0) AS transfer_total
    FROM user_affiliate_ledger
    GROUP BY user_id
)
SELECT ua.user_id,
       ua.aff_quota::numeric(20, 8)         AS col_available,
       ua.aff_frozen_quota::numeric(20, 8)  AS col_frozen,
       COALESCE(l.accrue_total, 0)::numeric(20, 8)   AS ledger_accrue,
       COALESCE(l.transfer_total, 0)::numeric(20, 8) AS ledger_transfer,
       ((ua.aff_quota + ua.aff_frozen_quota) - (COALESCE(l.accrue_total, 0) - COALESCE(l.transfer_total, 0)))::numeric(20, 8) AS diff
FROM user_affiliates ua
LEFT JOIN led l ON l.user_id = ua.user_id
WHERE ABS((ua.aff_quota + ua.aff_frozen_quota) - (COALESCE(l.accrue_total, 0) - COALESCE(l.transfer_total, 0))) > 1e-8
ORDER BY ABS((ua.aff_quota + ua.aff_frozen_quota) - (COALESCE(l.accrue_total, 0) - COALESCE(l.transfer_total, 0))) DESC
LIMIT 20;

\echo '--- 4.3 冻结列 vs ledger 未到期冻结（管理端把 matured 也计入可结算，用户端靠 lazy thaw，两边窗口差即此值） ---'
WITH led AS (
    SELECT user_id,
           COALESCE(SUM(amount) FILTER (WHERE frozen_until IS NOT NULL AND frozen_until > NOW()), 0)  AS still_frozen,
           COALESCE(SUM(amount) FILTER (WHERE frozen_until IS NOT NULL AND frozen_until <= NOW()), 0) AS matured_pending
    FROM user_affiliate_ledger
    WHERE action = 'accrue'
    GROUP BY user_id
)
SELECT COUNT(*) FILTER (WHERE ABS(ua.aff_frozen_quota - (COALESCE(l.still_frozen, 0) + COALESCE(l.matured_pending, 0))) > 1e-8)
           AS frozen_col_mismatch_users,
       COALESCE(SUM(ua.aff_frozen_quota), 0)::numeric(20, 8)      AS frozen_col_total,
       COALESCE(SUM(l.still_frozen), 0)::numeric(20, 8)           AS ledger_still_frozen_total,
       COALESCE(SUM(l.matured_pending), 0)::numeric(20, 8)        AS ledger_matured_pending_total
FROM user_affiliates ua
LEFT JOIN led l ON l.user_id = ua.user_id
WHERE ua.aff_frozen_quota <> 0 OR l.user_id IS NOT NULL;

\echo ''
\echo '================ [5] 返利比例：显式 0 / 未设置 / 全局值 ================'
SELECT COUNT(*) FILTER (WHERE aff_rebate_rate_percent IS NULL)      AS rate_unset_follow_global,
       COUNT(*) FILTER (WHERE aff_rebate_rate_percent = 0)          AS rate_explicit_zero,
       COUNT(*) FILTER (WHERE aff_rebate_rate_percent > 0)          AS rate_custom_positive,
       COUNT(*) FILTER (WHERE aff_rebate_rate_percent < 0)          AS rate_negative_invalid
FROM user_affiliates;

\echo '--- 显式 0 比例的用户明细（确认是否人为关闭；若有历史返利则疑似误写 0） ---'
SELECT ua.user_id,
       ua.aff_rebate_rate_percent,
       ua.aff_count,
       ua.aff_history_quota::numeric(20, 8) AS history_quota,
       ua.updated_at
FROM user_affiliates ua
WHERE ua.aff_rebate_rate_percent = 0
ORDER BY ua.aff_history_quota DESC
LIMIT 50;

\echo '--- 全局 affiliate 相关设置现值 ---'
SELECT key, value, updated_at
FROM settings
WHERE key IN ('affiliate_enabled', 'affiliate_rebate_rate', 'affiliate_rebate_freeze_hours',
              'affiliate_rebate_duration_days', 'affiliate_rebate_per_invitee_cap')
ORDER BY key;

ROLLBACK;
\echo ''
\echo '================ 审计结束（只读事务已 ROLLBACK，未改任何数据） ================'
