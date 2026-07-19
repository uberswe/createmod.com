-- Account merges are not reversible; only the index is dropped.
DROP INDEX IF EXISTS idx_users_email_lower;
