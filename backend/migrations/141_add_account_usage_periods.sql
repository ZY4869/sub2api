-- Persist account usage display period boundaries for fixed-cycle account statistics.
CREATE TABLE IF NOT EXISTS account_usage_periods (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  window_type VARCHAR(32) NOT NULL,
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ,
  reset_at TIMESTAMPTZ,
  source VARCHAR(32) NOT NULL DEFAULT 'derived',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_usage_periods_window_type_check CHECK (window_type IN ('weekly', 'monthly')),
  CONSTRAINT account_usage_periods_source_check CHECK (source IN ('calendar', 'fixed', 'upstream_reset', 'expiry', 'derived', 'fallback_30d'))
);

CREATE INDEX IF NOT EXISTS idx_account_usage_periods_account_window_start
  ON account_usage_periods (account_id, window_type, start_at DESC);

CREATE INDEX IF NOT EXISTS idx_account_usage_periods_account_window_open
  ON account_usage_periods (account_id, window_type)
  WHERE end_at IS NULL;
