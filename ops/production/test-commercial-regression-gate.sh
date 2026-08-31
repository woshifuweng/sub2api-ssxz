#!/bin/bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
gate_script="$script_dir/run-commercial-regression-gate.sh"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-commercial-gate-test.XXXXXX")"
trap 'rm -rf -- "$test_root"' EXIT
real_bash="$(command -v bash)"

assert_contains() {
  local file="$1" expected="$2"
  grep -Fq -- "$expected" "$file" || {
    printf 'expected %s to contain: %s\n' "$file" "$expected" >&2
    cat "$file" >&2
    exit 1
  }
}

make_fake_command() {
  local name="$1"
  cat >"$test_root/bin/$name" <<'EOF'
#!/bin/bash
printf '%s|%s|%s\n' "$(basename "$0")" "$PWD" "$*" >>"$COMMAND_LOG"
if [[ -n "${FAIL_MATCH:-}" && "$(basename "$0") $*" == *"$FAIL_MATCH"* ]]; then
  exit 23
fi
EOF
  chmod +x "$test_root/bin/$name"
}

mkdir -p "$test_root/bin" "$test_root/evidence" "$repo_root/backend" "$repo_root/frontend"
: >"$test_root/staging.env"
printf 'PGDMP commercial gate fixture\n' >"$test_root/backup.dump"
: >"$test_root/commands.log"

for command_name in go pnpm bash staging-verify restore-verify; do
  make_fake_command "$command_name"
done

export COMMAND_LOG="$test_root/commands.log"
export PATH="$test_root/bin:$PATH"
export GO_BIN=go
export PNPM_BIN=pnpm
export BASH_BIN=bash
export GATE_MODE=release
export GATE_PARALLELISM=3
export VITEST_MIN_WORKERS=1
export VITEST_MAX_WORKERS=2
export GATE_EVIDENCE_ROOT="$test_root/evidence"
export STAGING_ENV_FILE="$test_root/staging.env"
export PRODUCTION_DATABASE_NAME=sub2api
export BACKUP_FILE="$test_root/backup.dump"
export STAGING_VERIFY_SCRIPT="$test_root/bin/staging-verify"
export RESTORE_VERIFY_SCRIPT="$test_root/bin/restore-verify"
unset FAIL_MATCH

if ! "$real_bash" "$gate_script" >"$test_root/success.output" 2>&1; then
  echo "gate unexpectedly failed with all fake checks passing" >&2
  cat "$test_root/success.output" >&2
  exit 1
fi
assert_contains "$test_root/success.output" "PASS:"
assert_contains "$COMMAND_LOG" "go|$repo_root/backend|test -p 3 ./..."
assert_contains "$COMMAND_LOG" "go|$repo_root/backend|test -p 3 -tags=unit ./..."
assert_contains "$COMMAND_LOG" "go|$repo_root/backend|test -p 3 -tags=integration ./..."
assert_contains "$COMMAND_LOG" "go|$repo_root/backend|vet ./..."
assert_contains "$COMMAND_LOG" "pnpm|$repo_root|--dir frontend run lint:check"
assert_contains "$COMMAND_LOG" "pnpm|$repo_root|--dir frontend run typecheck"
assert_contains "$COMMAND_LOG" "pnpm|$repo_root|--dir frontend exec vitest run --minWorkers=1 --maxWorkers=2"
assert_contains "$COMMAND_LOG" "pnpm|$repo_root|--dir frontend run build"
assert_contains "$COMMAND_LOG" "go|$repo_root/backend|build -p 3 -tags=embed -trimpath -o"
assert_contains "$COMMAND_LOG" "./cmd/server"
assert_contains "$COMMAND_LOG" "bash|$repo_root|-n $script_dir/verify-backup-restore.sh"
assert_contains "$COMMAND_LOG" "bash|$repo_root|$script_dir/test-verify-backup-restore.sh"
assert_contains "$COMMAND_LOG" "staging-verify|$repo_root|$STAGING_ENV_FILE sub2api"
assert_contains "$COMMAND_LOG" "restore-verify|$repo_root|$BACKUP_FILE"

summary_file="$(find "$test_root/evidence" -name summary.tsv -type f -print -quit)"
metadata_file="$(find "$test_root/evidence" -name metadata.txt -type f -print -quit)"
[[ -n "$summary_file" && -n "$metadata_file" ]]
assert_contains "$summary_file" $'staging-isolation\tPASS'
assert_contains "$summary_file" $'backup-restore\tPASS'
assert_contains "$metadata_file" "gate_mode=release"
assert_contains "$metadata_file" "go_parallelism=3"

: >"$COMMAND_LOG"
export FAIL_MATCH="pnpm --dir frontend run typecheck"
if "$real_bash" "$gate_script" >"$test_root/failure.output" 2>&1; then
  echo "gate unexpectedly succeeded after an injected failure" >&2
  exit 1
fi
assert_contains "$test_root/failure.output" "frontend-typecheck failed"
if grep -Fq -- "vitest run" "$COMMAND_LOG"; then
  echo "gate continued to frontend tests after a failed typecheck" >&2
  exit 1
fi
if grep -Fq -- "staging-verify" "$COMMAND_LOG"; then
  echo "gate continued to operational checks after a source-check failure" >&2
  exit 1
fi

unset FAIL_MATCH
export GATE_MODE=invalid
if "$real_bash" "$gate_script" >"$test_root/invalid-mode.output" 2>&1; then
  echo "gate unexpectedly accepted an invalid mode" >&2
  exit 1
fi
assert_contains "$test_root/invalid-mode.output" "GATE_MODE must be source or release"

echo "commercial regression gate tests passed"
