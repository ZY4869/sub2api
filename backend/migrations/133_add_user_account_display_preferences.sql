-- Add account-management display preferences.
-- Defaults preserve the existing table presentation.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS account_today_stats_windows JSONB NOT NULL DEFAULT '["today","weekly","total"]'::jsonb,
  ADD COLUMN IF NOT EXISTS account_group_display_mode VARCHAR(32) NOT NULL DEFAULT 'full';

ALTER TABLE users
  ALTER COLUMN account_today_stats_windows SET DEFAULT '["today","weekly","total"]'::jsonb,
  ALTER COLUMN account_group_display_mode SET DEFAULT 'full';

UPDATE users
SET
  account_today_stats_windows = '["today","weekly","total"]'::jsonb
WHERE
  account_today_stats_windows IS NULL
  OR jsonb_typeof(account_today_stats_windows) <> 'array'
  OR jsonb_array_length(account_today_stats_windows) = 0;

UPDATE users
SET
  account_group_display_mode = 'full'
WHERE
  LOWER(TRIM(COALESCE(account_group_display_mode, ''))) NOT IN ('full', 'icon');

ALTER TABLE users
  ALTER COLUMN account_today_stats_windows SET NOT NULL,
  ALTER COLUMN account_group_display_mode SET NOT NULL;

COMMENT ON COLUMN users.account_today_stats_windows IS 'Account table today statistics windows: today, weekly, total.';
COMMENT ON COLUMN users.account_group_display_mode IS 'Account table group display mode: full or icon.';
