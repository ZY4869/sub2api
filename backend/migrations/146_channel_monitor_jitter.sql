ALTER TABLE channel_monitors
  ADD COLUMN IF NOT EXISTS jitter_seconds INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'channel_monitors_jitter_seconds_check'
  ) THEN
    ALTER TABLE channel_monitors
      ADD CONSTRAINT channel_monitors_jitter_seconds_check
      CHECK (
        jitter_seconds >= 0
        AND jitter_seconds <= 3585
        AND interval_seconds - jitter_seconds >= 15
      );
  END IF;
END $$;
