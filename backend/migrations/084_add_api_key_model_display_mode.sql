ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS model_display_mode VARCHAR(32) NOT NULL DEFAULT 'alias_only';
