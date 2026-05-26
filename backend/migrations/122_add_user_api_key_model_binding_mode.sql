ALTER TABLE users
  ADD COLUMN IF NOT EXISTS api_key_model_binding_mode VARCHAR(32) NOT NULL DEFAULT 'model_required';

UPDATE users
SET api_key_model_binding_mode = 'model_required'
WHERE TRIM(COALESCE(api_key_model_binding_mode, '')) = '';
