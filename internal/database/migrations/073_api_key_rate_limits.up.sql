-- Admin-managed per-key rate limit override. 0 means "use the endpoint default".
ALTER TABLE api_keys ADD COLUMN rate_limit_per_minute INTEGER NOT NULL DEFAULT 0;

-- Supports per-key usage aggregates (counts within time windows, last-used)
-- on the admin API keys page.
CREATE INDEX idx_api_key_usage_key_created ON api_key_usage (api_key_id, created);
