-- Runtime monitor and identity/payment/subscription schema alignment.
-- This migration only creates missing runtime tables and indexes.

CREATE TABLE IF NOT EXISTS channel_monitor_request_templates (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    description VARCHAR(500) NULL DEFAULT '',
    extra_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    body_override_mode VARCHAR(10) NOT NULL DEFAULT 'off',
    body_override JSONB NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS channelmonitorrequesttemplate_provider_name
    ON channel_monitor_request_templates (provider, name);

CREATE TABLE IF NOT EXISTS channel_monitors (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    api_key_encrypted VARCHAR(255) NOT NULL,
    primary_model VARCHAR(200) NOT NULL,
    extra_models JSONB NOT NULL DEFAULT '[]'::jsonb,
    group_name VARCHAR(100) NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    interval_seconds INTEGER NOT NULL,
    last_checked_at TIMESTAMPTZ NULL,
    created_by BIGINT NOT NULL,
    extra_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    body_override_mode VARCHAR(10) NOT NULL DEFAULT 'off',
    body_override JSONB NULL,
    template_id BIGINT NULL,
    CONSTRAINT channel_monitors_channel_monitor_request_templates_request_template
        FOREIGN KEY (template_id)
        REFERENCES channel_monitor_request_templates (id)
        ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS channelmonitor_enabled_last_checked_at
    ON channel_monitors (enabled, last_checked_at);
CREATE INDEX IF NOT EXISTS channelmonitor_provider
    ON channel_monitors (provider);
CREATE INDEX IF NOT EXISTS channelmonitor_group_name
    ON channel_monitors (group_name);
CREATE INDEX IF NOT EXISTS channelmonitor_template_id
    ON channel_monitors (template_id);

CREATE TABLE IF NOT EXISTS channel_monitor_histories (
    id BIGSERIAL PRIMARY KEY,
    model VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL,
    latency_ms INTEGER NULL,
    ping_latency_ms INTEGER NULL,
    message VARCHAR(500) NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    monitor_id BIGINT NOT NULL,
    CONSTRAINT channel_monitor_histories_channel_monitors_history
        FOREIGN KEY (monitor_id)
        REFERENCES channel_monitors (id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS channelmonitorhistory_monitor_id_model_checked_at
    ON channel_monitor_histories (monitor_id, model, checked_at);
CREATE INDEX IF NOT EXISTS channelmonitorhistory_checked_at
    ON channel_monitor_histories (checked_at);

CREATE TABLE IF NOT EXISTS channel_monitor_daily_rollups (
    id BIGSERIAL PRIMARY KEY,
    model VARCHAR(200) NOT NULL,
    bucket_date DATE NOT NULL,
    total_checks INTEGER NOT NULL DEFAULT 0,
    ok_count INTEGER NOT NULL DEFAULT 0,
    operational_count INTEGER NOT NULL DEFAULT 0,
    degraded_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    sum_latency_ms BIGINT NOT NULL DEFAULT 0,
    count_latency INTEGER NOT NULL DEFAULT 0,
    sum_ping_latency_ms BIGINT NOT NULL DEFAULT 0,
    count_ping_latency INTEGER NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    monitor_id BIGINT NOT NULL,
    CONSTRAINT channel_monitor_daily_rollups_channel_monitors_daily_rollups
        FOREIGN KEY (monitor_id)
        REFERENCES channel_monitors (id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS channelmonitordailyrollup_monitor_id_model_bucket_date
    ON channel_monitor_daily_rollups (monitor_id, model, bucket_date);
CREATE INDEX IF NOT EXISTS channelmonitordailyrollup_bucket_date
    ON channel_monitor_daily_rollups (bucket_date);

CREATE TABLE IF NOT EXISTS channel_monitor_aggregation_watermark (
    id INTEGER PRIMARY KEY,
    last_aggregated_date DATE NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_provider_instances (
    id BIGSERIAL PRIMARY KEY,
    provider_key VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT '',
    config TEXT NOT NULL,
    supported_types VARCHAR(200) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    payment_mode VARCHAR(20) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    limits TEXT NOT NULL DEFAULT '',
    refund_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    allow_user_refund BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS paymentproviderinstance_provider_key
    ON payment_provider_instances (provider_key);
CREATE INDEX IF NOT EXISTS paymentproviderinstance_enabled
    ON payment_provider_instances (enabled);

CREATE TABLE IF NOT EXISTS auth_identities (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    provider_type VARCHAR(20) NOT NULL,
    provider_key TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    verified_at TIMESTAMPTZ NULL,
    issuer TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    user_id BIGINT NOT NULL,
    CONSTRAINT auth_identities_users_auth_identities
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE NO ACTION
);

CREATE UNIQUE INDEX IF NOT EXISTS authidentity_provider_type_provider_key_provider_subject
    ON auth_identities (provider_type, provider_key, provider_subject);
CREATE INDEX IF NOT EXISTS authidentity_user_id
    ON auth_identities (user_id);
CREATE INDEX IF NOT EXISTS authidentity_user_id_provider_type
    ON auth_identities (user_id, provider_type);

CREATE TABLE IF NOT EXISTS auth_identity_channels (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    provider_type VARCHAR(20) NOT NULL,
    provider_key TEXT NOT NULL,
    channel VARCHAR(20) NOT NULL,
    channel_app_id TEXT NOT NULL,
    channel_subject TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    identity_id BIGINT NOT NULL,
    CONSTRAINT auth_identity_channels_auth_identities_channels
        FOREIGN KEY (identity_id)
        REFERENCES auth_identities (id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS authidentitychannel_provider_type_provider_key_channel_channel_app_id_channel_subject
    ON auth_identity_channels (provider_type, provider_key, channel, channel_app_id, channel_subject);
CREATE INDEX IF NOT EXISTS authidentitychannel_identity_id
    ON auth_identity_channels (identity_id);

CREATE TABLE IF NOT EXISTS pending_auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_token VARCHAR(255) NOT NULL,
    intent VARCHAR(40) NOT NULL,
    provider_type VARCHAR(20) NOT NULL,
    provider_key TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    redirect_to TEXT NOT NULL DEFAULT '',
    resolved_email TEXT NOT NULL DEFAULT '',
    registration_password_hash TEXT NOT NULL DEFAULT '',
    upstream_identity_claims JSONB NOT NULL DEFAULT '{}'::jsonb,
    local_flow_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    browser_session_key TEXT NOT NULL DEFAULT '',
    completion_code_hash TEXT NOT NULL DEFAULT '',
    completion_code_expires_at TIMESTAMPTZ NULL,
    email_verified_at TIMESTAMPTZ NULL,
    password_verified_at TIMESTAMPTZ NULL,
    totp_verified_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    target_user_id BIGINT NULL,
    CONSTRAINT pending_auth_sessions_users_pending_auth_sessions
        FOREIGN KEY (target_user_id)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS pendingauthsession_session_token
    ON pending_auth_sessions (session_token);
CREATE INDEX IF NOT EXISTS pendingauthsession_target_user_id
    ON pending_auth_sessions (target_user_id);
CREATE INDEX IF NOT EXISTS pendingauthsession_expires_at
    ON pending_auth_sessions (expires_at);
CREATE INDEX IF NOT EXISTS pendingauthsession_provider_type_provider_key_provider_subject
    ON pending_auth_sessions (provider_type, provider_key, provider_subject);
CREATE INDEX IF NOT EXISTS pendingauthsession_completion_code_hash
    ON pending_auth_sessions (completion_code_hash);

CREATE TABLE IF NOT EXISTS identity_adoption_decisions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    adopt_display_name BOOLEAN NOT NULL DEFAULT FALSE,
    adopt_avatar BOOLEAN NOT NULL DEFAULT FALSE,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    identity_id BIGINT NULL,
    pending_auth_session_id BIGINT NOT NULL,
    CONSTRAINT identity_adoption_decisions_auth_identities_adoption_decisions
        FOREIGN KEY (identity_id)
        REFERENCES auth_identities (id)
        ON DELETE SET NULL,
    CONSTRAINT identity_adoption_decisions_pending_auth_sessions_adoption_decision
        FOREIGN KEY (pending_auth_session_id)
        REFERENCES pending_auth_sessions (id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS identityadoptiondecision_pending_auth_session_id
    ON identity_adoption_decisions (pending_auth_session_id);
CREATE INDEX IF NOT EXISTS identityadoptiondecision_identity_id
    ON identity_adoption_decisions (identity_id);

CREATE TABLE IF NOT EXISTS subscription_plans (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price DECIMAL(20,2) NOT NULL,
    original_price DECIMAL(20,2) NULL,
    validity_days INTEGER NOT NULL DEFAULT 30,
    validity_unit VARCHAR(10) NOT NULL DEFAULT 'day',
    features TEXT NOT NULL DEFAULT '',
    product_name VARCHAR(100) NOT NULL DEFAULT '',
    for_sale BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS subscriptionplan_group_id
    ON subscription_plans (group_id);
CREATE INDEX IF NOT EXISTS subscriptionplan_for_sale
    ON subscription_plans (for_sale);
