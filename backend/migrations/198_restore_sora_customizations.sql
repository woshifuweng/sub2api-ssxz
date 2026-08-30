-- Restore the production Sora customization after upstream migration 090 removes it.
CREATE TABLE IF NOT EXISTS sora_accounts (
    account_id BIGINT PRIMARY KEY,
    credentials JSONB NOT NULL DEFAULT '{}'::jsonb,
    extra JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sora_accounts_account_id
        FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sora_generations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT,
    model VARCHAR(64) NOT NULL,
    prompt TEXT NOT NULL DEFAULT '',
    media_type VARCHAR(16) NOT NULL DEFAULT 'video',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    media_url TEXT NOT NULL DEFAULT '',
    media_urls JSONB,
    file_size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_type VARCHAR(16) NOT NULL DEFAULT 'none',
    s3_object_keys JSONB,
    upstream_task_id VARCHAR(128) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sora_gen_user_created
    ON sora_generations(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sora_gen_user_status
    ON sora_generations(user_id, status);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS sora_storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sora_storage_used_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS sora_image_price_360 DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS sora_image_price_540 DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS sora_video_price_per_request DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS sora_video_price_per_request_hd DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS sora_storage_quota_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS media_type VARCHAR(16);
