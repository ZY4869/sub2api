ALTER TABLE users
  ADD COLUMN IF NOT EXISTS global_realtime_countdown_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS account_realtime_countdown_enabled BOOLEAN NOT NULL DEFAULT TRUE;
