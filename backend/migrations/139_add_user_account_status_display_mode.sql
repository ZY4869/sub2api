-- Add account-management status display density preference.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS account_status_display_mode VARCHAR(32) NOT NULL DEFAULT 'detailed';

UPDATE users
SET account_status_display_mode = 'detailed'
WHERE LOWER(TRIM(COALESCE(account_status_display_mode, ''))) NOT IN ('simple', 'detailed');

ALTER TABLE users
  ALTER COLUMN account_status_display_mode SET NOT NULL,
  ALTER COLUMN account_status_display_mode SET DEFAULT 'detailed';

COMMENT ON COLUMN users.account_status_display_mode IS 'Account table status display mode: simple or detailed.';
