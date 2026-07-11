-- Emails are compared case-insensitively from now on (application code
-- lowercases on write, lookups match on LOWER(email)).
-- Lowercase existing emails except case-insensitive duplicate groups:
-- those are distinct accounts sharing an address in different casings and
-- need manual merging before they can be normalized.
UPDATE users SET email = LOWER(email)
WHERE email <> LOWER(email)
  AND NOT EXISTS (
    SELECT 1 FROM users u2
    WHERE u2.id <> users.id AND LOWER(u2.email) = LOWER(users.email)
  );

-- Support case-insensitive lookups. Not UNIQUE while the pre-existing
-- duplicate groups remain; new duplicates are rejected in the application.
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));
