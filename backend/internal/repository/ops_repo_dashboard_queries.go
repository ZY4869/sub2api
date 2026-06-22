package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) queryUsageCounts(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) (successCount int64, tokenConsumed int64, err error) {
	join, where, args, _ := buildUsageWhere(filter, start, end, 1)
	totalTokensExpr := usageTotalTokensSQL("ul.")

	q := `
SELECT
  COALESCE(COUNT(*), 0) AS success_count,
  COALESCE(SUM(` + totalTokensExpr + `), 0) AS token_consumed
FROM usage_logs ul
` + join + `
` + where

	var tokens sql.NullInt64
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&successCount, &tokens); err != nil {
		return 0, 0, err
	}
	if tokens.Valid {
		tokenConsumed = tokens.Int64
	}
	return successCount, tokenConsumed, nil
}

func (r *opsRepository) queryUsageLatency(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) (duration service.OpsPercentiles, ttft service.OpsPercentiles, ttftSampleCount int64, err error) {
	join, where, args, _ := buildUsageWhere(filter, start, end, 1)
	q := `
SELECT
  percentile_cont(0.50) WITHIN GROUP (ORDER BY duration_ms) FILTER (WHERE duration_ms IS NOT NULL) AS duration_p50,
  percentile_cont(0.90) WITHIN GROUP (ORDER BY duration_ms) FILTER (WHERE duration_ms IS NOT NULL) AS duration_p90,
  percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) FILTER (WHERE duration_ms IS NOT NULL) AS duration_p95,
  percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms) FILTER (WHERE duration_ms IS NOT NULL) AS duration_p99,
  AVG(duration_ms) FILTER (WHERE duration_ms IS NOT NULL) AS duration_avg,
  MAX(duration_ms) AS duration_max,
  percentile_cont(0.50) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL) AS ttft_p50,
  percentile_cont(0.90) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL) AS ttft_p90,
  percentile_cont(0.95) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL) AS ttft_p95,
  percentile_cont(0.99) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL) AS ttft_p99,
  AVG(first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL) AS ttft_avg,
  MAX(first_token_ms) AS ttft_max,
  COALESCE(COUNT(first_token_ms), 0) AS ttft_sample_count
FROM usage_logs ul
` + join + `
` + where

	var dP50, dP90, dP95, dP99 sql.NullFloat64
	var dAvg sql.NullFloat64
	var dMax sql.NullInt64
	var tP50, tP90, tP95, tP99 sql.NullFloat64
	var tAvg sql.NullFloat64
	var tMax sql.NullInt64
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&dP50, &dP90, &dP95, &dP99, &dAvg, &dMax,
		&tP50, &tP90, &tP95, &tP99, &tAvg, &tMax,
		&ttftSampleCount,
	); err != nil {
		return service.OpsPercentiles{}, service.OpsPercentiles{}, 0, err
	}

	duration.P50 = floatToIntPtr(dP50)
	duration.P90 = floatToIntPtr(dP90)
	duration.P95 = floatToIntPtr(dP95)
	duration.P99 = floatToIntPtr(dP99)
	duration.Avg = floatToIntPtr(dAvg)
	if dMax.Valid {
		v := int(dMax.Int64)
		duration.Max = &v
	}

	ttft.P50 = floatToIntPtr(tP50)
	ttft.P90 = floatToIntPtr(tP90)
	ttft.P95 = floatToIntPtr(tP95)
	ttft.P99 = floatToIntPtr(tP99)
	ttft.Avg = floatToIntPtr(tAvg)
	if tMax.Valid {
		v := int(tMax.Int64)
		ttft.Max = &v
	}

	return duration, ttft, ttftSampleCount, nil
}

func (r *opsRepository) queryErrorCounts(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) (
	errorTotal int64,
	businessLimited int64,
	errorCountSLA int64,
	upstreamExcl429529 int64,
	upstream429 int64,
	upstream529 int64,
	err error,
) {
	where, args, _ := buildErrorWhere(filter, start, end, 1)

	q := `
SELECT
  COALESCE(COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400), 0) AS error_total,
  COALESCE(COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400 AND is_business_limited), 0) AS business_limited,
  COALESCE(COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400 AND NOT is_business_limited), 0) AS error_sla,
  COALESCE(COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND COALESCE(upstream_status_code, status_code, 0) NOT IN (429, 529)), 0) AS upstream_excl,
  COALESCE(COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND COALESCE(upstream_status_code, status_code, 0) = 429), 0) AS upstream_429,
  COALESCE(COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND COALESCE(upstream_status_code, status_code, 0) = 529), 0) AS upstream_529
FROM ops_error_logs
` + where

	if err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&errorTotal,
		&businessLimited,
		&errorCountSLA,
		&upstreamExcl429529,
		&upstream429,
		&upstream529,
	); err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	return errorTotal, businessLimited, errorCountSLA, upstreamExcl429529, upstream429, upstream529, nil
}

func (r *opsRepository) queryCurrentRates(ctx context.Context, filter *service.OpsDashboardFilter, end time.Time) (qpsCurrent float64, tpsCurrent float64, err error) {
	windowStart := end.Add(-1 * time.Minute)

	successCount1m, token1m, err := r.queryUsageCounts(ctx, filter, windowStart, end)
	if err != nil {
		return 0, 0, err
	}
	errorCount1m, _, _, _, _, _, err := r.queryErrorCounts(ctx, filter, windowStart, end)
	if err != nil {
		return 0, 0, err
	}

	qpsCurrent = roundTo1DP(float64(successCount1m+errorCount1m) / 60.0)
	tpsCurrent = roundTo1DP(float64(token1m) / 60.0)
	return qpsCurrent, tpsCurrent, nil
}

func (r *opsRepository) queryPeakRates(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) (qpsPeak float64, tpsPeak float64, err error) {
	usageJoin, usageWhere, usageArgs, next := buildUsageWhere(filter, start, end, 1)
	errorWhere, errorArgs, _ := buildErrorWhere(filter, start, end, next)
	totalTokensExpr := usageTotalTokensSQL("ul.")

	q := `
WITH usage_buckets AS (
  SELECT
    date_trunc('minute', ul.created_at) AS bucket,
    COUNT(*) AS req_cnt,
    COALESCE(SUM(` + totalTokensExpr + `), 0) AS token_cnt
  FROM usage_logs ul
  ` + usageJoin + `
  ` + usageWhere + `
  GROUP BY 1
),
error_buckets AS (
  SELECT date_trunc('minute', created_at) AS bucket, COUNT(*) AS err_cnt
  FROM ops_error_logs
  ` + errorWhere + `
    AND COALESCE(status_code, 0) >= 400
  GROUP BY 1
),
combined AS (
  SELECT COALESCE(u.bucket, e.bucket) AS bucket,
         COALESCE(u.req_cnt, 0) + COALESCE(e.err_cnt, 0) AS total_req,
         COALESCE(u.token_cnt, 0) AS total_tokens
  FROM usage_buckets u
  FULL OUTER JOIN error_buckets e ON u.bucket = e.bucket
)
SELECT
  COALESCE(MAX(total_req), 0) AS max_req_per_min,
  COALESCE(MAX(total_tokens), 0) AS max_tokens_per_min
FROM combined`

	args := append(usageArgs, errorArgs...)

	var maxReqPerMinute, maxTokensPerMinute sql.NullInt64
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&maxReqPerMinute, &maxTokensPerMinute); err != nil {
		return 0, 0, err
	}
	if maxReqPerMinute.Valid && maxReqPerMinute.Int64 > 0 {
		qpsPeak = roundTo1DP(float64(maxReqPerMinute.Int64) / 60.0)
	}
	if maxTokensPerMinute.Valid && maxTokensPerMinute.Int64 > 0 {
		tpsPeak = roundTo1DP(float64(maxTokensPerMinute.Int64) / 60.0)
	}
	return qpsPeak, tpsPeak, nil
}
