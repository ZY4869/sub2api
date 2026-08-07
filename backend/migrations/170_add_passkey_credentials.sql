-- Add local passkey credential storage. Numbering intentionally follows the
-- local migration line and does not reuse upstream migration identifiers.

CREATE TABLE IF NOT EXISTS passkey_user_handles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    user_handle BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS passkey_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    credential_data JSONB NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_passkey_credentials_user_id
    ON passkey_credentials(user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_passkey_credentials_created_at
    ON passkey_credentials(created_at)
    WHERE deleted_at IS NULL;
