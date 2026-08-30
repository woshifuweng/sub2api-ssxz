-- Prevent duplicate balance-conversion requests when a client retries an
-- ambiguous response or a user clicks the submit action more than once.

ALTER TABLE affiliate_withdraw_requests
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128);

CREATE UNIQUE INDEX IF NOT EXISTS idx_affiliate_withdraw_requests_user_idempotency_key
    ON affiliate_withdraw_requests(user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

COMMENT ON COLUMN affiliate_withdraw_requests.idempotency_key
    IS 'Client-generated retry key, unique per reseller user when present';
