-- Persist upstream response-model observations without changing public model policy.
-- The nullable mismatch flag distinguishes "not observed" from a confirmed mismatch.

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_response_model VARCHAR(200),
    ADD COLUMN IF NOT EXISTS upstream_model_mismatch BOOLEAN;

CREATE INDEX IF NOT EXISTS idx_usage_logs_upstream_response_model
    ON usage_logs (upstream_response_model)
    WHERE upstream_response_model IS NOT NULL;
