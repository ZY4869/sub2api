-- Normalize legacy Baidu platform aliases to the canonical baidu_document_ai value.
-- This migration is idempotent and safe to rerun.

UPDATE accounts
SET platform = 'baidu_document_ai',
    updated_at = NOW()
WHERE LOWER(BTRIM(COALESCE(platform, ''))) = 'baidu';

UPDATE groups
SET platform = 'baidu_document_ai',
    updated_at = NOW()
WHERE LOWER(BTRIM(COALESCE(platform, ''))) = 'baidu';

UPDATE error_passthrough_rules
SET platforms = COALESCE((
        SELECT jsonb_agg(item.value ORDER BY item.ord)
        FROM (
            SELECT MIN(normalized.ord) AS ord, normalized.value
            FROM (
                SELECT entry.ord,
                       CASE
                           WHEN LOWER(BTRIM(entry.value)) = 'baidu' THEN 'baidu_document_ai'
                           ELSE BTRIM(entry.value)
                       END AS value
                FROM jsonb_array_elements_text(COALESCE(platforms, '[]'::jsonb)) WITH ORDINALITY AS entry(value, ord)
            ) AS normalized
            GROUP BY normalized.value
        ) AS item
    ), '[]'::jsonb),
    updated_at = NOW()
WHERE jsonb_typeof(COALESCE(platforms, '[]'::jsonb)) = 'array'
  AND EXISTS (
        SELECT 1
        FROM jsonb_array_elements_text(COALESCE(platforms, '[]'::jsonb)) AS entry(value)
        WHERE LOWER(BTRIM(entry.value)) = 'baidu'
    );
