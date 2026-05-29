package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) GetRequestTraceSummary(ctx context.Context, filter *service.OpsRequestTraceFilter) (*service.OpsRequestTraceSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	schema, err := r.getOpsRequestTraceSchema(ctx)
	if err != nil {
		return nil, err
	}
	_, _, startTime, endTime := filter.Normalize()
	filterCopy := &service.OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.StartTime = &startTime
	filterCopy.EndTime = &endTime

	where, args := buildOpsRequestTracesWhereWithSchema(filterCopy, schema)
	summary := &service.OpsRequestTraceSummary{
		StartTime: startTime,
		EndTime:   endTime,
	}

	totalsSQL := `
SELECT
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) < 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) >= 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.stream, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_tools, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_thinking, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.raw_available, false))::bigint,
  COALESCE(AVG(COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.50) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.95) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.99) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0)
FROM ops_request_traces t
` + where
	if err := r.db.QueryRowContext(ctx, totalsSQL, args...).Scan(
		&summary.Totals.RequestCount,
		&summary.Totals.SuccessCount,
		&summary.Totals.ErrorCount,
		&summary.Totals.StreamCount,
		&summary.Totals.ToolCount,
		&summary.Totals.ThinkingCount,
		&summary.Totals.RawAvailableCount,
		&summary.Totals.AvgDurationMs,
		&summary.Totals.P50DurationMs,
		&summary.Totals.P95DurationMs,
		&summary.Totals.P99DurationMs,
	); err != nil {
		return nil, err
	}

	bucketSeconds := opsRequestTraceBucketSeconds(endTime.Sub(startTime))
	trendSQL := `
SELECT
  to_timestamp(floor(extract(epoch from t.created_at) / $1) * $1) AT TIME ZONE 'UTC' AS bucket_start,
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) >= 400)::bigint,
  COALESCE(PERCENTILE_DISC(0.50) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.95) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.99) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0)
FROM ops_request_traces t
` + shiftSQLPlaceholders(where, 1) + `
GROUP BY 1
ORDER BY 1 ASC`
	trendArgs := append([]any{bucketSeconds}, args...)
	rows, err := r.db.QueryContext(ctx, trendSQL, trendArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	summary.Trend = make([]*service.OpsRequestTraceSummaryPoint, 0, 64)
	for rows.Next() {
		point := &service.OpsRequestTraceSummaryPoint{}
		if err := rows.Scan(
			&point.BucketStart,
			&point.RequestCount,
			&point.ErrorCount,
			&point.P50DurationMs,
			&point.P95DurationMs,
			&point.P99DurationMs,
		); err != nil {
			return nil, err
		}
		summary.Trend = append(summary.Trend, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	summary.StatusDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(t.status, '')", "COALESCE(t.status, '')", 8)
	if err != nil {
		return nil, err
	}
	summary.FinishReasonDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(NULLIF(t.finish_reason, ''), 'unknown')", "COALESCE(NULLIF(t.finish_reason, ''), 'unknown')", 10)
	if err != nil {
		return nil, err
	}
	summary.ProtocolPairDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(NULLIF(t.protocol_in, ''), 'unknown') || ' -> ' || COALESCE(NULLIF(t.protocol_out, ''), 'unknown')", "COALESCE(NULLIF(t.protocol_in, ''), 'unknown') || ' -> ' || COALESCE(NULLIF(t.protocol_out, ''), 'unknown')", 10)
	if err != nil {
		return nil, err
	}
	summary.ModelDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(NULLIF(t.requested_model, ''), NULLIF(t.upstream_model, ''), 'unknown')", "COALESCE(NULLIF(t.requested_model, ''), NULLIF(t.upstream_model, ''), 'unknown')", 10)
	if err != nil {
		return nil, err
	}
	summary.CapabilityDistribution, err = r.listRequestTraceCapabilityBreakdown(ctx, where, args)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *opsRepository) listRequestTraceSummaryBreakdown(ctx context.Context, where string, args []any, keyExpr string, labelExpr string, limit int) ([]*service.OpsRequestTraceSummaryBreakdownItem, error) {
	query := `
SELECT key, label, count
FROM (
  SELECT
    ` + keyExpr + ` AS key,
    ` + labelExpr + ` AS label,
    COUNT(*)::bigint AS count
  FROM ops_request_traces t
` + where + `
  GROUP BY 1,2
) s
WHERE key <> ''
ORDER BY count DESC, label ASC
LIMIT $` + itoa(len(args)+1)
	rows, err := r.db.QueryContext(ctx, query, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.OpsRequestTraceSummaryBreakdownItem, 0, limit)
	for rows.Next() {
		item := &service.OpsRequestTraceSummaryBreakdownItem{}
		if err := rows.Scan(&item.Key, &item.Label, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *opsRepository) listRequestTraceCapabilityBreakdown(ctx context.Context, where string, args []any) ([]*service.OpsRequestTraceSummaryBreakdownItem, error) {
	query := `
SELECT
  COUNT(*) FILTER (WHERE COALESCE(t.stream, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_tools, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_thinking, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.raw_available, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.count_tokens_source, '') = 'estimated')::bigint
FROM ops_request_traces t
` + where
	var streamCount, toolCount, thinkingCount, rawCount, estimatedCount int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&streamCount, &toolCount, &thinkingCount, &rawCount, &estimatedCount); err != nil {
		return nil, err
	}
	return []*service.OpsRequestTraceSummaryBreakdownItem{
		{Key: "stream", Label: "stream", Count: streamCount},
		{Key: "tools", Label: "tools", Count: toolCount},
		{Key: "thinking", Label: "thinking", Count: thinkingCount},
		{Key: "raw", Label: "raw", Count: rawCount},
		{Key: "estimated_count_tokens", Label: "estimated_count_tokens", Count: estimatedCount},
	}, nil
}
