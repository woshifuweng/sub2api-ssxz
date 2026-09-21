-- Explicit user request: make GPT-6 Astra available, retaining existing
-- group multipliers, account/model restrictions, and all existing prices.
-- Rates match the shipped pricing_service.go fallback (GPT-6 Astra).
-- The original operator run mistakenly also inserted GPT-5.1. That unrelated
-- row was removed by enable-verified-fable51.sql after user clarification.
BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
LOCK TABLE channel_model_pricing IN SHARE ROW EXCLUSIVE MODE;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM channels WHERE id=4 AND status='active' AND restrict_models AND billing_model_source='requested') THEN
    RAISE EXCEPTION 'Channel configuration changed; review required';
  END IF;
  IF EXISTS (SELECT 1 FROM channel_model_pricing WHERE channel_id=4 AND models ?| ARRAY['gpt-6','gpt-6-astra','gpt-5.1']) THEN
    RAISE EXCEPTION 'Requested pricing already exists; do not overwrite';
  END IF;
END $$;
CREATE TEMP TABLE existing_pricing AS SELECT * FROM channel_model_pricing;
INSERT INTO channel_model_pricing
  (channel_id,platform,models,billing_mode,input_price,output_price,cache_write_price,cache_read_price,fast_multiplier,flex_multiplier)
VALUES
  (4,'openai','["gpt-6","gpt-6-astra"]','token',0.000010,0.000050,0.0000125,0.000001,2,0.5);
DO $$ BEGIN
  IF EXISTS (SELECT * FROM existing_pricing EXCEPT SELECT * FROM channel_model_pricing) THEN
    RAISE EXCEPTION 'Existing pricing changed';
  END IF;
END $$;
COMMIT;
