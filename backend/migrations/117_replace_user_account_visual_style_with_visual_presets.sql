ALTER TABLE users
  ADD COLUMN IF NOT EXISTS visual_preset_preference VARCHAR(32) NOT NULL DEFAULT 'inherit',
  ADD COLUMN IF NOT EXISTS account_visual_preset_override VARCHAR(32) NOT NULL DEFAULT 'inherit';

UPDATE users
SET
  visual_preset_preference = CASE
    WHEN LOWER(TRIM(COALESCE(visual_preset_preference, ''))) IN ('inherit', 'classic', 'airy') THEN LOWER(TRIM(visual_preset_preference))
    ELSE 'inherit'
  END,
  account_visual_preset_override = CASE
    WHEN LOWER(TRIM(COALESCE(account_visual_preset_override, ''))) IN ('inherit', 'classic', 'airy') THEN LOWER(TRIM(account_visual_preset_override))
    WHEN LOWER(TRIM(COALESCE(account_visual_style, ''))) = 'enhanced' THEN 'airy'
    WHEN LOWER(TRIM(COALESCE(account_visual_style, ''))) = 'classic' THEN 'inherit'
    ELSE 'inherit'
  END;

ALTER TABLE users
  DROP COLUMN IF EXISTS account_visual_style;
