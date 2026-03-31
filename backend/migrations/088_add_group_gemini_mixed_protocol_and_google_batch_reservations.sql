ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS gemini_mixed_protocol_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS google_batch_quota_reservations (
    id BIGSERIAL PRIMARY KEY,
    provider_family VARCHAR(32) NOT NULL,
    account_id BIGINT NOT NULL,
    resource_name TEXT NOT NULL,
    model_family VARCHAR(128) NOT NULL DEFAULT '',
    reserved_tokens BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_google_batch_quota_reservations_resource_name
    ON google_batch_quota_reservations (resource_name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_quota_reservations_account_id
    ON google_batch_quota_reservations (account_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_quota_reservations_provider_status
    ON google_batch_quota_reservations (provider_family, status)
    WHERE deleted_at IS NULL;
