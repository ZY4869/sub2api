ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_url VARCHAR(1024);

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_service VARCHAR(64);
