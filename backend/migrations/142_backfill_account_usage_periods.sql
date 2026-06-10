-- Backfill one locally inferable monthly usage period for legacy accounts.
INSERT INTO account_usage_periods (account_id, window_type, start_at, end_at, reset_at, source)
SELECT
  a.id,
  'monthly',
  COALESCE(a.created_at, NOW()),
  CASE
    WHEN a.expires_at IS NOT NULL AND a.expires_at > COALESCE(a.created_at, NOW()) THEN a.expires_at
    ELSE NULL
  END,
  CASE
    WHEN a.expires_at IS NOT NULL AND a.expires_at > COALESCE(a.created_at, NOW()) THEN a.expires_at
    ELSE NULL
  END,
  CASE
    WHEN a.expires_at IS NOT NULL AND a.expires_at > COALESCE(a.created_at, NOW()) THEN 'expiry'
    ELSE 'fallback_30d'
  END
FROM accounts a
WHERE NOT EXISTS (
  SELECT 1
  FROM account_usage_periods p
  WHERE p.account_id = a.id
    AND p.window_type = 'monthly'
);
