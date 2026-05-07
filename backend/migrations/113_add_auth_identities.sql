CREATE TABLE IF NOT EXISTS auth_identities (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL DEFAULT '',
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_identities_provider_user
    ON auth_identities(provider, provider_user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_identities_user_provider
    ON auth_identities(user_id, provider);

CREATE INDEX IF NOT EXISTS idx_auth_identities_user_id
    ON auth_identities(user_id);

CREATE INDEX IF NOT EXISTS idx_auth_identities_email
    ON auth_identities(email);
