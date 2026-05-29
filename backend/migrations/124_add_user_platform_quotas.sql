CREATE TABLE IF NOT EXISTS user_platform_quotas (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(64) NOT NULL,
    daily_limit_usd DECIMAL(20, 10),
    weekly_limit_usd DECIMAL(20, 10),
    monthly_limit_usd DECIMAL(20, 10),
    daily_usage_usd DECIMAL(20, 10) NOT NULL DEFAULT 0,
    weekly_usage_usd DECIMAL(20, 10) NOT NULL DEFAULT 0,
    monthly_usage_usd DECIMAL(20, 10) NOT NULL DEFAULT 0,
    daily_window_start TIMESTAMPTZ,
    weekly_window_start TIMESTAMPTZ,
    monthly_window_start TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_platform_quotas_user_platform UNIQUE (user_id, platform),
    CONSTRAINT chk_user_platform_quotas_daily_limit CHECK (daily_limit_usd IS NULL OR daily_limit_usd >= 0),
    CONSTRAINT chk_user_platform_quotas_weekly_limit CHECK (weekly_limit_usd IS NULL OR weekly_limit_usd >= 0),
    CONSTRAINT chk_user_platform_quotas_monthly_limit CHECK (monthly_limit_usd IS NULL OR monthly_limit_usd >= 0),
    CONSTRAINT chk_user_platform_quotas_daily_usage CHECK (daily_usage_usd >= 0),
    CONSTRAINT chk_user_platform_quotas_weekly_usage CHECK (weekly_usage_usd >= 0),
    CONSTRAINT chk_user_platform_quotas_monthly_usage CHECK (monthly_usage_usd >= 0)
);

CREATE INDEX IF NOT EXISTS idx_user_platform_quotas_user_id
    ON user_platform_quotas (user_id);

CREATE INDEX IF NOT EXISTS idx_user_platform_quotas_platform
    ON user_platform_quotas (platform);
