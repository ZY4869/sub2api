-- 118_add_airwallex_payments.sql
-- Built-in payment orders, events, and refunds for Airwallex clean-room integration.

CREATE TABLE IF NOT EXISTS payment_orders (
	id BIGSERIAL PRIMARY KEY,
	order_no TEXT NOT NULL UNIQUE,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	product_type TEXT NOT NULL,
	status TEXT NOT NULL,
	provider TEXT NOT NULL,
	provider_env TEXT NOT NULL,
	amount_minor BIGINT NOT NULL,
	currency TEXT NOT NULL,
	country_code TEXT,
	provider_intent_id TEXT,
	resume_token_hash TEXT NOT NULL UNIQUE,
	idempotency_key_hash TEXT,
	snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
	paid_at TIMESTAMPTZ,
	refunded_at TIMESTAMPTZ,
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE payment_orders
	ADD COLUMN IF NOT EXISTS order_no TEXT,
	ADD COLUMN IF NOT EXISTS product_type TEXT,
	ADD COLUMN IF NOT EXISTS provider_env TEXT,
	ADD COLUMN IF NOT EXISTS amount_minor BIGINT,
	ADD COLUMN IF NOT EXISTS resume_token_hash TEXT,
	ADD COLUMN IF NOT EXISTS idempotency_key_hash TEXT,
	ADD COLUMN IF NOT EXISTS snapshot_json JSONB DEFAULT '{}'::jsonb,
	ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ,
	ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

DO $$
DECLARE
	has_payment_env BOOLEAN;
	has_amount BOOLEAN;
	has_merchant_order_id BOOLEAN;
	has_resume_token BOOLEAN;
	has_order_snapshot BOOLEAN;
	has_completed_at BOOLEAN;
BEGIN
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'payment_orders' AND column_name = 'payment_env'
	) INTO has_payment_env;
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'payment_orders' AND column_name = 'amount'
	) INTO has_amount;
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'payment_orders' AND column_name = 'merchant_order_id'
	) INTO has_merchant_order_id;
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'payment_orders' AND column_name = 'resume_token'
	) INTO has_resume_token;
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'payment_orders' AND column_name = 'order_snapshot'
	) INTO has_order_snapshot;
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'payment_orders' AND column_name = 'completed_at'
	) INTO has_completed_at;

	EXECUTE format(
		'UPDATE payment_orders
		SET
			order_no = COALESCE(NULLIF(order_no, ''''), %s, ''legacy_payment_'' || id::text),
			product_type = COALESCE(NULLIF(product_type, ''''), ''balance_topup''),
			provider_env = COALESCE(NULLIF(provider_env, ''''), %s, ''demo''),
			amount_minor = COALESCE(amount_minor, %s, 0),
			resume_token_hash = COALESCE(NULLIF(resume_token_hash, ''''), %s, ''legacy_resume_'' || id::text),
			snapshot_json = COALESCE(snapshot_json, %s, ''{}''::jsonb),
			paid_at = COALESCE(paid_at, %s)
		WHERE order_no IS NULL
			OR product_type IS NULL
			OR provider_env IS NULL
			OR amount_minor IS NULL
			OR resume_token_hash IS NULL
			OR snapshot_json IS NULL
			OR paid_at IS NULL',
		CASE WHEN has_merchant_order_id THEN 'NULLIF(merchant_order_id, '''')' ELSE 'NULL' END,
		CASE WHEN has_payment_env THEN 'NULLIF(payment_env, '''')' ELSE 'NULL' END,
		CASE WHEN has_amount THEN 'ROUND(COALESCE(amount, 0)::numeric * 100)::bigint' ELSE 'NULL' END,
		CASE WHEN has_resume_token THEN 'NULLIF(resume_token, '''')' ELSE 'NULL' END,
		CASE WHEN has_order_snapshot THEN 'order_snapshot' ELSE 'NULL' END,
		CASE WHEN has_completed_at THEN 'completed_at' ELSE 'NULL' END
	);
END $$;

ALTER TABLE payment_orders
	ALTER COLUMN order_no SET NOT NULL,
	ALTER COLUMN product_type SET NOT NULL,
	ALTER COLUMN provider_env SET NOT NULL,
	ALTER COLUMN amount_minor SET NOT NULL,
	ALTER COLUMN resume_token_hash SET NOT NULL,
	ALTER COLUMN snapshot_json SET NOT NULL;

DO $$
DECLARE
	col TEXT;
BEGIN
	FOREACH col IN ARRAY ARRAY[
		'payment_env',
		'amount',
		'credited_amount',
		'credited_currency',
		'merchant_order_id',
		'provider_payment_link_id',
		'provider_refund_id',
		'resume_token',
		'checkout_url',
		'return_url',
		'cancel_url',
		'provider_snapshot',
		'order_snapshot',
		'last_error'
	]
	LOOP
		IF EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
				AND table_name = 'payment_orders'
				AND column_name = col
		) THEN
			EXECUTE format('ALTER TABLE payment_orders ALTER COLUMN %I DROP NOT NULL', col);
		END IF;
	END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_order_no
	ON payment_orders(order_no);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_resume_token_hash
	ON payment_orders(resume_token_hash);

CREATE INDEX IF NOT EXISTS idx_payment_orders_user_created_at
	ON payment_orders(user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_payment_orders_status_created_at
	ON payment_orders(status, created_at DESC, id DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_provider_intent
	ON payment_orders(provider, provider_intent_id)
	WHERE provider_intent_id IS NOT NULL AND provider_intent_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_user_idempotency
	ON payment_orders(user_id, idempotency_key_hash)
	WHERE idempotency_key_hash IS NOT NULL AND idempotency_key_hash <> '';

CREATE TABLE IF NOT EXISTS payment_events (
	id BIGSERIAL PRIMARY KEY,
	provider TEXT NOT NULL,
	provider_event_id TEXT NOT NULL,
	order_no TEXT,
	event_type TEXT NOT NULL,
	event_status TEXT NOT NULL,
	payload_hash TEXT NOT NULL,
	payload_redacted_json JSONB NOT NULL DEFAULT '{}'::jsonb,
	processed_at TIMESTAMPTZ,
	error_reason TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_events_provider_event
	ON payment_events(provider, provider_event_id);

CREATE INDEX IF NOT EXISTS idx_payment_events_order_created_at
	ON payment_events(order_no, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS payment_refunds (
	id BIGSERIAL PRIMARY KEY,
	refund_no TEXT NOT NULL UNIQUE,
	order_no TEXT NOT NULL REFERENCES payment_orders(order_no) ON DELETE CASCADE,
	provider_refund_id TEXT,
	amount_minor BIGINT NOT NULL,
	currency TEXT NOT NULL,
	reason TEXT,
	status TEXT NOT NULL,
	requested_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
	idempotency_key_hash TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_refunds_provider_refund
	ON payment_refunds(provider_refund_id)
	WHERE provider_refund_id IS NOT NULL AND provider_refund_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_refunds_order_idempotency
	ON payment_refunds(order_no, idempotency_key_hash)
	WHERE idempotency_key_hash IS NOT NULL AND idempotency_key_hash <> '';

CREATE INDEX IF NOT EXISTS idx_payment_refunds_order_created_at
	ON payment_refunds(order_no, created_at DESC, id DESC);

ALTER TABLE user_affiliate_ledger
	ADD COLUMN IF NOT EXISTS payment_order_id BIGINT REFERENCES payment_orders(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_payment_topup_dedup
	ON user_affiliate_ledger(event_type, payment_order_id)
	WHERE event_type = 'topup_accrue' AND payment_order_id IS NOT NULL;
