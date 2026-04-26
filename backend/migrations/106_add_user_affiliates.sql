-- 106_add_user_affiliates.sql
-- Affiliate invite rebate: per-user affiliate row + inviter binding + balances.

CREATE TABLE IF NOT EXISTS user_affiliates (
	user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	aff_code TEXT NOT NULL UNIQUE,
	inviter_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
	inviter_bound_at TIMESTAMPTZ,
	invitee_count INT NOT NULL DEFAULT 0,
	rebate_balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
	rebate_frozen_balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
	lifetime_rebate NUMERIC(20, 8) NOT NULL DEFAULT 0,
	custom_rebate_rate_percent NUMERIC(10, 4),
	custom_aff_code BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_affiliates_inviter_user_id
	ON user_affiliates(inviter_user_id, user_id);

