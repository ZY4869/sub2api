-- Mixed-currency billing wallets and usage currency metadata.
-- Idempotent: safe to run more than once.

CREATE TABLE IF NOT EXISTS billing_wallets (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency VARCHAR(3) NOT NULL,
    balance DECIMAL(20, 10) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, currency)
);

CREATE INDEX IF NOT EXISTS idx_billing_wallets_currency
    ON billing_wallets (currency);

INSERT INTO billing_wallets (user_id, currency, balance)
SELECT id, 'USD', balance
FROM users
WHERE deleted_at IS NULL
ON CONFLICT (user_id, currency) DO NOTHING;

CREATE TABLE IF NOT EXISTS billing_ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency VARCHAR(3) NOT NULL,
    amount DECIMAL(20, 10) NOT NULL,
    type VARCHAR(32) NOT NULL,
    request_id VARCHAR(255),
    fx_rate DECIMAL(20, 8),
    fx_rate_date VARCHAR(16),
    fx_locked_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_ledger_entries_user_created_at
    ON billing_ledger_entries (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_billing_ledger_entries_request_id
    ON billing_ledger_entries (request_id)
    WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_billing_ledger_entries_currency_type
    ON billing_ledger_entries (currency, type);

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS billing_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS total_cost_usd_equivalent DECIMAL(20, 10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS actual_cost_usd_equivalent DECIMAL(20, 10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS usd_to_cny_rate DECIMAL(20, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS fx_rate_date VARCHAR(16),
    ADD COLUMN IF NOT EXISTS fx_locked_at TIMESTAMPTZ;

UPDATE usage_logs
SET
    billing_currency = COALESCE(NULLIF(billing_currency, ''), 'USD'),
    total_cost_usd_equivalent = CASE WHEN total_cost_usd_equivalent = 0 THEN total_cost ELSE total_cost_usd_equivalent END,
    actual_cost_usd_equivalent = CASE WHEN actual_cost_usd_equivalent = 0 THEN actual_cost ELSE actual_cost_usd_equivalent END
WHERE billing_currency IS NULL
   OR billing_currency = ''
   OR total_cost_usd_equivalent = 0
   OR actual_cost_usd_equivalent = 0;

CREATE INDEX IF NOT EXISTS idx_usage_logs_billing_currency_created_at
    ON usage_logs (billing_currency, created_at DESC);

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS daily_usage_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS weekly_usage_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS monthly_usage_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE user_subscriptions
SET
    daily_usage_by_currency = daily_usage_by_currency || jsonb_build_object('USD', daily_usage_usd),
    weekly_usage_by_currency = weekly_usage_by_currency || jsonb_build_object('USD', weekly_usage_usd),
    monthly_usage_by_currency = monthly_usage_by_currency || jsonb_build_object('USD', monthly_usage_usd)
WHERE NOT (daily_usage_by_currency ? 'USD')
   OR NOT (weekly_usage_by_currency ? 'USD')
   OR NOT (monthly_usage_by_currency ? 'USD');

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS quota_used_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS usage_5h_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS usage_1d_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS usage_7d_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE api_keys
SET
    quota_used_by_currency = quota_used_by_currency || jsonb_build_object('USD', quota_used),
    usage_5h_by_currency = usage_5h_by_currency || jsonb_build_object('USD', usage_5h),
    usage_1d_by_currency = usage_1d_by_currency || jsonb_build_object('USD', usage_1d),
    usage_7d_by_currency = usage_7d_by_currency || jsonb_build_object('USD', usage_7d)
WHERE NOT (quota_used_by_currency ? 'USD')
   OR NOT (usage_5h_by_currency ? 'USD')
   OR NOT (usage_1d_by_currency ? 'USD')
   OR NOT (usage_7d_by_currency ? 'USD');

ALTER TABLE api_key_groups
    ADD COLUMN IF NOT EXISTS quota_used_by_currency JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE api_key_groups
SET quota_used_by_currency = quota_used_by_currency || jsonb_build_object('USD', quota_used)
WHERE NOT (quota_used_by_currency ? 'USD');
