ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS auto_renew_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS auto_renew_period VARCHAR(20) NOT NULL DEFAULT 'month';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_accounts_auto_renew_period'
    ) THEN
        ALTER TABLE accounts
        ADD CONSTRAINT chk_accounts_auto_renew_period
        CHECK (auto_renew_period IN ('month', 'quarter', 'year'));
    END IF;
END $$;

COMMENT ON COLUMN accounts.auto_renew_enabled IS 'Automatically extend expires_at when the account reaches its expiration time.';
COMMENT ON COLUMN accounts.auto_renew_period IS 'Automatic renewal period: month, quarter, or year.';
