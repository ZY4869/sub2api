-- Disable user-managed image-count billing for non-admin API keys.
-- Admin-owned keys keep their existing image-count billing configuration.

UPDATE api_keys AS ak
SET
    image_count_billing_enabled = FALSE,
    image_max_count = 0,
    image_count_used = 0,
    image_count_weights = '{"1K":1,"2K":1,"4K":2}'::jsonb
FROM users AS u
WHERE ak.user_id = u.id
  AND u.role <> 'admin'
  AND ak.deleted_at IS NULL
  AND (
      ak.image_count_billing_enabled = TRUE
      OR ak.image_max_count <> 0
      OR ak.image_count_used <> 0
      OR ak.image_count_weights <> '{"1K":1,"2K":1,"4K":2}'::jsonb
  );
