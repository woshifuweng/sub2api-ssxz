ALTER TABLE user_reseller_roles
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    ADD COLUMN IF NOT EXISTS manager_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS disabled_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS disabled_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_user_reseller_roles_status
    ON user_reseller_roles(status)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_reseller_roles_manager
    ON user_reseller_roles(manager_id)
    WHERE revoked_at IS NULL;

UPDATE user_reseller_roles AS child
SET manager_id = child.granted_by
FROM user_reseller_roles AS manager_row
WHERE child.role = 'agent'
  AND child.revoked_at IS NULL
  AND child.manager_id IS NULL
  AND child.granted_by = manager_row.user_id
  AND manager_row.role = 'agent_manager'
  AND manager_row.status = 'active'
  AND manager_row.revoked_at IS NULL;
