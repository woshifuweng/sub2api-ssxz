#!/usr/bin/env bash
# ============================================================
# 邀请返利对账守卫 (affiliate reconcile guard)
#
# 背景：返利数据长期存在多口径（user_affiliates 冗余列 vs
# user_affiliate_ledger 台账推导），历史上已在生产产生过偏差
# （user 1 的 100 无底账遗留）。单一事实源重构裁决：台账为准，
# 冗余列降级为缓存，本脚本是缓存与台账之间的常态化对账闭环。
#
# 每个用户三条恒等式残差（全部应为 0）：
#   r_hist    = aff_history_quota − (SUM(accrue) + SUM(adjust))
#   r_conserv = (aff_quota + aff_frozen_quota) − (SUM(accrue) + SUM(adjust) − SUM(transfer))
#   r_count   = aff_count − COUNT(现存绑定 inviter_id 指向该用户)
# 说明：冻结/解冻只在 aff_quota 与 aff_frozen_quota 之间搬运，
# r_conserv 对 lazy-thaw 的时机不敏感；aff_count 的事实源是
# inviter_id 关系本身（绑定不写台账，被邀人删号 CASCADE 后随之减少）。
#
# 关键设计（与 billing-reconcile-guard.sh 相同）：
# **不对残差的绝对值告警，只对残差的“位移”告警。**
# 存量偏差（若有）会让绝对值告警永远红着而被忽略；而任何新的
# 双写不一致、账外改列、漏写台账，都会让对应用户的残差发生位移。
#
# 用法：
#   首次建立基线：  sudo bash affiliate-reconcile-guard.sh --baseline
#   日常巡检(cron)：sudo bash affiliate-reconcile-guard.sh
#   查看当前残差：  sudo bash affiliate-reconcile-guard.sh --show
#
# 退出码：0=无位移  1=检测到位移(需人工核实)  2=执行错误
# ============================================================
set -euo pipefail

DB_NAME="${DB_NAME:-sub2api}"
STATE_DIR="${STATE_DIR:-/var/lib/sub2api-guard}"
BASELINE="$STATE_DIR/affiliate-residual.baseline"
# 允许的浮点误差；小于该值的位移视为噪声
EPSILON="${EPSILON:-0.00000001}"

MODE="${1:-check}"

say() { echo "[affiliate-guard $(date '+%Y-%m-%d %H:%M:%S')] $*"; }
die() { say "ERROR: $*"; exit 2; }

command -v psql >/dev/null || die "缺少 psql"
[ "$(id -u)" -eq 0 ] || die "需要 root（脚本内部用 sudo -u postgres）"

# 单一事实来源：这条 SQL 同时被 --baseline / --check / --show 使用，
# 三者绝不允许各写一份，否则基线与巡检口径漂移就会静默失效。
# 行格式：user_id|r_hist|r_conserv|r_count
read -r -d '' RESIDUAL_SQL <<'SQL' || true
WITH led AS (
    SELECT user_id,
           COALESCE(SUM(amount) FILTER (WHERE action = 'accrue'), 0)   AS accrue,
           COALESCE(SUM(amount) FILTER (WHERE action = 'adjust'), 0)   AS adjust,
           COALESCE(SUM(amount) FILTER (WHERE action = 'transfer'), 0) AS transfer
    FROM user_affiliate_ledger
    GROUP BY user_id
),
bind AS (
    SELECT inviter_id AS user_id, COUNT(*) AS bound
    FROM user_affiliates
    WHERE inviter_id IS NOT NULL
    GROUP BY inviter_id
)
SELECT ua.user_id || '|' ||
       ROUND((ua.aff_history_quota
              - (COALESCE(l.accrue, 0) + COALESCE(l.adjust, 0)))::numeric, 8) || '|' ||
       ROUND(((ua.aff_quota + ua.aff_frozen_quota)
              - (COALESCE(l.accrue, 0) + COALESCE(l.adjust, 0) - COALESCE(l.transfer, 0)))::numeric, 8) || '|' ||
       (ua.aff_count - COALESCE(b.bound, 0))
FROM user_affiliates ua
LEFT JOIN led  l ON l.user_id = ua.user_id
LEFT JOIN bind b ON b.user_id = ua.user_id
WHERE ua.aff_history_quota <> 0 OR ua.aff_quota <> 0 OR ua.aff_frozen_quota <> 0
   OR ua.aff_count <> 0 OR l.user_id IS NOT NULL OR b.user_id IS NOT NULL
ORDER BY ua.user_id;
SQL

snapshot() {
    sudo -u postgres psql -d "$DB_NAME" -X -t -A -c "$RESIDUAL_SQL" | sed '/^$/d'
}

# 三个残差字段与基线逐一比较；$1=当前行值 $2=基线行值
row_shifted() {
    awk -F'|' -v e="$EPSILON" '
        NR==1 { split($0, cur, "|") }
        NR==2 {
            for (i = 2; i <= 4; i++) {
                d = cur[i] - $i; if (d < 0) d = -d
                if (d > e) exit 1
            }
            exit 0
        }' <<EOF
$1
$2
EOF
}

# 新用户（基线没有）三个残差都必须为 0
row_all_zero() {
    awk -F'|' -v e="$EPSILON" '{
        for (i = 2; i <= 4; i++) { d = $i; if (d < 0) d = -d; if (d > e) exit 1 }
        exit 0
    }' <<<"$1"
}

case "$MODE" in
--show)
    say "当前每用户恒等式残差（r_hist | r_conserv | r_count，均应为 0）："
    snapshot | awk -F'|' '{printf "  user %-6s r_hist %-14s r_conserv %-14s r_count %s\n", $1, $2, $3, $4}'
    exit 0
    ;;

--baseline)
    mkdir -p "$STATE_DIR"
    chmod 0700 "$STATE_DIR"
    snapshot > "$BASELINE"
    chmod 0600 "$BASELINE"
    say "已建立基线：$BASELINE（$(wc -l < "$BASELINE") 个用户）"
    say "此后任何用户残差发生位移都会被判为异常。"
    exit 0
    ;;

check)
    [ -f "$BASELINE" ] || die "基线不存在，请先运行：$0 --baseline"
    CUR="$(mktemp)"; trap 'rm -f "$CUR"' EXIT
    snapshot > "$CUR"

    DRIFT=0
    while IFS= read -r line; do
        [ -n "$line" ] || continue
        uid="${line%%|*}"
        base_line="$(grep "^${uid}|" "$BASELINE" | head -1 || true)"
        if [ -z "$base_line" ]; then
            if row_all_zero "$line"; then
                continue
            fi
            say "!! 新用户 $uid 残差非零: ${line#*|} （新账目应当完全对得上）"
            DRIFT=1
            continue
        fi
        if ! row_shifted "$line" "$base_line"; then
            say "!! 用户 $uid 残差位移: 基线 ${base_line#*|} -> 当前 ${line#*|}"
            DRIFT=1
        fi
    done < "$CUR"

    # 基线里有、当前查不到的用户：删号（CASCADE 清列）或数据被清，值得知道
    while IFS='|' read -r uid _; do
        [ -n "$uid" ] || continue
        grep -q "^${uid}|" "$CUR" || { say "!! 用户 $uid 从结果集中消失（删号/数据被清？）"; DRIFT=1; }
    done < "$BASELINE"

    if [ "$DRIFT" -eq 0 ]; then
        say "OK：$(wc -l < "$CUR") 个用户三条恒等式残差与基线一致，冗余列与台账无新漂移"
        exit 0
    fi
    say "检测到返利数据位移。排查顺序："
    say "  1) r_hist/r_conserv 位移：是否有列 UPDATE 未配对台账 INSERT（两处写点：affiliate_repo.AccrueQuota、usage_billing_repo.accrueUsageAffiliateRebate），或账外 SQL 直改 aff_* 列"
    say "  2) r_count 位移：BindInviter 计数与 inviter_id 关系是否脱节（并发绑定/手工改 inviter_id）"
    say "  3) 确认为合法调整（应走 action='adjust' 台账）后，重新建立基线：$0 --baseline"
    exit 1
    ;;

*)
    die "未知参数: $MODE（可用: --baseline | --show | 无参数=巡检）"
    ;;
esac
