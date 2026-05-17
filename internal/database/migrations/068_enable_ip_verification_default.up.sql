-- Enable IP verification for existing email/password users (not OAuth-only) who don't have security settings yet
INSERT INTO user_security_settings (id, user_id, new_ip_verification, totp_enabled, passkeys_enabled, created, updated)
SELECT
    substr(md5(random()::text), 1, 15),
    u.id,
    true,
    false,
    false,
    now(),
    now()
FROM users u
WHERE u.deleted IS NULL
  AND u.password_hash != ''
  AND NOT EXISTS (
    SELECT 1 FROM user_security_settings s WHERE s.user_id = u.id
  );

-- For existing email/password users who have settings but IP verification disabled, enable it
UPDATE user_security_settings
SET new_ip_verification = true, updated = now()
WHERE new_ip_verification = false
  AND user_id IN (SELECT id FROM users WHERE password_hash != '' AND deleted IS NULL);
