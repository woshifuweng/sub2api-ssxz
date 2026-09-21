-- Aggregate-only comparison before/after the closed-write migration window.
-- Contains no API keys, emails, prompts or credential dumps.
SELECT json_build_object(
 'users', (SELECT count(*) FROM users),
 'money', (SELECT md5(string_agg(concat_ws('|',id,balance,frozen_balance,total_recharged),',' ORDER BY id)) FROM users),
 'key_permissions', (SELECT md5(string_agg(concat_ws('|',id,user_id,group_id,group_ids,status,allowed_models,deleted_at),',' ORDER BY id)) FROM api_keys),
 'grants', (SELECT md5(string_agg(concat_ws('|',user_id,group_id),',' ORDER BY user_id,group_id)) FROM user_allowed_groups),
 'customer_rates', (SELECT md5(string_agg(concat_ws('|',user_id,group_id,rate_multiplier),',' ORDER BY user_id,group_id)) FROM user_group_rate_multipliers),
 'group_permissions', (SELECT md5(string_agg(concat_ws('|',id,status,is_exclusive,rate_multiplier,models_list_config),',' ORDER BY id)) FROM groups),
 'ledger_rows', (SELECT count(*) FROM account_balance_ledger),
 'ledger_delta', (SELECT coalesce(sum(amount_delta),0)::text FROM account_balance_ledger),
 'limited_quota_rows', (SELECT count(*) FROM user_platform_quotas WHERE daily_limit_usd IS NOT NULL OR weekly_limit_usd IS NOT NULL OR monthly_limit_usd IS NOT NULL),
 'limited_quota_digest', (SELECT md5(string_agg(to_jsonb(q)::text,',' ORDER BY id)) FROM user_platform_quotas q WHERE daily_limit_usd IS NOT NULL OR weekly_limit_usd IS NOT NULL OR monthly_limit_usd IS NOT NULL)
);
