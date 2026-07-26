ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS session_id VARCHAR(128);

CREATE INDEX IF NOT EXISTS idx_usage_logs_session_id
    ON usage_logs (session_id);
