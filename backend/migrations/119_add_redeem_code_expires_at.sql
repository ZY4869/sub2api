-- Add optional redeem code expiration separate from subscription validity_days.
ALTER TABLE redeem_codes
ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_redeem_codes_expires_at
ON redeem_codes (expires_at);
