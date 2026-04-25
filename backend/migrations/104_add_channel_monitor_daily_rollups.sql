-- 104_add_channel_monitor_daily_rollups.sql
-- Daily rollups to support 7/15/30-day availability stats without heavy history scans.

CREATE TABLE IF NOT EXISTS channel_monitor_daily_rollups (
	id BIGSERIAL PRIMARY KEY,
	monitor_id BIGINT NOT NULL REFERENCES channel_monitors(id) ON DELETE CASCADE,
	model_id TEXT NOT NULL,
	day DATE NOT NULL,
	total_checks INT NOT NULL DEFAULT 0,
	available_checks INT NOT NULL DEFAULT 0,
	degraded_checks INT NOT NULL DEFAULT 0,
	total_latency_ms BIGINT NOT NULL DEFAULT 0,
	max_latency_ms BIGINT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT channel_monitor_daily_rollups_unique UNIQUE (monitor_id, model_id, day)
);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_daily_rollups_monitor_day
	ON channel_monitor_daily_rollups(monitor_id, day DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_daily_rollups_day
	ON channel_monitor_daily_rollups(day DESC, id DESC);

CREATE TABLE IF NOT EXISTS channel_monitor_aggregation_watermark (
	id INT PRIMARY KEY DEFAULT 1,
	last_history_id BIGINT NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO channel_monitor_aggregation_watermark (id, last_history_id)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;
