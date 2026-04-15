ALTER TABLE ops_request_traces
    ADD COLUMN IF NOT EXISTS gemini_surface VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS billing_rule_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS probe_action VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_gemini_surface_time
    ON ops_request_traces (gemini_surface, created_at DESC)
    WHERE gemini_surface <> '';

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_billing_rule_id
    ON ops_request_traces (billing_rule_id)
    WHERE billing_rule_id <> '';

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_probe_action_time
    ON ops_request_traces (probe_action, created_at DESC)
    WHERE probe_action <> '';
