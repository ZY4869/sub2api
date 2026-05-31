-- Add claude-opus-4-8 to persisted Antigravity model mappings.
--
-- Accounts without a persisted model_mapping use the built-in default mapping
-- directly, so this only backfills existing mapping objects.

UPDATE accounts
SET credentials = jsonb_set(
    credentials,
    '{model_mapping,claude-opus-4-8}',
    '"claude-opus-4-8"'::jsonb,
    true
)
WHERE platform = 'antigravity'
  AND deleted_at IS NULL
  AND jsonb_typeof(credentials->'model_mapping') = 'object'
  AND credentials->'model_mapping'->>'claude-opus-4-8' IS NULL;
