package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsHourlyMetricsRow struct {
	bucketStart time.Time
	channelID   sql.NullInt64

	successCount         int64
	errorCountTotal      int64
	businessLimitedCount int64
	errorCountSLA        int64

	upstreamErrorCountExcl429529 int64
	upstream429Count             int64
	upstream529Count             int64

	tokenConsumed int64

	durationP50 sql.NullInt64
	durationP90 sql.NullInt64
	durationP95 sql.NullInt64
	durationP99 sql.NullInt64
	durationAvg sql.NullFloat64
	durationMax sql.NullInt64

	ttftP50 sql.NullInt64
	ttftP90 sql.NullInt64
	ttftP95 sql.NullInt64
	ttftP99 sql.NullInt64
	ttftAvg sql.NullFloat64
	ttftMax sql.NullInt64
}

func (r *opsRepository) listHourlyMetricsRows(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) ([]opsHourlyMetricsRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return []opsHourlyMetricsRow{}, nil
	}

	where := "bucket_start >= $1 AND bucket_start < $2"
	args := []any{start.UTC(), end.UTC()}
	idx := 3

	platform := ""
	groupID := (*int64)(nil)
	channelID := (*int64)(nil)
	if filter != nil {
		platform = strings.TrimSpace(strings.ToLower(filter.Platform))
		groupID = filter.GroupID
		channelID = filter.ChannelID
	}

	switch {
	case groupID != nil && *groupID > 0 && channelID != nil && *channelID > 0:
		where += fmt.Sprintf(" AND group_id = $%d", idx)
		args = append(args, *groupID)
		idx++
		where += fmt.Sprintf(" AND channel_id = $%d", idx)
		args = append(args, *channelID)
		idx++
		if platform != "" {
			where += fmt.Sprintf(" AND platform = $%d", idx)
			args = append(args, platform)
		}
	case groupID != nil && *groupID > 0:
		where += fmt.Sprintf(" AND group_id = $%d", idx)
		args = append(args, *groupID)
		idx++
		where += " AND channel_id IS NULL"
		if platform != "" {
			where += fmt.Sprintf(" AND platform = $%d", idx)
			args = append(args, platform)
		}
	case channelID != nil && *channelID > 0 && platform != "":
		where += fmt.Sprintf(" AND platform = $%d", idx)
		args = append(args, platform)
		idx++
		where += " AND group_id IS NULL"
		where += fmt.Sprintf(" AND channel_id = $%d", idx)
		args = append(args, *channelID)
	case channelID != nil && *channelID > 0:
		where += fmt.Sprintf(" AND channel_id = $%d", idx)
		args = append(args, *channelID)
		where += " AND platform IS NULL AND group_id IS NULL"
	case platform != "":
		where += fmt.Sprintf(" AND platform = $%d AND group_id IS NULL", idx)
		args = append(args, platform)
		where += " AND channel_id IS NULL"
	default:
		where += " AND platform IS NULL AND group_id IS NULL AND channel_id IS NULL"
	}

	q := `
SELECT
  bucket_start,
  channel_id,
  success_count,
  error_count_total,
  business_limited_count,
  error_count_sla,
  upstream_error_count_excl_429_529,
  upstream_429_count,
  upstream_529_count,
  token_consumed,
  duration_p50_ms,
  duration_p90_ms,
  duration_p95_ms,
  duration_p99_ms,
  duration_avg_ms,
  duration_max_ms,
  ttft_p50_ms,
  ttft_p90_ms,
  ttft_p95_ms,
  ttft_p99_ms,
  ttft_avg_ms,
  ttft_max_ms
FROM ops_metrics_hourly
WHERE ` + where + `
ORDER BY bucket_start ASC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]opsHourlyMetricsRow, 0, 64)
	for rows.Next() {
		var row opsHourlyMetricsRow
		if err := rows.Scan(
			&row.bucketStart,
			&row.channelID,
			&row.successCount,
			&row.errorCountTotal,
			&row.businessLimitedCount,
			&row.errorCountSLA,
			&row.upstreamErrorCountExcl429529,
			&row.upstream429Count,
			&row.upstream529Count,
			&row.tokenConsumed,
			&row.durationP50,
			&row.durationP90,
			&row.durationP95,
			&row.durationP99,
			&row.durationAvg,
			&row.durationMax,
			&row.ttftP50,
			&row.ttftP90,
			&row.ttftP95,
			&row.ttftP99,
			&row.ttftAvg,
			&row.ttftMax,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
