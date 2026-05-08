ALTER TABLE external_auths DROP COLUMN IF EXISTS metadata;
ALTER TABLE external_auths DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE external_auths DROP COLUMN IF EXISTS username;
ALTER TABLE external_auths DROP COLUMN IF EXISTS token_expiry;
ALTER TABLE external_auths DROP COLUMN IF EXISTS refresh_token_encrypted;
ALTER TABLE external_auths DROP COLUMN IF EXISTS access_token_encrypted;
