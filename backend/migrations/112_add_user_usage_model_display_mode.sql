ALTER TABLE users
  ADD COLUMN IF NOT EXISTS usage_model_display_mode VARCHAR(32) NOT NULL DEFAULT 'model_only';
