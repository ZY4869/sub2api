-- Track TTFT sample counts separately from success counts so pre-aggregated
-- dashboards do not over-weight non-streaming successes.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

ALTER TABLE ops_system_metrics
    ADD COLUMN IF NOT EXISTS ttft_sample_count BIGINT NOT NULL DEFAULT 0;

ALTER TABLE ops_metrics_hourly
    ADD COLUMN IF NOT EXISTS ttft_sample_count BIGINT NOT NULL DEFAULT 0;

ALTER TABLE ops_metrics_daily
    ADD COLUMN IF NOT EXISTS ttft_sample_count BIGINT NOT NULL DEFAULT 0;
