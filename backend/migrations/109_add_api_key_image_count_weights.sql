-- Add configurable per-resolution image count weights for image-only API keys.
-- Defaults preserve the previous behavior while allowing 4K images to count more units.

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS image_count_weights JSONB NOT NULL DEFAULT '{"1K":1,"2K":1,"4K":2}'::jsonb;

COMMENT ON COLUMN api_keys.image_count_weights IS 'Per-resolution image count billing weights for image-only API keys, e.g. {"2K":1,"4K":2}.';
