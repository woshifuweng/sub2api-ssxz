-- 三层代理体系：代理角色表 + 提现申请表
-- Phase 1: 2026-07-29

-- 代理角色表（谁是 agent/agent_manager，谁批的，什么时候）
CREATE TABLE IF NOT EXISTS user_reseller_roles (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL CHECK (role IN ('agent', 'agent_manager')),
    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,              -- NULL = 仍生效
    notes      TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_user_reseller_roles_role
    ON user_reseller_roles(role)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_reseller_roles_granted_by
    ON user_reseller_roles(granted_by)
    WHERE revoked_at IS NULL;

COMMENT ON TABLE  user_reseller_roles            IS '代理角色表 (agent=推广代理, agent_manager=运营管理员)';
COMMENT ON COLUMN user_reseller_roles.role       IS 'agent 或 agent_manager';
COMMENT ON COLUMN user_reseller_roles.granted_by IS '授权操作人（管理员或 owner）';
COMMENT ON COLUMN user_reseller_roles.revoked_at IS 'NULL 表示角色仍生效；有值表示已撤销';

-- 提现申请表（代理申请将 aff_quota 转入余额）
CREATE TABLE IF NOT EXISTS affiliate_withdraw_requests (
    id           BIGSERIAL   PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id),
    amount       DECIMAL(20,8) NOT NULL CHECK (amount > 0),
    method       VARCHAR(50) NOT NULL DEFAULT 'manual',
    account_info JSONB       NOT NULL DEFAULT '{}'
                    CHECK (jsonb_typeof(account_info) = 'object'),
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected')),
    note         TEXT        NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at  TIMESTAMPTZ,
    reviewed_by  BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_withdraw_requests_user_id
    ON affiliate_withdraw_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_withdraw_requests_status
    ON affiliate_withdraw_requests(status)
    WHERE status = 'pending';

COMMENT ON TABLE  affiliate_withdraw_requests         IS '代理提现申请';
COMMENT ON COLUMN affiliate_withdraw_requests.amount  IS '申请提取金额（从 user_affiliates.aff_quota 扣除）';
COMMENT ON COLUMN affiliate_withdraw_requests.method  IS 'alipay / wechat / bank / manual';
COMMENT ON COLUMN affiliate_withdraw_requests.status  IS 'pending=待审批 approved=已批准 rejected=已拒绝';
COMMENT ON COLUMN affiliate_withdraw_requests.note    IS '审批备注';
