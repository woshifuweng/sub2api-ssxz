-- 2026-09-21: provider catalogs for accounts 89/99 advertise Fable 5.1.
-- Preserve all customer grants, prices, multipliers and existing account mappings.
BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
LOCK TABLE channel_model_pricing IN SHARE ROW EXCLUSIVE MODE;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM channel_model_pricing WHERE id=312 AND channel_id=4
      AND models='["gpt-5.1"]'::jsonb AND platform='openai'
      AND input_price=0.00000125 AND output_price=0.00001
      AND cache_read_price=0.000000125 AND cache_write_price IS NULL) THEN
    RAISE EXCEPTION 'Mistaken new GPT-5.1 row changed; manual review required';
  END IF;
  IF EXISTS (SELECT 1 FROM channel_model_pricing WHERE channel_id=4
      AND models ?| ARRAY['claude-fable-5-1','claude-fable-5.1']) THEN
    RAISE EXCEPTION 'Fable 5.1 already configured; do not overwrite';
  END IF;
  IF (SELECT count(*) FROM accounts a WHERE a.id IN (89,99)
      AND a.status='active' AND a.schedulable AND a.deleted_at IS NULL
      AND a.platform='anthropic' AND a.type='apikey'
      AND jsonb_typeof(a.credentials->'model_mapping')='object'
      AND NOT (a.credentials->'model_mapping' ? 'claude-fable-5-1')
      AND EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id AND ag.group_id=11)) <> 2 THEN
    RAISE EXCEPTION 'Verified account configuration changed';
  END IF;
END $$;
DELETE FROM channel_model_pricing WHERE id=312;
INSERT INTO channel_model_pricing
  (channel_id,platform,models,billing_mode,input_price,output_price,
   cache_write_price,cache_write_1h_price,cache_read_price,max_reasoning_effort_multiplier)
VALUES (4,'anthropic','["claude-fable-5-1"]','token',0.000010,0.000050,
        0.0000125,0.000020,0.00000025,3);
UPDATE accounts SET credentials=jsonb_set(credentials,'{model_mapping}',
    (credentials->'model_mapping') || '{"claude-fable-5-1":"claude-fable-5-1"}'::jsonb),
    updated_at=now() WHERE id IN (89,99);
INSERT INTO scheduler_outbox(event_type,account_id,payload,created_at)
SELECT 'account_changed', a.id, jsonb_build_object('group_ids',
    (SELECT jsonb_agg(ag.group_id) FROM account_groups ag WHERE ag.account_id=a.id)),now()
FROM accounts a WHERE a.id IN (89,99);
COMMIT;
