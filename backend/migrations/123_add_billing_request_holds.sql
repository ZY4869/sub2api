CREATE TABLE IF NOT EXISTS billing_request_holds (
    id            BIGSERIAL PRIMARY KEY,
    request_id    TEXT NOT NULL,
    api_key_id    BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency      VARCHAR(3) NOT NULL DEFAULT 'USD',
    hold_amount   DECIMAL(20, 8) NOT NULL,
    actual_amount DECIMAL(20, 8),
    status        VARCHAR(20) NOT NULL DEFAULT 'held',
    request_fingerprint TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at    TIMESTAMPTZ,
    CONSTRAINT uq_billing_request_holds_request_key UNIQUE (request_id, api_key_id),
    CONSTRAINT chk_billing_request_holds_amount CHECK (hold_amount >= 0),
    CONSTRAINT chk_billing_request_holds_actual CHECK (actual_amount IS NULL OR actual_amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_billing_request_holds_user_status
    ON billing_request_holds (user_id, status, created_at);

ALTER TABLE billing_request_holds
    ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;
