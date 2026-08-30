#!/usr/bin/env bash
set -euo pipefail

backup_file="${1:-}"
maintenance_database="${MAINTENANCE_DATABASE:-postgres}"
restore_database_prefix="${RESTORE_DATABASE_PREFIX:-sub2api_restore_verify}"
restore_database_name="${RESTORE_DATABASE_NAME:-}"
restore_os_user="${RESTORE_OS_USER:-postgres}"
max_backup_age_hours="${MAX_BACKUP_AGE_HOURS:-48}"
required_tables_csv="${REQUIRED_TABLES_CSV:-schema_migrations,users,api_keys,groups,usage_logs,payment_orders}"
required_migrations_csv="${REQUIRED_MIGRATIONS_CSV:-232_reseller_withdrawal_idempotency.sql,233_payment_order_idempotency.sql,234_usage_billing_shortfall_ledger.sql,235_align_signup_source_width.sql}"
forbidden_databases_csv="${FORBIDDEN_DATABASES_CSV:-postgres,template0,template1,sub2api,sub2api_staging,sub2api_production}"

log() { printf '[restore-verify] %s\n' "$*"; }
die() { printf '[restore-verify] ERROR: %s\n' "$*" >&2; exit 1; }

[[ -n "$backup_file" ]] || die "usage: $0 <postgres-custom-format-backup>"
[[ "$max_backup_age_hours" =~ ^[1-9][0-9]*$ ]] || die "MAX_BACKUP_AGE_HOURS must be a positive integer"
[[ "$maintenance_database" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]*$ ]] || die "MAINTENANCE_DATABASE contains unsafe characters"
[[ "$restore_database_prefix" =~ ^[a-z][a-z0-9_]{2,47}$ ]] || die "RESTORE_DATABASE_PREFIX must be a safe lowercase database prefix"

for command_name in realpath stat date pg_restore psql createdb dropdb id mktemp install chown rm rmdir; do
  command -v "$command_name" >/dev/null 2>&1 || die "required command not found: $command_name"
done
if [[ -n "$restore_os_user" && "$(id -un)" != "$restore_os_user" ]]; then
  command -v runuser >/dev/null 2>&1 || die "runuser is required to switch to RESTORE_OS_USER"
  [[ "$(id -u)" == "0" ]] || die "must run as root or RESTORE_OS_USER=$restore_os_user"
fi

backup_file="$(realpath -e "$backup_file" 2>/dev/null)" || die "backup file not found"
[[ -f "$backup_file" ]] || die "backup path is not a regular file: $backup_file"
[[ -s "$backup_file" ]] || die "backup file is empty: $backup_file"

backup_mtime="$(stat -c %Y "$backup_file")"
now_epoch="$(date +%s)"
[[ "$backup_mtime" =~ ^[0-9]+$ && "$now_epoch" =~ ^[0-9]+$ ]] || die "could not determine backup age"
backup_age_seconds=$((now_epoch - backup_mtime))
(( backup_age_seconds >= 0 )) || die "backup modification time is in the future"
(( backup_age_seconds <= max_backup_age_hours * 3600 )) || die "backup is older than ${max_backup_age_hours} hours"

restore_database_created=false
archive_stage_dir=""
archive_for_restore="$backup_file"

cleanup_restore_resources() {
  if [[ "$restore_database_created" == "true" ]]; then
    if run_db dropdb --if-exists --force --maintenance-db "$maintenance_database" "$restore_database_name"; then
      log "disposable database removed: $restore_database_name"
    else
      printf '[restore-verify] ERROR: failed to remove disposable database: %s\n' "$restore_database_name" >&2
    fi
  fi
  if [[ -n "$archive_stage_dir" ]]; then
    case "$archive_stage_dir" in
      "${TMPDIR:-/tmp}/sub2api-restore-archive."*) ;;
      *) printf '[restore-verify] ERROR: refusing unsafe archive staging cleanup: %s\n' "$archive_stage_dir" >&2; return ;;
    esac
    rm -f -- "$archive_for_restore"
    rmdir -- "$archive_stage_dir"
    log "temporary archive copy removed"
  fi
}
trap cleanup_restore_resources EXIT

run_db() {
  if [[ -z "$restore_os_user" || "$(id -un)" == "$restore_os_user" ]]; then
    "$@"
  else
    (
      cd "${TMPDIR:-/tmp}"
      runuser -u "$restore_os_user" -- "$@"
    )
  fi
}

pg_restore --list "$backup_file" >/dev/null || die "backup is not a valid PostgreSQL custom-format archive"
if ! run_db test -r "$backup_file"; then
  [[ "$(id -u)" == "0" && -n "$restore_os_user" ]] || die "backup is unreadable by database restore user"
  archive_stage_dir="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-restore-archive.XXXXXX")"
  chown "$restore_os_user" "$archive_stage_dir"
  archive_for_restore="$archive_stage_dir/backup.dump"
  install -o "$restore_os_user" -m 0400 "$backup_file" "$archive_for_restore"
  log "staged a temporary read-only archive copy for the database restore user"
fi
run_db pg_restore --list "$archive_for_restore" >/dev/null || die "database restore user cannot read the PostgreSQL archive"

if [[ -z "$restore_database_name" ]]; then
  restore_database_name="${restore_database_prefix}_$(date +%Y%m%d%H%M%S)_$$_${RANDOM}"
fi
[[ "$restore_database_name" =~ ^${restore_database_prefix}_[a-zA-Z0-9_]+$ ]] || \
  die "restore database name must start with ${restore_database_prefix}_ and contain only safe characters"
(( ${#restore_database_name} <= 63 )) || die "restore database name exceeds PostgreSQL's 63-character limit"

IFS=',' read -r -a forbidden_databases <<<"$forbidden_databases_csv"
for forbidden_database in "${forbidden_databases[@]}"; do
  forbidden_database="${forbidden_database//[[:space:]]/}"
  [[ -z "$forbidden_database" ]] && continue
  [[ "$restore_database_name" != "$forbidden_database" ]] || die "refusing forbidden restore database name: $restore_database_name"
done
[[ "$restore_database_name" != "$maintenance_database" ]] || die "restore database must differ from maintenance database"

existing_database="$(run_db psql --no-psqlrc --tuples-only --no-align --dbname "$maintenance_database" \
  --command "SELECT 1 FROM pg_database WHERE datname = '$restore_database_name'" | tr -d '[:space:]')"
[[ -z "$existing_database" ]] || die "restore database already exists; refusing to reuse it: $restore_database_name"

run_db createdb --maintenance-db "$maintenance_database" "$restore_database_name"
restore_database_created=true
log "disposable database created: $restore_database_name"

run_db pg_restore \
  --dbname "$restore_database_name" \
  --no-owner \
  --no-privileges \
  --exit-on-error \
  "$archive_for_restore"

IFS=',' read -r -a required_tables <<<"$required_tables_csv"
for required_table in "${required_tables[@]}"; do
  required_table="${required_table//[[:space:]]/}"
  [[ "$required_table" =~ ^[a-z_][a-z0-9_]*$ ]] || die "unsafe required table name: $required_table"
  table_name="$(run_db psql --no-psqlrc --tuples-only --no-align --dbname "$restore_database_name" \
    --command "SELECT COALESCE(to_regclass('public.$required_table')::text, '')" | tr -d '[:space:]')"
  [[ "$table_name" == "$required_table" || "$table_name" == "public.$required_table" ]] || \
    die "required table missing after restore: $required_table"
done

IFS=',' read -r -a required_migrations <<<"$required_migrations_csv"
for required_migration in "${required_migrations[@]}"; do
  required_migration="${required_migration## }"
  required_migration="${required_migration%% }"
  [[ "$required_migration" =~ ^[0-9]{3}_[A-Za-z0-9_]+\.sql$ ]] || die "unsafe required migration name: $required_migration"
  migration_count="$(run_db psql --no-psqlrc --tuples-only --no-align --dbname "$restore_database_name" \
    --command "SELECT COUNT(*) FROM schema_migrations WHERE filename = '$required_migration'" | tr -d '[:space:]')"
  [[ "$migration_count" == "1" ]] || die "required migration missing after restore: $required_migration"
done

row_counts="$(run_db psql --no-psqlrc --tuples-only --no-align --dbname "$restore_database_name" --command "
  SELECT 'users=' || COUNT(*) FROM users
  UNION ALL SELECT 'api_keys=' || COUNT(*) FROM api_keys
  UNION ALL SELECT 'groups=' || COUNT(*) FROM groups
  UNION ALL SELECT 'usage_logs=' || COUNT(*) FROM usage_logs
  UNION ALL SELECT 'payment_orders=' || COUNT(*) FROM payment_orders
  ORDER BY 1
")"

log "archive validated: $backup_file"
log "required schema and migrations verified"
while IFS= read -r row_count; do
  [[ -n "$row_count" ]] && log "row count: $row_count"
done <<<"$row_counts"
log "backup restore verification passed"
