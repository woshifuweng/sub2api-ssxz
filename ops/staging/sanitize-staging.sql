\set ON_ERROR_STOP on

BEGIN;

-- Preserve only the two dedicated staging identities. All cloned production users
-- remain as UI fixtures but cannot authenticate and no longer contain PII.
UPDATE users
SET email = 'staging-user-' || id || '@example.invalid',
    password_hash = '!staging-disabled-' || md5(id::text || clock_timestamp()::text),
    role = 'user',
    balance = 0,
    frozen_balance = 0,
    status = 'disabled',
    username = 'staging-user-' || id,
    notes = 'anonymized for isolated staging',
    wechat = NULL,
    totp_secret_encrypted = NULL,
    totp_enabled = false,
    totp_enabled_at = NULL,
    balance_notify_enabled = false,
    balance_notify_extra_emails = '[]',
    total_recharged = 0,
    last_login_at = NULL,
    last_active_at = NULL
WHERE lower(email) NOT IN (lower(:'staging_admin_email'), lower(:'staging_beta_email'));

UPDATE users
SET password_hash = CASE
      WHEN lower(email) = lower(:'staging_admin_email') THEN :'staging_admin_password_hash'
      ELSE :'staging_beta_password_hash'
    END,
    username = CASE
      WHEN lower(email) = lower(:'staging_admin_email') THEN 'staging-admin'
      ELSE 'staging-beta'
    END,
    notes = '',
    wechat = NULL,
    totp_secret_encrypted = NULL,
    totp_enabled = false,
    totp_enabled_at = NULL,
    balance_notify_enabled = false,
    balance_notify_extra_emails = '[]',
    last_login_at = NULL,
    last_active_at = NULL
WHERE lower(email) IN (lower(:'staging_admin_email'), lower(:'staging_beta_email'));

-- Every copied API key is rotated. Only keys owned by the two staging identities
-- remain active, making overlap with production impossible.
UPDATE api_keys
SET key = 'sk-staging-' || lpad(id::text, 8, '0') || '-' || md5(id::text || clock_timestamp()::text),
    status = CASE
      WHEN deleted_at IS NULL AND user_id IN (
        SELECT id FROM users
        WHERE lower(email) IN (lower(:'staging_admin_email'), lower(:'staging_beta_email'))
      ) THEN 'active'
      ELSE 'disabled'
    END,
    ip_whitelist = '[]'::jsonb,
    ip_blacklist = '[]'::jsonb,
    quota_used = 0,
    usage_5h = 0,
    usage_1d = 0,
    usage_7d = 0,
    window_5h_start = NULL,
    window_1d_start = NULL,
    window_7d_start = NULL,
    last_used_at = NULL;

-- Retain provider rows for layout and routing inspection, but remove all usable
-- credentials and scheduling state.
UPDATE accounts
SET credentials = '{}'::jsonb,
    extra = '{}'::jsonb,
    status = 'inactive',
    schedulable = false,
    error_message = 'disabled in isolated staging',
    last_used_at = NULL,
    rate_limited_at = NULL,
    rate_limit_reset_at = NULL,
    overload_until = NULL,
    session_window_start = NULL,
    session_window_end = NULL,
    session_window_status = NULL,
    temp_unschedulable_until = NULL,
    temp_unschedulable_reason = NULL,
    notes = 'credentials removed for isolated staging';

UPDATE proxies
SET host = '127.0.0.1',
    port = 9,
    username = NULL,
    password = NULL,
    status = 'inactive',
    fallback_mode = 'none',
    backup_proxy_id = NULL;

UPDATE channel_monitors
SET endpoint = 'http://127.0.0.1:9',
    api_key_encrypted = '',
    enabled = false,
    extra_headers = '{}'::jsonb,
    body_override = NULL,
    last_checked_at = NULL;

UPDATE payment_provider_instances
SET config = '{}'::jsonb,
    enabled = false,
    refund_enabled = false,
    allow_user_refund = false;

UPDATE sora_accounts
SET access_token = '', refresh_token = '', session_token = NULL;

UPDATE security_secrets
SET value = 'staging-disabled-' || id || '-' || md5(id::text || clock_timestamp()::text);

UPDATE sub2api_plugin_installations
SET state = 'disabled',
    config_encrypted = '',
    enabled_at = NULL,
    last_error = '';

-- Remove all reusable login/session material.
TRUNCATE TABLE
  auth_identity_channels,
  auth_identities,
  identity_adoption_decisions,
  pending_auth_sessions,
  passkey_credentials,
  passkey_user_handles
RESTART IDENTITY;

TRUNCATE TABLE user_avatars RESTART IDENTITY;
TRUNCATE TABLE user_attribute_values RESTART IDENTITY;
TRUNCATE TABLE idempotency_records RESTART IDENTITY;
TRUNCATE TABLE scheduler_outbox RESTART IDENTITY;

-- Disable every external delivery or paid integration stored in settings.
DELETE FROM settings WHERE key LIKE 'notification_email_delivery:%';

UPDATE settings SET value = 'false', updated_at = now()
WHERE lower(key) IN (
  'email_verify_enabled',
  'password_reset_enabled',
  'payment_enabled',
  'payment_visible_method_alipay_enabled',
  'payment_visible_method_wxpay_enabled',
  'github_oauth_enabled',
  'google_oauth_enabled',
  'wechat_connect_enabled',
  'wechat_connect_mobile_enabled',
  'wechat_connect_mp_enabled',
  'wechat_connect_open_enabled',
  'turnstile_enabled'
);

UPDATE settings SET value = 'true', updated_at = now()
WHERE lower(key) = 'balance_payment_disabled';

UPDATE settings SET value = '[]', updated_at = now()
WHERE lower(key) IN ('enabled_payment_types', 'account_quota_notify_emails');

UPDATE settings SET value = '{}', updated_at = now()
WHERE lower(key) IN ('backup_s3_config', 'ops_email_notification_config');

UPDATE settings SET value = '', updated_at = now()
WHERE lower(key) IN (
  'smtp_host', 'smtp_username', 'smtp_password', 'smtp_from', 'smtp_from_name',
  'github_oauth_client_id', 'github_oauth_redirect_url', 'github_oauth_frontend_redirect_url',
  'google_oauth_client_id', 'google_oauth_redirect_url', 'google_oauth_frontend_redirect_url',
  'oidc_connect_token_url',
  'turnstile_secret_key', 'turnstile_site_key',
  'wechat_connect_app_id', 'wechat_connect_mobile_app_id', 'wechat_connect_mp_app_id',
  'wechat_connect_open_app_id', 'wechat_connect_redirect_url',
  'payment_help_image_url', 'alipay_mobile_precreate_deep_link'
);

UPDATE settings
SET value = 'staging-' || md5(clock_timestamp()::text || id::text), updated_at = now()
WHERE lower(key) = 'admin_api_key';

-- Redact cloned customer content while preserving enough rows for UI testing.
UPDATE workspace_conversations SET title = '[redacted staging conversation]';
UPDATE workspace_messages SET content = '[redacted in isolated staging]', metadata = '{}'::jsonb;
UPDATE chat_conversations SET title = '[redacted staging conversation]';
UPDATE chat_messages SET content = '[redacted in isolated staging]', metadata_json = '{}'::jsonb;
UPDATE chat_image_tasks
SET request_id = '',
    prompt = '[redacted in isolated staging]',
    enhanced_prompt = '[redacted in isolated staging]',
    error_message = '';
UPDATE chat_assets
SET storage_key = 'staging-redacted/' || id,
    url = '',
    preview_url = NULL,
    original_name = 'redacted';

UPDATE batch_image_events SET payload = '{}'::jsonb;
UPDATE batch_image_items
SET prompt_preview = '[redacted]', provider_source_object = NULL, error_message = '';
UPDATE batch_image_jobs
SET gcs_input_uri = NULL,
    gcs_output_uri = NULL,
    provider_input_ref = NULL,
    provider_output_ref = NULL,
    session_id = NULL,
    last_error_message = '';

UPDATE sora_generations
SET prompt = '[redacted in isolated staging]',
    media_url = '',
    media_urls = '[]'::jsonb,
    s3_object_keys = '[]'::jsonb,
    upstream_task_id = '',
    error_message = '';

UPDATE scheduled_test_results
SET response_text = '[redacted]', error_message = '';

UPDATE audit_logs
SET actor_email = 'staging-redacted@example.invalid',
    credential_masked = '',
    client_ip = '127.0.0.1',
    request_body = '';
UPDATE usage_logs SET ip_address = '127.0.0.1';
UPDATE ops_system_logs
SET message = '[redacted in isolated staging]',
    request_id = '',
    client_request_id = '',
    host = 'staging';
UPDATE ops_error_logs
SET client_ip = NULL,
    error_message = '[redacted in isolated staging]',
    upstream_error_message = NULL,
    upstream_error_detail = NULL,
    attempted_key_prefix = NULL,
    deleted_key_name = NULL,
    api_key_prefix = NULL;
WITH ranked_ingress_rejects AS (
  SELECT ctid,
         row_number() OVER (
           PARTITION BY bucket_start, reject_reason, route_family, protocol, user_id, api_key_id
           ORDER BY ctid
         ) AS synthetic_host
  FROM ops_ingress_reject_aggregates
)
UPDATE ops_ingress_reject_aggregates AS rejects
SET client_ip = '2001:db8::'::inet + ranked.synthetic_host
FROM ranked_ingress_rejects AS ranked
WHERE rejects.ctid = ranked.ctid;
UPDATE content_moderation_logs
SET user_email = 'staging-redacted@example.invalid',
    api_key_name = '[redacted]',
    email_sent = false;
UPDATE prompt_audit_events
SET username_snapshot = '[redacted]',
    user_email_snapshot = 'staging-redacted@example.invalid',
    api_key_name_snapshot = '[redacted]',
    full_prompt = '';
UPDATE prompt_audit_jobs
SET username_snapshot = '[redacted]',
    user_email_snapshot = 'staging-redacted@example.invalid',
    api_key_name_snapshot = '[redacted]',
    last_error_message = '';
UPDATE ops_alert_rules SET enabled = false, notify_email = false;
UPDATE ops_alert_events SET email_sent = false;

UPDATE deleted_api_key_audits
SET key = 'staging-redacted-' || id, key_name = '[redacted]';
UPDATE payment_orders
SET user_email = 'staging-redacted@example.invalid',
    user_name = '[redacted]',
    user_notes = '',
    recharge_code = '',
    payment_trade_no = '',
    pay_url = '',
    qr_code = '',
    qr_code_img = '',
    refund_reason = '',
    refund_request_reason = '',
    failed_reason = '',
    client_ip = '',
    src_host = '',
    src_url = '',
    provider_snapshot = '{}'::jsonb;
UPDATE payment_audit_logs SET detail = '', operator = '[redacted]';

UPDATE redeem_codes
SET code = 'STG-' || lpad(id::text, 8, '0') || '-' || left(md5(id::text || clock_timestamp()::text), 12),
    notes = 'rotated for isolated staging';
UPDATE promo_codes
SET code = 'STG-P-' || lpad(id::text, 8, '0') || '-' || left(md5(id::text || clock_timestamp()::text), 12),
    notes = 'rotated for isolated staging';
UPDATE user_affiliates
SET aff_code = 'STAGING-AFF-' || user_id,
    aff_code_custom = false;

UPDATE account_balance_ledger SET note = '[redacted in isolated staging]';
UPDATE user_affiliate_ledger SET notes = '[redacted in isolated staging]';
UPDATE affiliate_withdraw_requests SET note = '[redacted in isolated staging]';
UPDATE user_subscriptions SET notes = '[redacted in isolated staging]';
UPDATE user_reseller_roles
SET notes = '[redacted in isolated staging]', disabled_reason = '[redacted in isolated staging]';

-- Updates above may emit cache invalidations. A stopped, isolated staging instance
-- starts with an empty cache and does not need cloned or generated outbox entries.
TRUNCATE TABLE auth_cache_invalidation_outbox RESTART IDENTITY;

COMMIT;
