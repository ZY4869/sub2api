CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    country_code VARCHAR(8) NOT NULL DEFAULT '',
    payment_env VARCHAR(32) NOT NULL DEFAULT 'production',
    amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    credited_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    credited_currency VARCHAR(16) NOT NULL DEFAULT 'USD',
    merchant_order_id VARCHAR(128) NOT NULL,
    provider_intent_id VARCHAR(128) NOT NULL DEFAULT '',
    provider_payment_link_id VARCHAR(128) NOT NULL DEFAULT '',
    provider_refund_id VARCHAR(128) NOT NULL DEFAULT '',
    resume_token VARCHAR(128) NOT NULL,
    checkout_url TEXT NOT NULL DEFAULT '',
    return_url TEXT NOT NULL DEFAULT '',
    cancel_url TEXT NOT NULL DEFAULT '',
    provider_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    order_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    completed_at TIMESTAMPTZ NULL,
    refunded_at TIMESTAMPTZ NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_resume_token
    ON payment_orders (resume_token);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_merchant_order_id
    ON payment_orders (merchant_order_id);

CREATE INDEX IF NOT EXISTS idx_payment_orders_user_created_at
    ON payment_orders (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_payment_orders_provider_intent
    ON payment_orders (provider_intent_id);

CREATE INDEX IF NOT EXISTS idx_payment_orders_provider_payment_link
    ON payment_orders (provider_payment_link_id);
