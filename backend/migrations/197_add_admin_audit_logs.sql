-- Generic, append-only audit trail for privileged administrator mutations.
-- Sensitive values such as API keys and credentials must never be stored here.
CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    actor_user_id   BIGINT NOT NULL,
    actor_role      VARCHAR(32) NOT NULL DEFAULT 'admin',
    action          VARCHAR(64) NOT NULL,
    resource_type   VARCHAR(64) NOT NULL,
    resource_id     BIGINT NOT NULL,
    target_user_id  BIGINT,
    before_values   JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_values    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_created_at
    ON admin_audit_logs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_actor_created
    ON admin_audit_logs(actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_resource_created
    ON admin_audit_logs(resource_type, resource_id, created_at DESC);
