-- Add explicit thinking mode flag to usage_logs.
-- NULL keeps historical rows compatible when no reliable thinking signal exists.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS thinking_enabled BOOLEAN;
