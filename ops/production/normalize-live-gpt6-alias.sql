-- Provider GET /v1/models reports canonical gpt-6-astra on all eight active
-- OpenAI accounts. Normalize the bare family alias; do not substitute families.
BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
SELECT id FROM channels WHERE id=4 FOR UPDATE;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM channels WHERE id=4 AND model_mapping->'openai'->>'gpt-6' IS NOT NULL AND model_mapping->'openai'->>'gpt-6' <> 'gpt-6-astra') THEN
    RAISE EXCEPTION 'Existing alias differs; review required';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM channels WHERE id=4 AND jsonb_typeof(model_mapping->'openai')='object') THEN
    RAISE EXCEPTION 'Unexpected channel mapping structure';
  END IF;
END $$;
UPDATE channels SET model_mapping=jsonb_set(model_mapping,'{openai,gpt-6}','"gpt-6-astra"'::jsonb),updated_at=NOW() WHERE id=4;
COMMIT;
