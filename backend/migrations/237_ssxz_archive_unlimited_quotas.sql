-- Native 238 deletes unlimited quota rows. Some still have usage/window history.
-- Preserve the complete rows BEFORE that cleanup; do not restore stale windows
-- into live quota accounting automatically. No FK: the snapshot must survive a
-- later group/user cleanup until a separately approved retention decision.
CREATE TABLE IF NOT EXISTS ssxz_v027_unlimited_quota_archive (
    quota_id BIGINT PRIMARY KEY,
    row_data JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
REVOKE ALL ON ssxz_v027_unlimited_quota_archive FROM PUBLIC;

LOCK TABLE user_platform_quotas IN SHARE ROW EXCLUSIVE MODE;
INSERT INTO ssxz_v027_unlimited_quota_archive (quota_id, row_data)
    SELECT id, to_jsonb(q)
      FROM user_platform_quotas q
     WHERE daily_limit_usd IS NULL
       AND weekly_limit_usd IS NULL
       AND monthly_limit_usd IS NULL;

COMMENT ON TABLE ssxz_v027_unlimited_quota_archive IS
    'Private immutable snapshot before native 238 unlimited quota cleanup; no public API';
