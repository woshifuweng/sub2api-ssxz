-- Enable the verified GPT Image 2.5 aliases without changing customer prices.
--
-- Read-only provider catalog probes on 2026-09-22 confirmed these aliases on
-- the accounts updated below.  This script intentionally does not enable the
-- generic gpt-image-2.5 alias: the site has no explicit price contract for it.
-- Existing group image prices remain authoritative (1K/2K = $0.15, 4K =
-- $0.30); the channel row only makes the model routable and image-billed.
--
-- Safe to re-run. It changes only model mappings, the two dedicated image
-- group listings, and channel 4's image model catalog.

BEGIN;

-- Active/schedulable image-capable accounts and the verified general pools.
UPDATE accounts
SET credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{model_mapping}',
        COALESCE(credentials->'model_mapping', '{}'::jsonb) ||
        '{"gpt-image-2.5-flare":"gpt-image-2.5-flare","gpt-image-2.5-sunburst":"gpt-image-2.5-sunburst"}'::jsonb,
        true
    ),
    updated_at = NOW()
WHERE id IN (45, 72, 98, 100, 106, 111, 112)
  AND deleted_at IS NULL
  AND status = 'active';

-- Keep the dedicated image groups' customer-visible catalog in sync with the
-- verified account mappings. Their existing image resolution prices remain.
UPDATE groups
SET models_list_config = jsonb_set(
        COALESCE(models_list_config, '{}'::jsonb),
        '{models}',
        (
            SELECT jsonb_agg(model ORDER BY model)
            FROM (
                SELECT DISTINCT model
                FROM jsonb_array_elements_text(
                    COALESCE(models_list_config->'models', '[]'::jsonb) ||
                    '["gpt-image-2.5-flare","gpt-image-2.5-sunburst"]'::jsonb
                ) AS model
            ) models
        ),
        true
    ),
    updated_at = NOW()
WHERE id IN (18, 19)
  AND deleted_at IS NULL
  AND status = 'active';

-- Channel 4 is the shared OpenAI route for the normal customer pools. The
-- group-level image resolution price overrides the display/billing amount
-- where configured; this row prevents the model from being rejected as
-- unpriced by restrict_models.
INSERT INTO channel_model_pricing (
    channel_id, platform, models, billing_mode,
    image_output_price, per_request_price, created_at, updated_at
)
SELECT 4, 'openai',
       '["gpt-image-2.5-flare","gpt-image-2.5-sunburst"]'::jsonb,
       'image', 0.15, 0.15, NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1
    FROM channel_model_pricing
    WHERE channel_id = 4
      AND platform = 'openai'
      AND billing_mode = 'image'
      AND models @> '["gpt-image-2.5-flare"]'::jsonb
      AND models @> '["gpt-image-2.5-sunburst"]'::jsonb
);

COMMIT;

-- Post-condition checks (read-only output for the operator).
SELECT 'accounts=' || count(*)
FROM accounts
WHERE id IN (45, 72, 98, 100, 106, 111, 112)
  AND credentials->'model_mapping' ?& array['gpt-image-2.5-flare','gpt-image-2.5-sunburst'];
SELECT 'groups=' || count(*)
FROM groups
WHERE id IN (18, 19)
  AND models_list_config->'models' ?& array['gpt-image-2.5-flare','gpt-image-2.5-sunburst'];
SELECT 'channel_pricing=' || count(*)
FROM channel_model_pricing
WHERE channel_id = 4
  AND platform = 'openai'
  AND billing_mode = 'image'
  AND models @> '["gpt-image-2.5-flare"]'::jsonb
  AND models @> '["gpt-image-2.5-sunburst"]'::jsonb;
