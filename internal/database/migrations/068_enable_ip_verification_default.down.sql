-- Revert: disable IP verification for all users
UPDATE user_security_settings
SET new_ip_verification = false, updated = now();
