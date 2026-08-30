#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
mode="${GATE_MODE:-source}"
parallelism="${GATE_PARALLELISM:-4}"
vitest_min_workers="${VITEST_MIN_WORKERS:-1}"
vitest_max_workers="${VITEST_MAX_WORKERS:-$parallelism}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
evidence_root="${GATE_EVIDENCE_ROOT:-$repo_root/.codex-evidence}"
evidence_dir="$evidence_root/commercial-regression-$timestamp-$$"

go_bin="${GO_BIN:-go}"
pnpm_bin="${PNPM_BIN:-pnpm}"
bash_bin="${BASH_BIN:-bash}"
staging_verify_script="${STAGING_VERIFY_SCRIPT:-$repo_root/ops/staging/verify-staging-isolation.sh}"
restore_verify_script="${RESTORE_VERIFY_SCRIPT:-$script_dir/verify-backup-restore.sh}"
staging_env_file="${STAGING_ENV_FILE:-}"
production_database_name="${PRODUCTION_DATABASE_NAME:-sub2api}"
backup_file="${BACKUP_FILE:-}"

die() {
  printf '[commercial-gate] ERROR: %s\n' "$*" >&2
  exit 1
}

[[ "$mode" == "source" || "$mode" == "release" ]] || \
  die "GATE_MODE must be source or release"
[[ "$parallelism" =~ ^[1-9][0-9]*$ ]] || \
  die "GATE_PARALLELISM must be a positive integer"
[[ "$vitest_min_workers" =~ ^[1-9][0-9]*$ ]] || \
  die "VITEST_MIN_WORKERS must be a positive integer"
[[ "$vitest_max_workers" =~ ^[1-9][0-9]*$ ]] || \
  die "VITEST_MAX_WORKERS must be a positive integer"
(( vitest_min_workers <= vitest_max_workers )) || \
  die "VITEST_MIN_WORKERS cannot exceed VITEST_MAX_WORKERS"

for command_name in "$go_bin" "$pnpm_bin" "$bash_bin" date mkdir chmod tee git; do
  command -v "$command_name" >/dev/null 2>&1 || \
    die "required command not found: $command_name"
done

if [[ "$mode" == "release" ]]; then
  [[ -n "$staging_env_file" ]] || \
    die "STAGING_ENV_FILE is required in release mode"
  [[ -f "$staging_env_file" ]] || \
    die "staging environment file not found: $staging_env_file"
  [[ -n "$backup_file" ]] || \
    die "BACKUP_FILE is required in release mode"
  [[ -f "$backup_file" ]] || \
    die "backup file not found: $backup_file"
  [[ -x "$staging_verify_script" ]] || \
    die "staging isolation verifier is missing or not executable: $staging_verify_script"
  [[ -x "$restore_verify_script" ]] || \
    die "backup restore verifier is missing or not executable: $restore_verify_script"
fi

mkdir -p "$evidence_dir"
chmod 0700 "$evidence_dir"
summary_file="$evidence_dir/summary.tsv"
metadata_file="$evidence_dir/metadata.txt"
: >"$summary_file"
chmod 0600 "$summary_file"

commit="$(git -C "$repo_root" rev-parse HEAD 2>/dev/null || printf 'unknown')"
{
  printf 'gate_mode=%s\n' "$mode"
  printf 'started_at_utc=%s\n' "$timestamp"
  printf 'source_commit=%s\n' "$commit"
  printf 'go_parallelism=%s\n' "$parallelism"
  printf 'vitest_workers=%s..%s\n' "$vitest_min_workers" "$vitest_max_workers"
} >"$metadata_file"
chmod 0600 "$metadata_file"

step_number=0
run_step() {
  local name="$1"
  shift
  step_number=$((step_number + 1))
  local log_file="$evidence_dir/$(printf '%02d' "$step_number")-$name.log"
  printf '[commercial-gate] %s\n' "$name"
  if "$@" >"$log_file" 2>&1; then
    printf '%s\tPASS\t%s\n' "$name" "$(basename "$log_file")" | tee -a "$summary_file" >/dev/null
    chmod 0600 "$log_file"
    return 0
  else
    local status=$?
    printf '%s\tFAIL(%s)\t%s\n' "$name" "$status" "$(basename "$log_file")" | tee -a "$summary_file" >/dev/null
    chmod 0600 "$log_file"
    tail -n 80 "$log_file" >&2 || true
    die "$name failed; evidence: $log_file"
  fi
}

cd "$repo_root"
run_step shell-syntax-restore "$bash_bin" -n "$script_dir/verify-backup-restore.sh"
run_step shell-syntax-restore-test "$bash_bin" -n "$script_dir/test-verify-backup-restore.sh"
run_step shell-syntax-release "$bash_bin" -n "$script_dir/preflight-systemd-release.sh"
run_step shell-syntax-release-test "$bash_bin" -n "$script_dir/test-preflight-systemd-release.sh"
run_step shell-syntax-commercial-gate "$bash_bin" -n "$script_dir/run-commercial-regression-gate.sh"
run_step restore-verifier-self-test "$bash_bin" "$script_dir/test-verify-backup-restore.sh"
run_step guarded-release-self-test "$bash_bin" "$script_dir/test-preflight-systemd-release.sh"

cd "$repo_root/backend"
run_step backend-tests "$go_bin" test -p "$parallelism" ./...
run_step backend-unit-contract-tests "$go_bin" test -p "$parallelism" -tags=unit ./...
run_step backend-integration-tests "$go_bin" test -p "$parallelism" -tags=integration ./...
run_step backend-vet "$go_bin" vet ./...

cd "$repo_root"
run_step frontend-lint "$pnpm_bin" --dir frontend run lint:check
run_step frontend-typecheck "$pnpm_bin" --dir frontend run typecheck
run_step frontend-tests "$pnpm_bin" --dir frontend exec vitest run \
  --minWorkers="$vitest_min_workers" --maxWorkers="$vitest_max_workers"
run_step frontend-production-build "$pnpm_bin" --dir frontend run build

if [[ "$mode" == "release" ]]; then
  run_step staging-isolation "$staging_verify_script" \
    "$staging_env_file" "$production_database_name"
  run_step backup-restore "$restore_verify_script" "$backup_file"
fi

completed_at="$(date -u +%Y%m%dT%H%M%SZ)"
printf 'completed_at_utc=%s\n' "$completed_at" >>"$metadata_file"
printf '[commercial-gate] PASS: %s\n' "$evidence_dir"
