#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
restore_script="$script_dir/verify-backup-restore.sh"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-restore-test.XXXXXX")"
trap 'rm -rf -- "$test_root"' EXIT

assert_contains() {
  local file="$1" expected="$2"
  grep -Fq -- "$expected" "$file" || {
    printf 'expected %s to contain: %s\n' "$file" "$expected" >&2
    cat "$file" >&2
    exit 1
  }
}

assert_nonempty() {
  local file="$1" output="${2:-}"
  [[ -s "$file" ]] || {
    printf 'expected non-empty file: %s\n' "$file" >&2
    [[ -n "$output" && -f "$output" ]] && cat "$output" >&2
    exit 1
  }
}

run_expect_failure() {
  local output="$1"
  shift
  if "$@" >"$output" 2>&1; then
    printf 'command unexpectedly succeeded: %s\n' "$*" >&2
    exit 1
  fi
}

setup_fixture() {
  local name="$1"
  fixture="$test_root/$name"
  mkdir -p "$fixture/bin"
  printf 'PGDMP fake custom archive\n' >"$fixture/backup.dump"
  : >"$fixture/created.log"
  : >"$fixture/dropped.log"
  : >"$fixture/restored.log"

  cat >"$fixture/bin/id" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  -un) printf 'test-user\n' ;;
  -u) printf '1000\n' ;;
  *) printf '1000\n' ;;
esac
EOF

  cat >"$fixture/bin/pg_restore" <<'EOF'
#!/usr/bin/env bash
if [[ " $* " == *" --list "* ]]; then
  [[ "${INVALID_BACKUP:-false}" != "true" ]]
  exit
fi
printf '%s\n' "$*" >>"$RESTORED_LOG"
[[ "${FAIL_RESTORE:-false}" != "true" ]]
EOF

  cat >"$fixture/bin/createdb" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$CREATED_LOG"
EOF

  cat >"$fixture/bin/dropdb" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$DROPPED_LOG"
EOF

  cat >"$fixture/bin/psql" <<'EOF'
#!/usr/bin/env bash
query=""
for ((index=1; index<=$#; index++)); do
  if [[ "${!index}" == "--command" ]]; then
    next=$((index + 1))
    query="${!next}"
  fi
done
if [[ "$query" == *"FROM pg_database"* ]]; then
  [[ "${DATABASE_EXISTS:-false}" == "true" ]] && printf '1\n'
elif [[ "$query" == *"to_regclass"* ]]; then
  table="${query#*public.}"
  table="${table%%\'*}"
  if [[ "${MISSING_TABLE:-}" != "$table" ]]; then printf '%s\n' "$table"; fi
elif [[ "$query" == *"FROM schema_migrations"* ]]; then
  [[ "${MISSING_MIGRATION:-false}" == "true" ]] && printf '0\n' || printf '1\n'
elif [[ "$query" == *"users="* ]]; then
  printf 'api_keys=3\ngroups=2\npayment_orders=0\nusage_logs=7\nusers=4\n'
fi
exit 0
EOF

  cat >"$fixture/bin/runuser" <<'EOF'
#!/usr/bin/env bash
while [[ $# -gt 0 ]]; do
  case "$1" in
    -u) shift 2 ;;
    --) shift; break ;;
    *) break ;;
  esac
done
exec "$@"
EOF

  chmod +x "$fixture/bin/"*
  export PATH="$fixture/bin:$original_path"
  export RESTORE_OS_USER="test-user"
  export RESTORE_DATABASE_PREFIX="sub2api_restore_verify"
  export RESTORE_DATABASE_NAME="sub2api_restore_verify_test_${name//-/_}"
  export MAX_BACKUP_AGE_HOURS=48
  export CREATED_LOG="$fixture/created.log"
  export DROPPED_LOG="$fixture/dropped.log"
  export RESTORED_LOG="$fixture/restored.log"
  unset INVALID_BACKUP FAIL_RESTORE DATABASE_EXISTS MISSING_TABLE MISSING_MIGRATION
}

original_path="$PATH"

setup_fixture missing
run_expect_failure "$fixture/output" "$restore_script" "$fixture/not-found.dump"
assert_contains "$fixture/output" "backup file not found"
[[ ! -s "$CREATED_LOG" ]]

setup_fixture empty
: >"$fixture/backup.dump"
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "backup file is empty"
[[ ! -s "$CREATED_LOG" ]]

setup_fixture stale
touch -d '3 hours ago' "$fixture/backup.dump"
export MAX_BACKUP_AGE_HOURS=1
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "backup is older than 1 hours"
[[ ! -s "$CREATED_LOG" ]]

setup_fixture invalid
export INVALID_BACKUP=true
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "not a valid PostgreSQL custom-format archive"
[[ ! -s "$CREATED_LOG" ]]

setup_fixture unsafe-name
export RESTORE_DATABASE_NAME="sub2api"
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "must start with sub2api_restore_verify_"
[[ ! -s "$CREATED_LOG" ]]

setup_fixture existing
export DATABASE_EXISTS=true
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "already exists; refusing to reuse"
[[ ! -s "$CREATED_LOG" ]]

setup_fixture restore-failure
export FAIL_RESTORE=true
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_nonempty "$CREATED_LOG" "$fixture/output"
assert_nonempty "$DROPPED_LOG" "$fixture/output"

setup_fixture missing-table
export MISSING_TABLE=users
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "required table missing after restore: users"
assert_nonempty "$DROPPED_LOG" "$fixture/output"

setup_fixture missing-migration
export MISSING_MIGRATION=true
run_expect_failure "$fixture/output" "$restore_script" "$fixture/backup.dump"
assert_contains "$fixture/output" "required migration missing after restore"
assert_nonempty "$DROPPED_LOG" "$fixture/output"

setup_fixture success
"$restore_script" "$fixture/backup.dump" >"$fixture/output" 2>&1
assert_contains "$fixture/output" "backup restore verification passed"
assert_contains "$fixture/output" "row count: users=4"
assert_nonempty "$CREATED_LOG" "$fixture/output"
assert_nonempty "$RESTORED_LOG" "$fixture/output"
assert_nonempty "$DROPPED_LOG" "$fixture/output"

echo "verify-backup-restore tests passed"
