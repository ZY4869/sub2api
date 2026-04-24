-- Add image-only API key fields and optional image-count quota tracking.
-- Defaults keep existing keys unaffected.

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS image_only_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS image_count_billing_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS image_max_count INT NOT NULL DEFAULT 0;

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS image_count_used INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN api_keys.image_only_enabled IS 'When enabled, this API key can only access image generation models/endpoints.';
COMMENT ON COLUMN api_keys.image_count_billing_enabled IS 'When enabled (and image_max_count>0), image requests are limited by image_max_count and billed by image count.';
COMMENT ON COLUMN api_keys.image_max_count IS 'Max allowed image outputs for this API key when image_count_billing_enabled is true. 0 means unlimited / token-billing.';
COMMENT ON COLUMN api_keys.image_count_used IS 'Used image outputs for image count billing (only successful images are counted).';

