ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS allow_live BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS max_reasoning_effort VARCHAR(20) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS reasoning_effort_mappings JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE usage_logs
  DROP CONSTRAINT IF EXISTS usage_logs_request_type_check;

ALTER TABLE usage_logs
  ADD CONSTRAINT usage_logs_request_type_check
  CHECK (request_type IN (0, 1, 2, 3, 4, 5, 6));
