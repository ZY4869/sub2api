-- Add optional group profit controls. Defaults preserve existing billing behavior.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS profit_control_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS profit_min_margin DECIMAL(10,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS profit_safety_buffer DECIMAL(10,4) NOT NULL DEFAULT 0;
