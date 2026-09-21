-- SSXZ upgrade bridge, before native 235/236. Legacy /v1/models presentation
-- preferences were NOT request authorization. Do not promote them into denials.
-- Keep the old column for a guarded old-binary rollback; native policies remain
-- separate. Never change API-key, account, user/group grants or routing here.
CREATE TABLE IF NOT EXISTS ssxz_v027_legacy_model_lists (
    group_id BIGINT PRIMARY KEY,
    models_list_config JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
REVOKE ALL ON ssxz_v027_legacy_model_lists FROM PUBLIC;

DO $$
DECLARE
    has_legacy BOOLEAN;
    has_native BOOLEAN;
BEGIN
    LOCK TABLE groups IN ACCESS EXCLUSIVE MODE;
    SELECT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='groups'::regclass
        AND attname='models_list_config' AND NOT attisdropped) INTO has_legacy;
    SELECT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='groups'::regclass
        AND attname='model_allowlist' AND NOT attisdropped) INTO has_native;

    IF has_native THEN
        -- A native installation already owns its admission policy. If both
        -- columns predate this bridge, their provenance cannot be inferred.
        IF has_legacy THEN
            RAISE EXCEPTION 'SSXZ v027: ambiguous dual model columns; review provenance before upgrade';
        END IF;
        RETURN;
    END IF;
    IF NOT has_legacy THEN
        RAISE EXCEPTION 'SSXZ v027: missing legacy and native model columns';
    END IF;
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE filename IN
        ('235_group_model_allowlist.sql','236_group_model_allowlist_repair.sql')) THEN
        RAISE EXCEPTION 'SSXZ v027: native migration previously applied but native column missing';
    END IF;

    INSERT INTO ssxz_v027_legacy_model_lists (group_id, models_list_config)
        SELECT id, models_list_config FROM groups;
    -- Deliberately nonempty: native 236 backfills only {} and must not copy
    -- display-only preferences into the new admission column.
    ALTER TABLE groups ADD COLUMN model_allowlist JSONB NOT NULL
        DEFAULT '{"enabled":false,"models":[]}'::jsonb;
END
$$;

COMMENT ON TABLE ssxz_v027_legacy_model_lists IS
    'Private upgrade archive of display-only preferences; not live authorization';
