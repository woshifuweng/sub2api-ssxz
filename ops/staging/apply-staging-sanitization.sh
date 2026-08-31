#!/usr/bin/env bash
set -euo pipefail

env_file="${1:-/etc/sub2api/staging.env}"
sql_file="${2:-/opt/sub2api/ops/staging/sanitize-staging.sql}"

if [[ ! -f "$env_file" || ! -f "$sql_file" ]]; then
  echo "required staging environment or SQL file is missing" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "$env_file"
set +a

required=(DATABASE_DBNAME STAGING_ADMIN_EMAIL STAGING_ADMIN_PASSWORD STAGING_BETA_EMAIL STAGING_BETA_PASSWORD)
for key in "${required[@]}"; do
  if [[ -z "${!key:-}" ]]; then
    echo "missing required staging variable: $key" >&2
    exit 1
  fi
done

identity_precheck="$(
  printf "%s\n" \
    "SELECT lower(:'admin') <> lower(:'beta')," \
    "(SELECT count(*) FROM users WHERE lower(email) = lower(:'admin'))," \
    "(SELECT count(*) FROM users WHERE lower(email) = lower(:'beta'));" \
  | sudo -u postgres psql -P pager=off -d "$DATABASE_DBNAME" \
      -v admin="$STAGING_ADMIN_EMAIL" -v beta="$STAGING_BETA_EMAIL" -At
)"

if [[ "$identity_precheck" != "t|1|1" ]]; then
  echo "staging identity precheck failed" >&2
  exit 1
fi

admin_hash="$(python3 - <<'PY'
import bcrypt
import os
print(bcrypt.hashpw(os.environ["STAGING_ADMIN_PASSWORD"].encode(), bcrypt.gensalt(rounds=12)).decode())
PY
)"
beta_hash="$(python3 - <<'PY'
import bcrypt
import os
print(bcrypt.hashpw(os.environ["STAGING_BETA_PASSWORD"].encode(), bcrypt.gensalt(rounds=12)).decode())
PY
)"

sudo -u postgres psql \
  -P pager=off \
  -v ON_ERROR_STOP=1 \
  -v staging_admin_email="$STAGING_ADMIN_EMAIL" \
  -v staging_beta_email="$STAGING_BETA_EMAIL" \
  -v staging_admin_password_hash="$admin_hash" \
  -v staging_beta_password_hash="$beta_hash" \
  -d "$DATABASE_DBNAME" \
  -f "$sql_file"

unset admin_hash beta_hash STAGING_ADMIN_PASSWORD STAGING_BETA_PASSWORD
echo "staging database sanitization completed"
