-- Persist Gemini-specific correlation fields on ops_error_logs so runtime errors
-- can be filtered by surface, matched billing rule, and recovery probe action.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

ALTER TABLE ops_error_logs
    ADD COLUMN IF NOT EXISTS gemini_surface VARCHAR(64),
    ADD COLUMN IF NOT EXISTS billing_rule_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS probe_action VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_gemini_surface_time
    ON ops_error_logs (gemini_surface, created_at DESC)
    WHERE gemini_surface IS NOT NULL
      AND gemini_surface <> '';

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_billing_rule_id
    ON ops_error_logs (billing_rule_id)
    WHERE billing_rule_id IS NOT NULL
      AND billing_rule_id <> '';

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_probe_action_time
    ON ops_error_logs (probe_action, created_at DESC)
    WHERE probe_action IS NOT NULL
      AND probe_action <> '';
