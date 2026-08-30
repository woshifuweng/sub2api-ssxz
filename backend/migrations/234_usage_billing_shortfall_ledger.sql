ALTER TABLE account_balance_ledger
    DROP CONSTRAINT IF EXISTS chk_abl_event_type;

ALTER TABLE account_balance_ledger
    ADD CONSTRAINT chk_abl_event_type CHECK (event_type IN (
        'admin_credit',
        'admin_debit',
        'admin_set',
        'redeem_code',
        'admin_redeem',
        'payment',
        'refund',
        'affiliate',
        'usage_shortfall'
    ));
