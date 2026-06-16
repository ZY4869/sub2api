ALTER TABLE users
  ADD COLUMN IF NOT EXISTS external_model_catalog_view_mode VARCHAR(32) NOT NULL DEFAULT 'follow_key_binding';

UPDATE users
SET external_model_catalog_view_mode = 'follow_key_binding'
WHERE TRIM(COALESCE(external_model_catalog_view_mode, '')) = ''
   OR external_model_catalog_view_mode NOT IN ('follow_key_binding', 'group_first', 'model_only');
