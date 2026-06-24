-- Add cross-device usage records display preferences.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS usage_view_preferences JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE users
SET usage_view_preferences = '{}'::jsonb
WHERE
  usage_view_preferences IS NULL
  OR jsonb_typeof(usage_view_preferences) <> 'object';

ALTER TABLE users
  ALTER COLUMN usage_view_preferences SET NOT NULL,
  ALTER COLUMN usage_view_preferences SET DEFAULT '{}'::jsonb;

COMMENT ON COLUMN users.usage_view_preferences IS 'Usage records page display preferences keyed by admin/user page.';
