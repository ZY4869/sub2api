-- Support user-scoped failed request and deleted-user usage lookups.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_user_time
    ON ops_error_logs (user_id, created_at DESC)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_api_key_time
    ON ops_error_logs (api_key_id, created_at DESC)
    WHERE api_key_id IS NOT NULL;
