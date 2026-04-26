-- 107_add_user_affiliate_ledger.sql
-- Affiliate rebate ledger for auditing + deduplication.

CREATE TABLE IF NOT EXISTS user_affiliate_ledger (
	id BIGSERIAL PRIMARY KEY,
	inviter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	invitee_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
	event_type TEXT NOT NULL,
	amount NUMERIC(20, 8) NOT NULL,
	base_amount NUMERIC(20, 8),
	rate_percent NUMERIC(10, 4),
	frozen_until TIMESTAMPTZ,
	request_id TEXT,
	api_key_id BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
	redeem_code_id BIGINT REFERENCES redeem_codes(id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_inviter_created_at
	ON user_affiliate_ledger(inviter_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_invitee_created_at
	ON user_affiliate_ledger(invitee_user_id, created_at DESC, id DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_usage_dedup
	ON user_affiliate_ledger(event_type, api_key_id, request_id)
	WHERE event_type = 'usage_accrue';

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_topup_dedup
	ON user_affiliate_ledger(event_type, redeem_code_id)
	WHERE event_type = 'topup_accrue';

