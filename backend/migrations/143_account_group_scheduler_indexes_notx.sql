-- Improve account/group scheduling lookups without blocking writes.
-- notx: CREATE INDEX CONCURRENTLY must run outside a transaction.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_key_groups_group_api_key
    ON api_key_groups (group_id, api_key_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounts_status_platform_type
    ON accounts (status, platform, type);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_account_groups_group_account
    ON account_groups (group_id, account_id);
