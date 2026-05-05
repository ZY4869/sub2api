ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS requested_model_raw TEXT;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS requested_model_normalized TEXT;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS million_context_requested BOOLEAN;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS million_context_effective BOOLEAN;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS million_context_source VARCHAR(64);

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS million_context_beta_token VARCHAR(128);
