CREATE TABLE IF NOT EXISTS workspace_conversations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_workspace_conversations_user_updated
    ON workspace_conversations (user_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_workspace_conversations_user_last_message
    ON workspace_conversations (user_id, last_message_at DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS workspace_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES workspace_conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_type VARCHAR(32) NOT NULL DEFAULT 'text',
    role VARCHAR(16) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    model VARCHAR(100) NOT NULL DEFAULT '',
    intent VARCHAR(32) NOT NULL DEFAULT 'chat',
    status VARCHAR(16) NOT NULL DEFAULT 'completed',
    metadata JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT workspace_messages_message_type_check CHECK (message_type = 'text'),
    CONSTRAINT workspace_messages_role_check CHECK (role IN ('user', 'assistant', 'system')),
    CONSTRAINT workspace_messages_intent_check CHECK (intent = 'chat'),
    CONSTRAINT workspace_messages_status_check CHECK (status IN ('pending', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_workspace_messages_conversation_created
    ON workspace_messages (conversation_id, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_workspace_messages_user_created
    ON workspace_messages (user_id, created_at DESC);
