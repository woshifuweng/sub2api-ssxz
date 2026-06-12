ALTER TABLE workspace_messages
    DROP CONSTRAINT IF EXISTS workspace_messages_message_type_check;

ALTER TABLE workspace_messages
    ADD CONSTRAINT workspace_messages_message_type_check
    CHECK (message_type IN ('text', 'image'));

ALTER TABLE workspace_messages
    DROP CONSTRAINT IF EXISTS workspace_messages_intent_check;

ALTER TABLE workspace_messages
    ADD CONSTRAINT workspace_messages_intent_check
    CHECK (intent IN ('chat', 'image_generation'));
