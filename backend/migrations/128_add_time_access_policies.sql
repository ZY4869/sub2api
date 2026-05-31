-- Add scheduled activation and time-window policy controls.

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS access_time_policy JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_api_keys_starts_at ON api_keys (starts_at);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS api_key_access_time_policy JSONB NOT NULL DEFAULT '{}'::jsonb;
