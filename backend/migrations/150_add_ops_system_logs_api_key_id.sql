-- Add API key correlation to indexed system logs.

ALTER TABLE ops_system_logs
  ADD COLUMN IF NOT EXISTS api_key_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_ops_system_logs_api_key_id_created_at
  ON ops_system_logs (api_key_id, created_at DESC);
