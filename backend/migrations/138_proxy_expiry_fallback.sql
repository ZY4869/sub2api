-- Add proxy expiry metadata and account original-proxy tracking for fallback recovery.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS expiry_remind_days INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS fallback_proxy_id BIGINT REFERENCES proxies(id) ON DELETE SET NULL;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS original_proxy_id BIGINT,
    ADD COLUMN IF NOT EXISTS original_proxy_name TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'proxies_expiry_remind_days_non_negative'
    ) THEN
        ALTER TABLE proxies
            ADD CONSTRAINT proxies_expiry_remind_days_non_negative
            CHECK (expiry_remind_days >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_proxies_expires_at
    ON proxies (expires_at)
    WHERE expires_at IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_proxies_fallback_proxy_id
    ON proxies (fallback_proxy_id)
    WHERE fallback_proxy_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_original_proxy_id
    ON accounts (original_proxy_id)
    WHERE original_proxy_id IS NOT NULL AND deleted_at IS NULL;
