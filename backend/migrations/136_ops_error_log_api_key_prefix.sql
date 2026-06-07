-- Persist a short, non-secret API key prefix for deleted-key attribution.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

ALTER TABLE ops_error_logs
    ADD COLUMN IF NOT EXISTS api_key_prefix VARCHAR(32);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_api_key_prefix_time
    ON ops_error_logs (api_key_prefix, created_at DESC)
    WHERE api_key_prefix IS NOT NULL AND api_key_prefix <> '';
