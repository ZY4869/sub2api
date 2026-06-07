-- Record deleted API key identity metadata for later ops/usage attribution.
-- This keeps deleted users/keys searchable without exposing the original key.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

CREATE TABLE IF NOT EXISTS deleted_api_key_audits (
    api_key_id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    key_prefix VARCHAR(32) NOT NULL DEFAULT '',
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deleted_api_key_audits_user_deleted
    ON deleted_api_key_audits (user_id, deleted_at DESC);

CREATE INDEX IF NOT EXISTS idx_deleted_api_key_audits_key_prefix
    ON deleted_api_key_audits (key_prefix)
    WHERE key_prefix <> '';
