#!/usr/bin/env bash
set -euo pipefail

env_file="${1:-/etc/sub2api/staging.env}"
prod_db="${2:-sub2api}"

set -a
# shellcheck disable=SC1090
. "$env_file"
set +a

if [[ -z "${DATABASE_DBNAME:-}" || "$DATABASE_DBNAME" == "$prod_db" ]]; then
  echo "invalid staging database selection" >&2
  exit 1
fi

temp_dir="$(mktemp -d /tmp/sub2api-staging-verify.XXXXXX)"
trap 'rm -rf -- "$temp_dir"' EXIT

hash_query() {
  local db="$1"
  local sql="$2"
  sudo -u postgres psql -P pager=off -d "$db" -Atqc "$sql" | sort -u
}

count_query() {
  local sql="$1"
  sudo -u postgres psql -P pager=off -v ON_ERROR_STOP=1 -d "$DATABASE_DBNAME" -Atqc "$sql"
}

hash_query "$prod_db" "SELECT md5(key) FROM api_keys WHERE deleted_at IS NULL" > "$temp_dir/prod-api-keys"
hash_query "$DATABASE_DBNAME" "SELECT md5(key) FROM api_keys WHERE deleted_at IS NULL" > "$temp_dir/staging-api-keys"
hash_query "$prod_db" "SELECT md5(password_hash) FROM users WHERE deleted_at IS NULL" > "$temp_dir/prod-passwords"
hash_query "$DATABASE_DBNAME" "SELECT md5(password_hash) FROM users WHERE deleted_at IS NULL" > "$temp_dir/staging-passwords"

api_overlap="$(comm -12 "$temp_dir/prod-api-keys" "$temp_dir/staging-api-keys" | wc -l)"
password_overlap="$(comm -12 "$temp_dir/prod-passwords" "$temp_dir/staging-passwords" | wc -l)"

active_accounts="$(count_query "SELECT count(*) FROM accounts WHERE status = 'active' OR schedulable")"
credential_accounts="$(count_query "
  SELECT count(*)
  FROM accounts
  WHERE credentials <> '{}'::jsonb
     OR (extra - 'openai_long_context_billing_enabled') <> '{}'::jsonb
")"
active_non_test_keys="$(count_query "
  SELECT count(*)
  FROM api_keys k
  JOIN users u ON u.id = k.user_id
  WHERE k.status = 'active'
    AND lower(u.email) NOT IN (lower('$STAGING_ADMIN_EMAIL'), lower('$STAGING_BETA_EMAIL'))
")"
auth_sessions="$(count_query "
  SELECT (SELECT count(*) FROM pending_auth_sessions)
       + (SELECT count(*) FROM auth_identities)
       + (SELECT count(*) FROM passkey_credentials)
")"
provider_configs="$(count_query "
  SELECT (SELECT count(*) FROM payment_provider_instances WHERE enabled OR config <> '{}')
       + (SELECT count(*) FROM channel_monitors WHERE enabled OR api_key_encrypted <> '')
       + (SELECT count(*) FROM sora_accounts WHERE access_token <> '' OR refresh_token <> '')
")"

printf 'api_key_overlap=%s\n' "$api_overlap"
printf 'password_hash_overlap=%s\n' "$password_overlap"
printf 'active_accounts=%s\n' "$active_accounts"
printf 'credential_accounts=%s\n' "$credential_accounts"
printf 'active_non_test_keys=%s\n' "$active_non_test_keys"
printf 'auth_session_material=%s\n' "$auth_sessions"
printf 'enabled_provider_configs=%s\n' "$provider_configs"

if (( api_overlap != 0 || password_overlap != 0 || active_accounts != 0 || credential_accounts != 0 || active_non_test_keys != 0 || auth_sessions != 0 || provider_configs != 0 )); then
  echo "staging isolation verification failed" >&2
  exit 1
fi

echo "staging isolation verification passed"
