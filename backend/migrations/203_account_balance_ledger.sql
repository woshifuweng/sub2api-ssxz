CREATE TABLE IF NOT EXISTS account_balance_ledger (
    id             BIGSERIAL     PRIMARY KEY,
    user_id        BIGINT        NOT NULL,
    event_type     TEXT          NOT NULL,
    amount_delta   NUMERIC(20,8) NOT NULL,
    balance_before NUMERIC(20,8) NOT NULL,
    balance_after  NUMERIC(20,8) NOT NULL,
    actor_type     TEXT          NOT NULL,
    actor_id       BIGINT,
    source_type    TEXT,
    source_id      TEXT,
    note           TEXT          NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_abl_event_type CHECK (event_type IN (
        'admin_credit',
        'admin_debit',
        'admin_set',
        'redeem_code',
        'admin_redeem',
        'payment',
        'refund',
        'affiliate'
    )),
    CONSTRAINT chk_abl_actor_type CHECK (actor_type IN ('admin', 'user', 'system'))
);

CREATE INDEX IF NOT EXISTS idx_abl_user_created
    ON account_balance_ledger (user_id, created_at DESC);
