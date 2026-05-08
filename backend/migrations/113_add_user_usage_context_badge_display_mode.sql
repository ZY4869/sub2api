ALTER TABLE users
  ADD COLUMN IF NOT EXISTS usage_context_badge_display_mode VARCHAR(32) NOT NULL DEFAULT 'request_only';
