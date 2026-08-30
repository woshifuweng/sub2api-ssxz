#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
release_script="$script_dir/preflight-systemd-release.sh"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-release-test.XXXXXX")"
trap 'rm -rf -- "$test_root"' EXIT

assert_contains() {
  local file="$1" expected="$2"
  grep -Fq -- "$expected" "$file" || {
    printf 'expected %s to contain: %s\n' "$file" "$expected" >&2
    cat "$file" >&2
    exit 1
  }
}

setup_fixture() {
  local name="$1"
  fixture="$test_root/$name"
  mkdir -p "$fixture/bin" "$fixture/releases/old" "$fixture/releases/new"
  : >"$fixture/production.env"
  : >"$fixture/preflight.env"
  printf 'PGDMP fake release backup\n' >"$fixture/backup.dump"
  ln -s "$fixture/releases/old" "$fixture/current"

  cat >"$fixture/releases/new/sub2api" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "-version" ]]; then
  echo "Sub2API test-candidate"
  exit 0
fi
trap 'exit 0' TERM INT
while :; do sleep 1; done
EOF
  chmod +x "$fixture/releases/new/sub2api"
  cp "$fixture/releases/new/sub2api" "$fixture/releases/old/sub2api"

  cat >"$fixture/bin/runuser" <<'EOF'
#!/usr/bin/env bash
if [[ "${RUNUSER_DENY_READ:-}" != "" && "$*" == *"test -r ${RUNUSER_DENY_READ}"* ]]; then
  exit 1
fi
while [[ $# -gt 0 ]]; do
  case "$1" in
    -u) shift 2 ;;
    --) shift; break ;;
    *) break ;;
  esac
done
exec "$@"
EOF

  cat >"$fixture/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
echo "$*" >>"$SYSTEMCTL_LOG"
if [[ "${SYSTEMCTL_FAIL:-false}" == "true" ]]; then exit 1; fi
EOF

  cat >"$fixture/bin/curl" <<'EOF'
#!/usr/bin/env bash
url=""
for arg in "$@"; do url="$arg"; done
if [[ "$url" == *":18081/"* ]]; then
  [[ "${FAIL_PREFLIGHT_HEALTH:-false}" != "true" ]]
else
  count=0
  if [[ -f "$PRODUCTION_CURL_COUNT" ]]; then count="$(cat "$PRODUCTION_CURL_COUNT")"; fi
  count=$((count + 1))
  printf '%s' "$count" >"$PRODUCTION_CURL_COUNT"
  if [[ "${FAIL_FIRST_PRODUCTION_HEALTH:-false}" == "true" && "$count" -eq 1 ]]; then exit 1; fi
fi
EOF

  cat >"$fixture/bin/chown" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

  cat >"$fixture/verify-backup-restore.sh" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$RESTORE_VERIFY_LOG"
[[ "${FAIL_RESTORE_VERIFY:-false}" != "true" ]]
EOF

  chmod +x "$fixture/bin/runuser" "$fixture/bin/systemctl" "$fixture/bin/curl" "$fixture/bin/chown" "$fixture/verify-backup-restore.sh"
  export PATH="$fixture/bin:$original_path"
  export RELEASE_ROOT="$fixture/releases"
  export CURRENT_LINK="$fixture/current"
  export PRODUCTION_ENV_FILE="$fixture/production.env"
  export PREFLIGHT_ENV_FILE="$fixture/preflight.env"
  export SERVICE_USER="test-user"
  export SERVICE_NAME="sub2api-test.service"
  export PREFLIGHT_PORT=18081
  export PREFLIGHT_HEALTH_URL="http://127.0.0.1:18081/health"
  export PRODUCTION_HEALTH_URL="http://127.0.0.1:18080/health"
  export HEALTH_TIMEOUT_SECONDS=1
  export BACKUP_FILE="$fixture/backup.dump"
  export RESTORE_VERIFY_SCRIPT="$fixture/verify-backup-restore.sh"
  export RESTORE_VERIFY_LOG="$fixture/restore-verify.log"
  export SYSTEMCTL_LOG="$fixture/systemctl.log"
  export PRODUCTION_CURL_COUNT="$fixture/production-curl-count"
  unset RUNUSER_DENY_READ SYSTEMCTL_FAIL FAIL_PREFLIGHT_HEALTH FAIL_FIRST_PRODUCTION_HEALTH FAIL_RESTORE_VERIFY
}

run_expect_failure() {
  local output="$1"
  shift
  if "$@" >"$output" 2>&1; then
    printf 'command unexpectedly succeeded: %s\n' "$*" >&2
    exit 1
  fi
}

original_path="$PATH"

setup_fixture missing-backup
unset BACKUP_FILE
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "BACKUP_FILE is required"
[[ ! -e "$SYSTEMCTL_LOG" ]]

setup_fixture failed-restore-verification
export FAIL_RESTORE_VERIFY=true
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "backup restore verification failed"
[[ ! -e "$SYSTEMCTL_LOG" ]]

setup_fixture unreadable-config
export RUNUSER_DENY_READ="$PREFLIGHT_ENV_FILE"
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "preflight environment is unreadable"
[[ ! -e "$SYSTEMCTL_LOG" ]]

setup_fixture non-executable
chmod -x "$fixture/releases/new/sub2api"
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "candidate binary is not executable"
[[ "$(readlink -f "$CURRENT_LINK")" == "$fixture/releases/old" ]]

setup_fixture invalid-port
export PREFLIGHT_PORT=70000
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "PREFLIGHT_PORT must be at most 65535"
[[ ! -e "$SYSTEMCTL_LOG" ]]

setup_fixture shared-environment
export PREFLIGHT_ENV_FILE="$PRODUCTION_ENV_FILE"
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "preflight must use an isolated environment file"
[[ ! -e "$SYSTEMCTL_LOG" ]]

setup_fixture failed-preflight
export FAIL_PREFLIGHT_HEALTH=true
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "candidate failed isolated startup health check"
[[ "$(readlink -f "$CURRENT_LINK")" == "$fixture/releases/old" ]]
[[ ! -e "$SYSTEMCTL_LOG" ]]

setup_fixture rollback
export FAIL_FIRST_PRODUCTION_HEALTH=true
run_expect_failure "$fixture/output" "$release_script" "$fixture/releases/new"
assert_contains "$fixture/output" "previous release restored"
[[ "$(readlink -f "$CURRENT_LINK")" == "$fixture/releases/old" ]]
[[ "$(wc -l <"$SYSTEMCTL_LOG" | tr -d ' ')" == "2" ]]

setup_fixture success
"$release_script" "$fixture/releases/new" >"$fixture/output" 2>&1
assert_contains "$fixture/output" "release activated"
[[ "$(readlink -f "$CURRENT_LINK")" == "$fixture/releases/new" ]]
[[ "$(wc -l <"$SYSTEMCTL_LOG" | tr -d ' ')" == "1" ]]

echo "preflight-systemd-release tests passed"
