CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_api_key_created_ip_not_null
    ON usage_logs (api_key_id, created_at DESC, ip_address)
    WHERE api_key_id IS NOT NULL AND ip_address IS NOT NULL;
