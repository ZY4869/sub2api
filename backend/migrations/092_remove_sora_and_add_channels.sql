-- Migration: 092_remove_sora_and_add_channels
-- 1. Remove Sora runtime data and schema leftovers.
-- 2. Introduce channel management tables used by admin channels CRUD.

-- =========================
-- Remove Sora data
-- =========================
DELETE FROM account_groups
WHERE account_id IN (SELECT id FROM accounts WHERE platform = 'sora');

DELETE FROM accounts
WHERE platform = 'sora';

DROP TABLE IF EXISTS sora_generations;
DROP TABLE IF EXISTS sora_accounts;

ALTER TABLE users
	DROP COLUMN IF EXISTS sora_storage_quota_bytes,
	DROP COLUMN IF EXISTS sora_storage_used_bytes;

ALTER TABLE groups
	DROP COLUMN IF EXISTS sora_image_price_360,
	DROP COLUMN IF EXISTS sora_image_price_540,
	DROP COLUMN IF EXISTS sora_video_price_per_request,
	DROP COLUMN IF EXISTS sora_video_price_per_request_hd,
	DROP COLUMN IF EXISTS sora_storage_quota_bytes;

ALTER TABLE usage_logs
	DROP COLUMN IF EXISTS media_type;

DELETE FROM settings
WHERE key = 'sora_client_enabled'
   OR key LIKE 'sora_%';

-- =========================
-- Channel management
-- =========================
CREATE TABLE IF NOT EXISTS channels (
	id BIGSERIAL PRIMARY KEY,
	name VARCHAR(100) NOT NULL,
	description TEXT,
	status VARCHAR(20) NOT NULL DEFAULT 'active',
	restrict_models BOOLEAN NOT NULL DEFAULT FALSE,
	billing_model_source VARCHAR(32) NOT NULL DEFAULT 'channel_mapped',
	model_mapping JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT channels_unique_name UNIQUE (name)
);

CREATE INDEX IF NOT EXISTS idx_channels_status_created_at
	ON channels(status, created_at DESC);

CREATE TABLE IF NOT EXISTS channel_groups (
	channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
	group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (channel_id, group_id),
	CONSTRAINT channel_groups_unique_group UNIQUE (group_id)
);

CREATE INDEX IF NOT EXISTS idx_channel_groups_channel_id
	ON channel_groups(channel_id);

CREATE TABLE IF NOT EXISTS channel_model_pricing (
	id BIGSERIAL PRIMARY KEY,
	channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
	platform VARCHAR(50) NOT NULL,
	models TEXT[] NOT NULL DEFAULT '{}'::text[],
	billing_mode VARCHAR(32) NOT NULL DEFAULT 'token',
	input_price DECIMAL(20,10),
	output_price DECIMAL(20,10),
	cache_write_price DECIMAL(20,10),
	cache_read_price DECIMAL(20,10),
	image_output_price DECIMAL(20,10),
	per_request_price DECIMAL(20,10),
	sort_order INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channel_model_pricing_channel_id
	ON channel_model_pricing(channel_id, platform, sort_order, id);

CREATE TABLE IF NOT EXISTS channel_pricing_intervals (
	id BIGSERIAL PRIMARY KEY,
	pricing_id BIGINT NOT NULL REFERENCES channel_model_pricing(id) ON DELETE CASCADE,
	min_tokens BIGINT NOT NULL DEFAULT 0,
	max_tokens BIGINT,
	tier_label VARCHAR(100) NOT NULL DEFAULT '',
	input_price DECIMAL(20,10),
	output_price DECIMAL(20,10),
	cache_write_price DECIMAL(20,10),
	cache_read_price DECIMAL(20,10),
	per_request_price DECIMAL(20,10),
	sort_order INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channel_pricing_intervals_pricing_id
	ON channel_pricing_intervals(pricing_id, sort_order, id);

ALTER TABLE usage_logs
	ADD COLUMN IF NOT EXISTS channel_id BIGINT REFERENCES channels(id) ON DELETE SET NULL,
	ADD COLUMN IF NOT EXISTS model_mapping_chain TEXT,
	ADD COLUMN IF NOT EXISTS billing_tier VARCHAR(100),
	ADD COLUMN IF NOT EXISTS billing_mode VARCHAR(32),
	ADD COLUMN IF NOT EXISTS image_output_tokens INT,
	ADD COLUMN IF NOT EXISTS image_output_cost DECIMAL(20,10);

CREATE INDEX IF NOT EXISTS idx_usage_logs_channel_id_created_at
	ON usage_logs(channel_id, created_at DESC);
