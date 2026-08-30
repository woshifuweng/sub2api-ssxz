-- Older production databases created signup_source at VARCHAR(20) through
-- migration 108, while fresh databases already use VARCHAR(32) from migration
-- 085. Widening is metadata-only for existing values and makes both upgrade
-- paths converge without truncating or rewriting authentication data.

ALTER TABLE users
    ALTER COLUMN signup_source TYPE VARCHAR(32);
