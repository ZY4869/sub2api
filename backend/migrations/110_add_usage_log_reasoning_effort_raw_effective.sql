ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS reasoning_effort_raw VARCHAR(20);

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS reasoning_effort_effective VARCHAR(20);

UPDATE usage_logs
SET reasoning_effort_effective = COALESCE(reasoning_effort_effective, reasoning_effort)
WHERE reasoning_effort IS NOT NULL
  AND COALESCE(reasoning_effort_effective, '') = '';

