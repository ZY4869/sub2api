-- Add a group-level public model visibility filter.
-- Empty array keeps existing behavior; configured values only narrow visible/callable public models.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS visible_model_patterns JSONB NOT NULL DEFAULT '[]'::jsonb;

