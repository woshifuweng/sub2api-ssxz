-- Run with psql -X -v ON_ERROR_STOP=1 against the explicitly selected database.
-- READ ONLY. Necessary schema/policy gate, not authorization to switch binaries.
-- Valid only during the closed-traffic, no-admin-write acceptance window. After
-- opening traffic, use a forward-compatible fix; do not rewind customer data.
BEGIN READ ONLY;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='groups'::regclass
        AND attname='models_list_config' AND NOT attisdropped)
       OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='groups'::regclass
        AND attname='model_allowlist' AND NOT attisdropped) THEN
        RAISE EXCEPTION 'rollback blocked: both legacy and native group columns are required';
    END IF;
    IF EXISTS (SELECT 1 FROM groups WHERE
        jsonb_typeof(model_allowlist) IS DISTINCT FROM 'object'
        OR (model_allowlist ? 'enabled' AND jsonb_typeof(model_allowlist->'enabled') IS DISTINCT FROM 'boolean')
        OR (model_allowlist ? 'models' AND jsonb_typeof(model_allowlist->'models') IS DISTINCT FROM 'array')
        OR jsonb_typeof(codex_models_manifest_config) IS DISTINCT FROM 'object'
        OR (codex_models_manifest_config ? 'enabled' AND jsonb_typeof(codex_models_manifest_config->'enabled') IS DISTINCT FROM 'boolean')
        OR (model_pricing IS NOT NULL AND jsonb_typeof(model_pricing) NOT IN ('array','null'))) THEN
        RAISE EXCEPTION 'rollback blocked: malformed or unknown native policy shape';
    END IF;
    IF EXISTS (SELECT 1 FROM groups WHERE
        COALESCE(model_allowlist->>'enabled','false') <> 'false'
        OR force_openai_fast OR free_openai_fast
        OR max_reasoning_effort_over_limit <> 'downgrade'
        OR COALESCE(codex_models_manifest_config->>'enabled','false') <> 'false')
       OR EXISTS (SELECT 1 FROM users WHERE restrict_public_groups) THEN
        RAISE EXCEPTION 'rollback blocked: native-only request policy would be lost';
    END IF;
    IF EXISTS (SELECT 1 FROM channel_model_pricing WHERE
        cache_write_1h_price IS NOT NULL OR max_reasoning_effort_multiplier IS NOT NULL)
       OR EXISTS (SELECT 1 FROM channel_pricing_intervals WHERE cache_write_1h_price IS NOT NULL)
       OR EXISTS (SELECT 1 FROM channel_account_stats_model_pricing WHERE cache_write_1h_price IS NOT NULL)
       OR EXISTS (SELECT 1 FROM channel_account_stats_pricing_intervals WHERE cache_write_1h_price IS NOT NULL) THEN
        RAISE EXCEPTION 'rollback blocked: native-only billing rules would be lost';
    END IF;
    IF EXISTS (SELECT 1 FROM groups WHERE
        jsonb_path_exists(COALESCE(model_pricing,'[]'::jsonb), '$.**.cache_write_1h_price ? (@ != null)')
        OR jsonb_path_exists(COALESCE(model_pricing,'[]'::jsonb), '$.**.max_reasoning_effort_multiplier ? (@ != null)')) THEN
        RAISE EXCEPTION 'rollback blocked: native-only group billing rules would be lost';
    END IF;
    IF EXISTS (SELECT 1 FROM accounts WHERE platform IN ('minimax','opencode_go'))
       OR EXISTS (SELECT 1 FROM groups WHERE platform IN ('minimax','opencode_go'))
       OR EXISTS (SELECT 1 FROM user_platform_quotas WHERE platform IN ('minimax','opencode_go'))
       OR EXISTS (SELECT 1 FROM composite_model_routes WHERE target_platform IN ('minimax','opencode_go'))
       OR EXISTS (SELECT 1 FROM channel_monitors WHERE provider IN ('minimax','opencode_go'))
       OR EXISTS (SELECT 1 FROM channel_monitor_request_templates WHERE provider IN ('minimax','opencode_go')) THEN
        RAISE EXCEPTION 'rollback blocked: native-only provider data exists';
    END IF;
    IF to_regclass('ssxz_v027_legacy_model_lists') IS NULL
       OR to_regclass('ssxz_v027_unlimited_quota_archive') IS NULL THEN
        RAISE EXCEPTION 'rollback blocked: preservation archives missing';
    END IF;
END
$$;
COMMIT;
