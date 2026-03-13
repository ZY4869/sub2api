ALTER TABLE scheduled_test_plans
ADD COLUMN IF NOT EXISTS notify_policy VARCHAR(20) NOT NULL DEFAULT 'none',
ADD COLUMN IF NOT EXISTS notify_failure_threshold INT NOT NULL DEFAULT 3,
ADD COLUMN IF NOT EXISTS retry_interval_minutes INT NOT NULL DEFAULT 5,
ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 3,
ADD COLUMN IF NOT EXISTS consecutive_failures INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS current_retry_count INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN scheduled_test_plans.notify_policy IS 'Notification policy: none, always, failure_only';
COMMENT ON COLUMN scheduled_test_plans.notify_failure_threshold IS 'Send notification after N consecutive failures';
COMMENT ON COLUMN scheduled_test_plans.retry_interval_minutes IS 'Retry delay in minutes after a failed scheduled test';
COMMENT ON COLUMN scheduled_test_plans.max_retries IS 'Maximum attempts in a single scheduled test cycle';
COMMENT ON COLUMN scheduled_test_plans.consecutive_failures IS 'Current consecutive failure count across cycles';
COMMENT ON COLUMN scheduled_test_plans.current_retry_count IS 'Retry attempts already used in the current schedule cycle';
