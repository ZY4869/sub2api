-- 103_add_channel_monitors.sql
-- Channel monitor tables (admin-managed monitors + per-run history records).

CREATE TABLE IF NOT EXISTS channel_monitors (
	id BIGSERIAL PRIMARY KEY,
	name VARCHAR(100) NOT NULL,
	provider VARCHAR(32) NOT NULL,
	endpoint TEXT NOT NULL,
	api_key_encrypted TEXT,
	interval_seconds INT NOT NULL DEFAULT 60,
	enabled BOOLEAN NOT NULL DEFAULT FALSE,
	primary_model_id TEXT NOT NULL,
	additional_model_ids TEXT[] NOT NULL DEFAULT '{}'::text[],
	last_run_at TIMESTAMPTZ,
	next_run_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT channel_monitors_unique_name UNIQUE (name),
	CONSTRAINT channel_monitors_provider_check CHECK (provider IN ('openai', 'anthropic', 'gemini', 'grok', 'antigravity')),
	CONSTRAINT channel_monitors_interval_seconds_check CHECK (interval_seconds >= 15 AND interval_seconds <= 3600)
);

CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider_id
	ON channel_monitors(provider, id);

CREATE INDEX IF NOT EXISTS idx_channel_monitors_enabled_next_run_at
	ON channel_monitors(enabled, next_run_at, id)
	WHERE enabled = true;

CREATE TABLE IF NOT EXISTS channel_monitor_histories (
	id BIGSERIAL PRIMARY KEY,
	monitor_id BIGINT NOT NULL REFERENCES channel_monitors(id) ON DELETE CASCADE,
	model_id TEXT NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'success',
	response_text TEXT NOT NULL DEFAULT '',
	error_message TEXT NOT NULL DEFAULT '',
	http_status INT,
	latency_ms BIGINT NOT NULL DEFAULT 0,
	started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	finished_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_histories_monitor_created_at
	ON channel_monitor_histories(monitor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_histories_monitor_model_created_at
	ON channel_monitor_histories(monitor_id, model_id, created_at DESC);
