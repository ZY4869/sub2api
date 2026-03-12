ALTER TABLE users
  ADD COLUMN IF NOT EXISTS admin_free_billing boolean NOT NULL DEFAULT false;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS billing_exempt_reason text NULL;

CREATE INDEX IF NOT EXISTS idx_usage_logs_billing_exempt_reason_created_at
  ON usage_logs (billing_exempt_reason, created_at);
