-- name: UpsertKnownIP :one
INSERT INTO user_known_ips (user_id, ip_address, user_agent, verified, last_seen)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id, ip_address) DO UPDATE SET
    user_agent = EXCLUDED.user_agent,
    last_seen = NOW()
RETURNING *;

-- name: GetKnownIP :one
SELECT * FROM user_known_ips
WHERE user_id = $1 AND ip_address = $2
LIMIT 1;

-- name: ListKnownIPs :many
SELECT * FROM user_known_ips
WHERE user_id = $1
ORDER BY last_seen DESC;

-- name: VerifyKnownIP :exec
UPDATE user_known_ips SET verified = true, last_seen = NOW()
WHERE user_id = $1 AND ip_address = $2;

-- name: DeleteKnownIP :exec
DELETE FROM user_known_ips WHERE id = $1 AND user_id = $2;

-- name: CreateIPVerificationCode :one
INSERT INTO ip_verification_codes (user_id, ip_address, code_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetIPVerificationCode :one
SELECT * FROM ip_verification_codes
WHERE user_id = $1 AND ip_address = $2 AND used = false AND expires_at > NOW()
ORDER BY created DESC LIMIT 1;

-- name: MarkIPVerificationCodeUsed :exec
UPDATE ip_verification_codes SET used = true WHERE id = $1;

-- name: CleanupExpiredIPCodes :exec
DELETE FROM ip_verification_codes WHERE expires_at < NOW();

-- name: UpsertTOTP :one
INSERT INTO user_totp (user_id, secret_encrypted, enabled, verified)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    secret_encrypted = EXCLUDED.secret_encrypted,
    enabled = EXCLUDED.enabled,
    verified = EXCLUDED.verified,
    updated = NOW()
RETURNING *;

-- name: GetTOTP :one
SELECT * FROM user_totp WHERE user_id = $1 LIMIT 1;

-- name: EnableTOTP :exec
UPDATE user_totp SET enabled = true, verified = true, updated = NOW()
WHERE user_id = $1;

-- name: DisableTOTP :exec
UPDATE user_totp SET enabled = false, updated = NOW()
WHERE user_id = $1;

-- name: DeleteTOTP :exec
DELETE FROM user_totp WHERE user_id = $1;

-- name: CreateTOTPBackupCode :exec
INSERT INTO user_totp_backup_codes (user_id, code_hash) VALUES ($1, $2);

-- name: ListTOTPBackupCodes :many
SELECT * FROM user_totp_backup_codes
WHERE user_id = $1 AND used = false
ORDER BY created ASC;

-- name: MarkBackupCodeUsed :exec
UPDATE user_totp_backup_codes SET used = true WHERE id = $1;

-- name: DeleteTOTPBackupCodes :exec
DELETE FROM user_totp_backup_codes WHERE user_id = $1;

-- name: CreatePasskey :one
INSERT INTO user_passkeys (user_id, credential_id, public_key, attestation_type, transport, aaguid, sign_count, friendly_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPasskeyByCredentialID :one
SELECT * FROM user_passkeys WHERE credential_id = $1 LIMIT 1;

-- name: ListPasskeys :many
SELECT * FROM user_passkeys WHERE user_id = $1 ORDER BY created DESC;

-- name: UpdatePasskeySignCount :exec
UPDATE user_passkeys SET sign_count = $2, last_used = NOW()
WHERE id = $1;

-- name: DeletePasskey :exec
DELETE FROM user_passkeys WHERE id = $1 AND user_id = $2;

-- name: UpsertSecuritySettings :one
INSERT INTO user_security_settings (user_id, new_ip_verification, totp_enabled, passkeys_enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    new_ip_verification = EXCLUDED.new_ip_verification,
    totp_enabled = EXCLUDED.totp_enabled,
    passkeys_enabled = EXCLUDED.passkeys_enabled,
    updated = NOW()
RETURNING *;

-- name: GetSecuritySettings :one
SELECT * FROM user_security_settings WHERE user_id = $1 LIMIT 1;
