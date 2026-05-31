ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS discount_applied BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS discount_percent DECIMAL(5, 2);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS discount_window_id VARCHAR(64);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS discount_window_type VARCHAR(16);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS discount_completed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_usage_logs_discount_applied_created_at
    ON usage_logs(discount_applied, created_at);
