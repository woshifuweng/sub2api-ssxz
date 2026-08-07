#!/usr/bin/env bash
# 生产状态闸门：发现回归就以非 0 退出，不只是打印给人看。
#
# 用法:
#   bash scripts/verify-deployed.sh                    # 检查生产
#   bash scripts/verify-deployed.sh https://host       # 检查指定站点
#   bash scripts/verify-deployed.sh --update-baseline  # 部署成功后记录新基线
#
# 设计原则：这个脚本防的是「部署把已上线功能顶掉」——2026-08-06 经销商
# 功能（13 接口 + 6 页面）就是这么在没人察觉的情况下从生产消失的。
# 文档防不住这种事，只有会失败的检查防得住。

set -u

HOST="https://api.ssxzapi.com"
UPDATE_BASELINE=0
for arg in "$@"; do
  case "$arg" in
    --update-baseline) UPDATE_BASELINE=1 ;;
    http*)             HOST="$arg" ;;
  esac
done

BASELINE="$(dirname "$0")/deployed-baseline.txt"
FAILED=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAILED=$((FAILED + 1)); }
info() { printf '        %s\n' "$1"; }

code_of() {
  curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$HOST$1" 2>/dev/null
}

# ---------------------------------------------------------------- 1. 探测法自检
# 未鉴权 GET admin 路由: 401=路由存在, 404=路由不存在。
# 先验证方法本身有效——控制组必须 401，否则后面所有 404 都没有意义。
echo "== 0. 探测法自检 =="
CONTROL=$(code_of /api/v1/admin/users)
if [ "$CONTROL" = "401" ]; then
  pass "控制组 /admin/users = 401，探测法有效"
else
  fail "控制组 /admin/users = $CONTROL（期望 401）"
  info "探测法失效：站点可能宕机、被 WAF 拦截或鉴权方式已变。"
  info "后续 404 结果全部不可采信，直接中止。"
  exit 1
fi

# ---------------------------------------------------------------- 2. 版本戳
echo
echo "== 1. 版本戳 =="
VERSION=$(curl -s --max-time 15 "$HOST/api/v1/settings/public" 2>/dev/null \
  | tr ',' '\n' | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
if [ -z "$VERSION" ]; then
  fail "取不到版本号"
elif echo "$VERSION" | grep -q -- '-ssxz\.'; then
  pass "版本 $VERSION（含血脉戳）"
else
  fail "版本 $VERSION 没有 -ssxz. 戳"
  info "说明这次构建漏了 ldflags，二进制无法自证血脉——循环会重新开始。"
  info "正确构建见 DEPLOYED.md「构建命令」段。"
fi

# ---------------------------------------------------------------- 3. 路由面
# 必须存在的路由。任何一条变 404 = 有功能被部署顶掉了。
echo
echo "== 2. 路由面（防功能被顶掉）=="
MUST_EXIST="
/api/v1/admin/users
/api/v1/admin/accounts
/api/v1/admin/channel-monitors
/api/v1/admin/channel-monitor-templates
"
for route in $MUST_EXIST; do
  c=$(code_of "$route")
  if [ "$c" = "401" ]; then
    pass "$route"
  else
    fail "$route = $c（期望 401=存在）"
    info "这条路由从生产消失了，很可能是部署用了不含该功能的分支。"
  fi
done

# 已知缺失的路由：仅作观测，不算失败。变 401 是好事（功能回来了）。
echo
echo "== 3. 已知缺失（变 401 = 功能已恢复，记得更新 DEPLOYED.md）=="
for route in /api/v1/admin/reseller/agents /api/v1/admin/audit-logs; do
  c=$(code_of "$route")
  if [ "$c" = "401" ]; then
    info "$route = 401 → 已恢复，请更新 DEPLOYED.md"
  else
    info "$route = $c → 仍缺失（已知状态）"
  fi
done

# ---------------------------------------------------------------- 4. CSP
echo
echo "== 4. CSP（作图页内嵌 + 没修坏支付/验证码）=="
CSP=$(curl -sI --max-time 15 "$HOST/app/image" 2>/dev/null \
  | tr -d '\r' | grep -i '^content-security-policy:')
FRAME_SRC=$(echo "$CSP" | tr ';' '\n' | grep -i 'frame-src')

if [ -z "$FRAME_SRC" ]; then
  fail "取不到 frame-src"
else
  echo "$FRAME_SRC" | grep -q "'self'" \
    && pass "frame-src 含 'self'（同源 /image/ 可内嵌）" \
    || { fail "frame-src 缺 'self'"; info "作图页会显示成被屏蔽——显式 frame-src 不继承 default-src。"; }

  # 第三方域不能在修 CSP 时被顺手删掉
  for domain in challenges.cloudflare.com js.stripe.com hooks.stripe.com pay.ldxp.cn; do
    echo "$FRAME_SRC" | grep -q "$domain" \
      && pass "frame-src 保留 $domain" \
      || fail "frame-src 丢了 $domain（验证码/支付/充值会坏）"
  done
fi

# frame-ancestors 必须仍是 'none'：放宽会让外站能嵌我们（点击劫持）
echo "$CSP" | tr ';' '\n' | grep -i 'frame-ancestors' | grep -q "'none'" \
  && pass "frame-ancestors 仍为 'none'（外站嵌不了我们）" \
  || fail "frame-ancestors 被放宽了，存在点击劫持风险"

# ---------------------------------------------------------------- 5. 前端资源
# 后端用 -tags embed 构建会把当前树的 dist 打进二进制。若树里 dist 是旧的，
# 部署后端会顺手把前端打回旧版。这一步拿生产实际引用的资源和基线比。
echo
echo "== 5. 前端资源（防 embed 打回旧前端）=="
ASSETS=$(curl -s --max-time 15 "$HOST/" 2>/dev/null \
  | grep -o '/assets/[A-Za-z0-9._-]*' | sort -u)
if [ -z "$ASSETS" ]; then
  fail "取不到前端资源清单"
else
  COUNT=$(echo "$ASSETS" | wc -l | tr -d ' ')
  if [ "$UPDATE_BASELINE" = "1" ]; then
    printf '%s\n' "$ASSETS" > "$BASELINE"
    pass "已写入基线：$COUNT 个资源 → $BASELINE"
  elif [ ! -f "$BASELINE" ]; then
    printf '%s\n' "$ASSETS" > "$BASELINE"
    info "基线不存在，已用当前状态初始化（$COUNT 个资源）"
  else
    if DIFF=$(printf '%s\n' "$ASSETS" | diff "$BASELINE" - 2>&1); then
      pass "前端资源与基线一致（$COUNT 个）"
    else
      fail "前端资源与基线不一致"
      info "可能是正常的前端更新，也可能是 embed 把前端打回了旧版。"
      info "确认是正常更新后跑：bash scripts/verify-deployed.sh --update-baseline"
      printf '%s\n' "$DIFF" | head -20 | sed 's/^/        /'
    fi
  fi
fi

# ---------------------------------------------------------------- 6. 作图页
echo
echo "== 6. 作图页自身 =="
IMG=$(code_of /image/)
[ "$IMG" = "200" ] && pass "/image/ = 200" || fail "/image/ = $IMG（期望 200）"

# ---------------------------------------------------------------- 结论
echo
if [ "$FAILED" -eq 0 ]; then
  printf '\033[32m全部通过\033[0m  生产版本 %s\n' "${VERSION:-未知}"
  exit 0
fi
printf '\033[31m%s 项失败\033[0m  部署前请先解决，或确认是有意变更后更新基线/DEPLOYED.md\n' "$FAILED"
exit 1
