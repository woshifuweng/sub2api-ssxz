#!/usr/bin/env bash
# ============================================================
# 计费对账守卫 (billing reconcile guard)
#
# 背景：2026-07-25 14:13 引入的 double-charge 缺陷（deduct + settle
# 在同一事务内各扣一次）在生产上跑了约 34 小时才被发现，期间没有任何
# 自动信号。发现它的手段就是本脚本里的这条 SQL —— 所以把它常态化。
#
# 原理：对每个产生过用量的用户计算账目残差
#     residual = 入账合计 - usage_logs.actual_cost 合计 - 当前余额
# 入账 = redeem_codes(status='used', 含管理员调整 type='admin_balance')
#      + user_affiliate_ledger(action='transfer', 返利转入余额)
#
# 关键设计：**不对残差的绝对值告警，只对残差的“变化”告警。**
# 生产存在若干历史账外写入（早于本脚本的手工 SQL 调账），它们让某些
# 用户的残差恒为非零。绝对值告警会永远红着而被忽略；而任何新的双扣、
# 漏扣、账外改余额，都会让对应用户的残差**发生位移**——那才是信号。
#
# 用法：
#   首次建立基线：  sudo bash billing-reconcile-guard.sh --baseline
#   日常巡检(cron)：sudo bash billing-reconcile-guard.sh
#   查看当前残差：  sudo bash billing-reconcile-guard.sh --show
#
# 退出码：0=无位移  1=检测到位移(需人工核实)  2=执行错误
# ============================================================
set -euo pipefail

DB_NAME="${DB_NAME:-sub2api}"
STATE_DIR="${STATE_DIR:-/var/lib/sub2api-guard}"
BASELINE="$STATE_DIR/billing-residual.baseline"
# 允许的浮点误差；小于该值的位移视为噪声
EPSILON="${EPSILON:-0.00000001}"

MODE="${1:-check}"

say() { echo "[billing-guard $(date '+%Y-%m-%d %H:%M:%S')] $*"; }
die() { say "ERROR: $*"; exit 2; }

command -v psql >/dev/null || die "缺少 psql"
[ "$(id -u)" -eq 0 ] || die "需要 root（脚本内部用 sudo -u postgres）"

# 单一事实来源：这条 SQL 同时被 --baseline / --check / --show 使用，
# 三者绝不允许各写一份，否则基线与巡检口径漂移就会静默失效。
read -r -d '' RESIDUAL_SQL <<'SQL' || true
WITH credit AS (
    SELECT used_by AS uid, SUM(value) AS amt
    FROM redeem_codes
    WHERE status = 'used' AND used_by IS NOT NULL
    GROUP BY 1
),
transfer AS (
    SELECT user_id AS uid, SUM(amount) AS amt
    FROM user_affiliate_ledger
    WHERE action = 'transfer'
    GROUP BY 1
),
spend AS (
    SELECT user_id AS uid, SUM(actual_cost) AS amt, COUNT(*) AS calls
    FROM usage_logs
    GROUP BY 1
)
SELECT u.id || '|' || ROUND(
           ( COALESCE(c.amt, 0) + COALESCE(t.amt, 0)
             - COALESCE(s.amt, 0) - u.balance )::numeric, 8)
FROM users u
LEFT JOIN credit   c ON c.uid = u.id
LEFT JOIN transfer t ON t.uid = u.id
LEFT JOIN spend    s ON s.uid = u.id
WHERE COALESCE(s.calls, 0) > 0
ORDER BY u.id;
SQL

snapshot() {
    # cd /tmp：postgres 用户读不到 root 的 cwd，不切换会每次刷
    # "could not change directory to /root" 警告，进 cron 后变成纯噪声
    (cd /tmp && sudo -u postgres psql -d "$DB_NAME" -X -t -A -c "$RESIDUAL_SQL") | sed '/^$/d'
}

case "$MODE" in
--show)
    say "当前每用户账目残差（入账 - 消费 - 余额）："
    snapshot | awk -F'|' '{printf "  user %-5s residual %s\n", $1, $2}'
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
    # 新用户（基线里没有）残差必须为 0：他们的全部账目都发生在修复之后
    while IFS='|' read -r uid residual; do
        [ -n "$uid" ] || continue
        base="$(grep "^${uid}|" "$BASELINE" | head -1 | cut -d'|' -f2 || true)"
        if [ -z "$base" ]; then
            if awk -v r="$residual" -v e="$EPSILON" 'BEGIN{exit (r<0?-r:r) <= e ? 0 : 1}'; then
                continue
            fi
            say "!! 新用户 $uid 残差非零: $residual （新账目应当完全对得上）"
            DRIFT=1
            continue
        fi
        if ! awk -v a="$residual" -v b="$base" -v e="$EPSILON" \
             'BEGIN{d=a-b; if(d<0)d=-d; exit d <= e ? 0 : 1}'; then
            say "!! 用户 $uid 残差位移: 基线 $base -> 当前 $residual"
            DRIFT=1
        fi
    done < "$CUR"

    # 基线里有、当前查不到的用户：可能被删号或用量被清，同样值得知道
    while IFS='|' read -r uid _; do
        [ -n "$uid" ] || continue
        grep -q "^${uid}|" "$CUR" || { say "!! 用户 $uid 从结果集中消失（删号/用量被清？）"; DRIFT=1; }
    done < "$BASELINE"

    if [ "$DRIFT" -eq 0 ]; then
        say "OK：$(wc -l < "$CUR") 个用户账目残差与基线一致，无双扣/漏扣/账外改动迹象"
        exit 0
    fi
    say "检测到账目位移。排查顺序："
    say "  1) 该用户是否有合法的管理员调额（查 redeem_codes type='admin_balance'）"
    say "  2) 是否有账外 SQL 直接改过 users.balance（本次事故的教训：一律走管理端接口）"
    say "  3) 是否又出现了重复扣费（查 usage_billing_repo.go 余额段是否只剩一次写入）"
    say "确认为合法调整后，重新建立基线：$0 --baseline"
    exit 1
    ;;

*)
    die "未知参数: $MODE（可用: --baseline | --show | 无参数=巡检）"
    ;;
esac
