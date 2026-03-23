ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS lifecycle_state VARCHAR(20) NOT NULL DEFAULT 'normal';

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS lifecycle_reason_code VARCHAR(100);

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS lifecycle_reason_message TEXT;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS blacklisted_at TIMESTAMPTZ;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS blacklist_purge_at TIMESTAMPTZ;

UPDATE accounts
SET lifecycle_state = 'normal'
WHERE lifecycle_state IS NULL OR btrim(lifecycle_state) = '';

CREATE INDEX IF NOT EXISTS idx_accounts_lifecycle_state
    ON accounts (lifecycle_state)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_blacklist_purge_at
    ON accounts (blacklist_purge_at)
    WHERE deleted_at IS NULL AND lifecycle_state = 'blacklisted';
