CREATE TABLE IF NOT EXISTS chat_conversations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_message_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS chatconversation_user_id
    ON chat_conversations (user_id);
CREATE INDEX IF NOT EXISTS chatconversation_user_id_status
    ON chat_conversations (user_id, status);
CREATE INDEX IF NOT EXISTS chatconversation_user_id_updated_at
    ON chat_conversations (user_id, updated_at);
CREATE INDEX IF NOT EXISTS chatconversation_user_id_last_message_at
    ON chat_conversations (user_id, last_message_at);

CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id BIGINT NOT NULL,
    conversation_id BIGINT NOT NULL,
    message_type VARCHAR(32) NOT NULL,
    role VARCHAR(32) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    task_id BIGINT NULL,
    asset_id BIGINT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS chatmessage_user_id
    ON chat_messages (user_id);
CREATE INDEX IF NOT EXISTS chatmessage_user_id_conversation_id
    ON chat_messages (user_id, conversation_id);
CREATE INDEX IF NOT EXISTS chatmessage_conversation_id
    ON chat_messages (conversation_id);
CREATE INDEX IF NOT EXISTS chatmessage_conversation_id_id
    ON chat_messages (conversation_id, id);
CREATE INDEX IF NOT EXISTS chatmessage_conversation_id_created_at
    ON chat_messages (conversation_id, created_at);
CREATE INDEX IF NOT EXISTS chatmessage_task_id
    ON chat_messages (task_id);
CREATE INDEX IF NOT EXISTS chatmessage_asset_id
    ON chat_messages (asset_id);
CREATE INDEX IF NOT EXISTS chatmessage_task_id_id
    ON chat_messages (task_id, id);

CREATE TABLE IF NOT EXISTS chat_image_tasks (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id BIGINT NOT NULL,
    conversation_id BIGINT NOT NULL,
    message_id BIGINT NULL,
    reference_asset_id BIGINT NULL,
    result_asset_id BIGINT NULL,
    request_id VARCHAR(128) NOT NULL,
    idempotency_key_hash VARCHAR(128) NULL,
    task_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    billing_status VARCHAR(32) NOT NULL DEFAULT 'not_billed',
    actual_cost DECIMAL(20, 10) NULL,
    completed_at TIMESTAMPTZ NULL,
    error_code VARCHAR(64) NULL,
    error_message TEXT NULL,
    prompt TEXT NOT NULL DEFAULT '',
    enhanced_prompt TEXT NOT NULL DEFAULT '',
    model VARCHAR(128) NOT NULL DEFAULT '',
    ratio VARCHAR(64) NOT NULL DEFAULT '',
    purpose VARCHAR(128) NOT NULL DEFAULT '',
    style VARCHAR(128) NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS chatimagetask_request_id_key
    ON chat_image_tasks (request_id);
CREATE INDEX IF NOT EXISTS chatimagetask_user_id
    ON chat_image_tasks (user_id);
CREATE INDEX IF NOT EXISTS chatimagetask_conversation_id
    ON chat_image_tasks (conversation_id);
CREATE INDEX IF NOT EXISTS chatimagetask_message_id
    ON chat_image_tasks (message_id);
CREATE INDEX IF NOT EXISTS chatimagetask_message_id_id
    ON chat_image_tasks (message_id, id);
CREATE INDEX IF NOT EXISTS chatimagetask_reference_asset_id
    ON chat_image_tasks (reference_asset_id);
CREATE INDEX IF NOT EXISTS chatimagetask_result_asset_id
    ON chat_image_tasks (result_asset_id);
CREATE INDEX IF NOT EXISTS chatimagetask_task_status
    ON chat_image_tasks (task_status);
CREATE INDEX IF NOT EXISTS chatimagetask_task_status_updated_at
    ON chat_image_tasks (task_status, updated_at);
CREATE INDEX IF NOT EXISTS chatimagetask_billing_status
    ON chat_image_tasks (billing_status);
CREATE INDEX IF NOT EXISTS chatimagetask_user_id_conversation_id
    ON chat_image_tasks (user_id, conversation_id);
CREATE INDEX IF NOT EXISTS chatimagetask_user_id_request_id
    ON chat_image_tasks (user_id, request_id);
CREATE UNIQUE INDEX IF NOT EXISTS chatimagetask_user_id_idempotency_key_hash
    ON chat_image_tasks (user_id, idempotency_key_hash);

CREATE TABLE IF NOT EXISTS chat_assets (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id BIGINT NOT NULL,
    conversation_id BIGINT NULL,
    message_id BIGINT NULL,
    task_id BIGINT NULL,
    asset_kind VARCHAR(32) NOT NULL,
    source_type VARCHAR(32) NOT NULL DEFAULT 'user_upload',
    asset_role VARCHAR(64) NOT NULL DEFAULT 'unknown',
    storage_provider VARCHAR(64) NOT NULL DEFAULT 'pending',
    storage_key VARCHAR(512) NOT NULL DEFAULT '',
    url VARCHAR(2048) NOT NULL DEFAULT '',
    preview_url VARCHAR(2048) NULL,
    original_name VARCHAR(512) NOT NULL DEFAULT '',
    content_type VARCHAR(128) NOT NULL DEFAULT '',
    byte_size BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'registered'
);

CREATE INDEX IF NOT EXISTS chatasset_user_id
    ON chat_assets (user_id);
CREATE INDEX IF NOT EXISTS chatasset_conversation_id
    ON chat_assets (conversation_id);
CREATE INDEX IF NOT EXISTS chatasset_message_id
    ON chat_assets (message_id);
CREATE INDEX IF NOT EXISTS chatasset_task_id
    ON chat_assets (task_id);
CREATE INDEX IF NOT EXISTS chatasset_user_id_conversation_id
    ON chat_assets (user_id, conversation_id);
CREATE INDEX IF NOT EXISTS chatasset_user_id_storage_key
    ON chat_assets (user_id, storage_key);
