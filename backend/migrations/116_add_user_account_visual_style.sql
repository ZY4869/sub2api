ALTER TABLE users
  ADD COLUMN IF NOT EXISTS account_visual_style VARCHAR(32) NOT NULL DEFAULT 'classic';
