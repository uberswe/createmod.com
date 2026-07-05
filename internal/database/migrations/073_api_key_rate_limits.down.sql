DROP INDEX IF EXISTS idx_api_key_usage_key_created;
ALTER TABLE api_keys DROP COLUMN IF EXISTS rate_limit_per_minute;
