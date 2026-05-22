ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS request_context_length_tokens INTEGER;
