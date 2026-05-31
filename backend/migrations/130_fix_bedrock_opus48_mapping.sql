-- Correct persisted Claude Opus 4.8 Bedrock mapping to the official inference ID.
--
-- Only rewrite the old generated/default value. User-customized mappings keep their value.

UPDATE accounts
SET credentials = jsonb_set(
    credentials,
    '{model_mapping,claude-opus-4-8}',
    '"us.anthropic.claude-opus-4-8"'::jsonb,
    true
)
WHERE platform = 'anthropic'
  AND type = 'bedrock'
  AND deleted_at IS NULL
  AND jsonb_typeof(credentials->'model_mapping') = 'object'
  AND credentials->'model_mapping'->>'claude-opus-4-8' = 'us.anthropic.claude-opus-4-8-v1';
