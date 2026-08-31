-- Give each user payment submission one durable identity so browser retries,
-- double-clicks and ambiguous network responses cannot create a second order.

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128),
    ADD COLUMN IF NOT EXISTS idempotency_request_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS idempotency_response JSONB;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_user_idempotency_key
    ON payment_orders(user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

COMMENT ON COLUMN payment_orders.idempotency_key
    IS 'Client-generated payment submission identity, unique per user when present';
COMMENT ON COLUMN payment_orders.idempotency_request_hash
    IS 'SHA-256 fingerprint of stable business inputs for conflict detection';
COMMENT ON COLUMN payment_orders.idempotency_response
    IS 'Successful create-order response used for exact client replay';
