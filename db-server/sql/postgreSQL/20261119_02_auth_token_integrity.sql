-- Stop older API instances before applying, then deploy the matching API.
-- Preserves existing bearer cookies by hashing their stored counterpart.
-- Requires PostgreSQL 13 or later; no extension is needed.
BEGIN;
ALTER TABLE st_token ALTER COLUMN token TYPE text;
UPDATE st_token SET token='sha256:' || encode(sha256(convert_to(token,'UTF8')),'hex')
WHERE token IS NOT NULL AND token NOT LIKE 'sha256:%';
CREATE UNIQUE INDEX IF NOT EXISTS uq_st_token_name_digest ON st_token(name,token);
CREATE TABLE IF NOT EXISTS st_auth_rate_limits (
    key_hash text NOT NULL,
    window_start timestamptz NOT NULL,
    attempts integer NOT NULL,
    PRIMARY KEY(key_hash,window_start)
);
CREATE INDEX IF NOT EXISTS idx_auth_rate_limit_expiry ON st_auth_rate_limits(window_start);
COMMIT;
