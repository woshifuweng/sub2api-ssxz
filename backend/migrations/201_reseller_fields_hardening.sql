-- Reseller fields and withdrawal state hardening.
-- Migration 200 is already deployed and must remain immutable.

ALTER TABLE user_reseller_roles
    ADD COLUMN IF NOT EXISTS commission_rate NUMERIC(5,4) NOT NULL DEFAULT 0;

ALTER TABLE user_reseller_roles
    DROP CONSTRAINT IF EXISTS user_reseller_roles_commission_rate_check;

ALTER TABLE user_reseller_roles
    ADD CONSTRAINT user_reseller_roles_commission_rate_check
    CHECK (commission_rate >= 0 AND commission_rate <= 1);

ALTER TABLE affiliate_withdraw_requests
    DROP CONSTRAINT IF EXISTS affiliate_withdraw_requests_status_check;

ALTER TABLE affiliate_withdraw_requests
    ADD CONSTRAINT affiliate_withdraw_requests_status_check
    CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'));

ALTER TABLE affiliate_withdraw_requests
    ALTER COLUMN method SET DEFAULT 'balance_transfer';

COMMENT ON COLUMN user_reseller_roles.commission_rate
    IS '代理佣金比例，范围 0 到 1';
COMMENT ON COLUMN affiliate_withdraw_requests.method
    IS 'balance_transfer（默认）/ alipay / wechat / bank / manual';
COMMENT ON COLUMN affiliate_withdraw_requests.status
    IS 'pending=待审批 approved=已批准 rejected=已拒绝 cancelled=用户已撤销';
