-- Add account today statistics cycle display preference.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS account_today_stats_cycle_mode VARCHAR(32) NOT NULL DEFAULT 'calendar';

UPDATE users
SET account_today_stats_cycle_mode = 'calendar'
WHERE LOWER(TRIM(COALESCE(account_today_stats_cycle_mode, ''))) NOT IN ('calendar', 'fixed');

ALTER TABLE users
  ALTER COLUMN account_today_stats_windows SET DEFAULT '["today","weekly","monthly","total"]'::jsonb;

UPDATE users
SET account_today_stats_windows = '["today","weekly","monthly","total"]'::jsonb
WHERE account_today_stats_windows = '["today","weekly","total"]'::jsonb;

ALTER TABLE users
  ALTER COLUMN account_today_stats_cycle_mode SET NOT NULL,
  ALTER COLUMN account_today_stats_cycle_mode SET DEFAULT 'calendar';

COMMENT ON COLUMN users.account_today_stats_cycle_mode IS 'Account today statistics cycle mode: calendar or fixed.';
COMMENT ON COLUMN users.account_today_stats_windows IS 'Account table today statistics windows: today, weekly, monthly, total.';
