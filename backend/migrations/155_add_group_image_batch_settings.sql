-- Add group-level controls for public image batch generation.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS image_batch_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS image_batch_allowed_providers TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    ADD COLUMN IF NOT EXISTS image_batch_allowed_models TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    ADD COLUMN IF NOT EXISTS image_batch_max_items INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS image_batch_max_download_bytes BIGINT NOT NULL DEFAULT 104857600,
    ADD COLUMN IF NOT EXISTS image_batch_download_concurrency INTEGER NOT NULL DEFAULT 2;

UPDATE groups
SET image_batch_max_items = 50
WHERE image_batch_max_items <= 0;

UPDATE groups
SET image_batch_max_download_bytes = 104857600
WHERE image_batch_max_download_bytes <= 0;

UPDATE groups
SET image_batch_download_concurrency = 2
WHERE image_batch_download_concurrency <= 0;
