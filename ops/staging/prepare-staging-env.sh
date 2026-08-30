#!/usr/bin/env bash
set -euo pipefail

env_file="${1:-/etc/sub2api/staging.env}"

if [[ ! -f "$env_file" ]]; then
  echo "staging environment file not found: $env_file" >&2
  exit 1
fi

set_env() {
  local key="$1"
  local value="$2"
  local temp_file
  temp_file="$(mktemp "${env_file}.XXXXXX")"
  awk -v target="$key" -v replacement="$value" '
    BEGIN { found = 0 }
    index($0, target "=") == 1 {
      if (!found) {
        print target "=" replacement
        found = 1
      }
      next
    }
    { print }
    END {
      if (!found) print target "=" replacement
    }
  ' "$env_file" > "$temp_file"
  chmod 600 "$temp_file"
  mv "$temp_file" "$env_file"
}

unset_env() {
  local key="$1"
  local temp_file
  temp_file="$(mktemp "${env_file}.XXXXXX")"
  awk -v target="$key" 'index($0, target "=") != 1 { print }' "$env_file" > "$temp_file"
  chmod 600 "$temp_file"
  mv "$temp_file" "$env_file"
}

random_hex() {
  openssl rand -hex 32
}

set_env LOG_ENV staging
set_env APP_RUNTIME_ROLE staging
set_env BACKGROUND_JOBS_ENABLED false
set_env SCHEDULERS_ENABLED false
set_env STAGING_API_ONLY true
set_env TURNSTILE_REQUIRED false

set_env JWT_SECRET "$(random_hex)"
set_env GATEWAY_SORA_MEDIA_SIGNING_KEY "$(random_hex)"
set_env PAYMENT_RESUME_SIGNING_KEY "$(random_hex)"
unset_env nGATEWAY_SORA_MEDIA_SIGNING_KEY

set_env DEEPSEEK_API_KEY ""
set_env JINA_API_KEY ""

set_env WORKSPACE_AVAILABLE_CHANNELS_STAGING_OVERRIDE_ENABLED false
set_env WORKSPACE_SORA_CLIENT_STAGING_OVERRIDE_ENABLED false
set_env WORKSPACE_IMAGE_EXECUTION_ENABLED false
set_env WORKSPACE_IMAGE_EXECUTION_FAKE_PROVIDER_ENABLED false
set_env WORKSPACE_IMAGE_EXECUTION_KILL_SWITCH true
set_env WORKSPACE_IMAGE_REAL_PROVIDER_ENABLED false
set_env WORKSPACE_IMAGE_REAL_PROVIDER_KILL_SWITCH true
set_env WORKSPACE_TEXT_PROVIDER_ENABLED false
set_env WORKSPACE_TEXT_PROVIDER_KILL_SWITCH true
set_env WORKSPACE_TEXT_PROVIDER_BETA_ALLOWLIST_ENABLED false
set_env WORKSPACE_TEXT_PROVIDER_BILLING_ELIGIBILITY_KNOWN true
set_env WORKSPACE_TEXT_PROVIDER_BILLING_ELIGIBLE false
set_env WORKSPACE_TEXT_PROVIDER_FAILURE_POLICY closed
set_env WORKSPACE_WEB_SEARCH_ENABLED false
set_env WORKSPACE_WEB_SEARCH_KILL_SWITCH true

chmod 600 "$env_file"
echo "staging environment rotated and passive-provider switches enforced"
