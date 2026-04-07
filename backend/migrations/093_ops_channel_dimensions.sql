-- Migration: 093_ops_channel_dimensions
-- 1. Persist channel_id on ops_error_logs for channel-aware error analytics.
-- 2. Extend ops_metrics_hourly/daily with channel dimension for pre-aggregated ops dashboard reads.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

ALTER TABLE ops_error_logs
    ADD COLUMN IF NOT EXISTS channel_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_channel_time
    ON ops_error_logs (channel_id, created_at DESC)
    WHERE channel_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_platform_channel_time
    ON ops_error_logs (platform, channel_id, created_at DESC)
    WHERE channel_id IS NOT NULL
      AND platform IS NOT NULL
      AND platform <> '';

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_group_channel_time
    ON ops_error_logs (group_id, channel_id, created_at DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NOT NULL;

ALTER TABLE ops_metrics_hourly
    ADD COLUMN IF NOT EXISTS channel_id BIGINT;

ALTER TABLE ops_metrics_daily
    ADD COLUMN IF NOT EXISTS channel_id BIGINT;

DROP INDEX IF EXISTS idx_ops_metrics_hourly_unique_dim;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_metrics_hourly_unique_dim_channel
    ON ops_metrics_hourly (
        bucket_start,
        COALESCE(platform, ''),
        COALESCE(group_id, 0),
        COALESCE(channel_id, 0)
    );

DROP INDEX IF EXISTS idx_ops_metrics_daily_unique_dim;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_metrics_daily_unique_dim_channel
    ON ops_metrics_daily (
        bucket_date,
        COALESCE(platform, ''),
        COALESCE(group_id, 0),
        COALESCE(channel_id, 0)
    );

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_channel_bucket
    ON ops_metrics_hourly (channel_id, bucket_start DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NULL
      AND (platform IS NULL OR platform = '');

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_platform_channel_bucket
    ON ops_metrics_hourly (platform, channel_id, bucket_start DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NULL
      AND platform IS NOT NULL
      AND platform <> '';

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_group_channel_bucket
    ON ops_metrics_hourly (group_id, channel_id, bucket_start DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_channel_bucket
    ON ops_metrics_daily (channel_id, bucket_date DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NULL
      AND (platform IS NULL OR platform = '');

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_platform_channel_bucket
    ON ops_metrics_daily (platform, channel_id, bucket_date DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NULL
      AND platform IS NOT NULL
      AND platform <> '';

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_group_channel_bucket
    ON ops_metrics_daily (group_id, channel_id, bucket_date DESC)
    WHERE channel_id IS NOT NULL
      AND group_id IS NOT NULL;
