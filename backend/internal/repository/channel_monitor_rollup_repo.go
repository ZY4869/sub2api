package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelMonitorRollupRepository struct {
	db *sql.DB
}

func NewChannelMonitorRollupRepository(db *sql.DB) service.ChannelMonitorRollupRepository {
	return &channelMonitorRollupRepository{db: db}
}

func (r *channelMonitorRollupRepository) UpsertIncrement(
	ctx context.Context,
	monitorID int64,
	modelID string,
	day time.Time,
	deltaTotal int64,
	deltaAvailable int64,
	deltaDegraded int64,
	deltaLatency int64,
	maxLatencyCandidate int64,
) error {
	if monitorID <= 0 || modelID == "" {
		return errors.New("invalid rollup key")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO channel_monitor_daily_rollups (
	monitor_id,
	model_id,
	day,
	total_checks,
	available_checks,
	degraded_checks,
	total_latency_ms,
	max_latency_ms,
	created_at,
	updated_at
)
VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, NOW(), NOW())
ON CONFLICT (monitor_id, model_id, day)
DO UPDATE SET
	total_checks = channel_monitor_daily_rollups.total_checks + EXCLUDED.total_checks,
	available_checks = channel_monitor_daily_rollups.available_checks + EXCLUDED.available_checks,
	degraded_checks = channel_monitor_daily_rollups.degraded_checks + EXCLUDED.degraded_checks,
	total_latency_ms = channel_monitor_daily_rollups.total_latency_ms + EXCLUDED.total_latency_ms,
	max_latency_ms = GREATEST(channel_monitor_daily_rollups.max_latency_ms, EXCLUDED.max_latency_ms),
	updated_at = NOW()
`, monitorID, modelID, day.UTC().Format("2006-01-02"), deltaTotal, deltaAvailable, deltaDegraded, deltaLatency, maxLatencyCandidate)
	return err
}

func (r *channelMonitorRollupRepository) SumAvailability(ctx context.Context, monitorIDs []int64, startDay time.Time) (map[int64]map[string]*service.ChannelMonitorDailyRollup, error) {
	if len(monitorIDs) == 0 {
		return map[int64]map[string]*service.ChannelMonitorDailyRollup{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
	monitor_id,
	model_id,
	SUM(total_checks) AS total_checks,
	SUM(available_checks) AS available_checks,
	SUM(degraded_checks) AS degraded_checks,
	SUM(total_latency_ms) AS total_latency_ms,
	MAX(max_latency_ms) AS max_latency_ms
FROM channel_monitor_daily_rollups
WHERE monitor_id = ANY($1)
  AND day >= $2::date
GROUP BY monitor_id, model_id
`, pq.Array(monitorIDs), startDay.UTC().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[int64]map[string]*service.ChannelMonitorDailyRollup{}
	for rows.Next() {
		rp := &service.ChannelMonitorDailyRollup{}
		if err := rows.Scan(
			&rp.MonitorID,
			&rp.ModelID,
			&rp.TotalChecks,
			&rp.AvailableChecks,
			&rp.DegradedChecks,
			&rp.TotalLatencyMs,
			&rp.MaxLatencyMs,
		); err != nil {
			return nil, err
		}
		if out[rp.MonitorID] == nil {
			out[rp.MonitorID] = map[string]*service.ChannelMonitorDailyRollup{}
		}
		out[rp.MonitorID][rp.ModelID] = rp
	}
	return out, rows.Err()
}

func (r *channelMonitorRollupRepository) SumAvailabilityWindows(ctx context.Context, monitorID int64, start7 time.Time, start15 time.Time, start30 time.Time) (map[string]*service.ChannelMonitorAvailabilityWindows, error) {
	if monitorID <= 0 {
		return nil, errors.New("invalid monitor id")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
	model_id,
	SUM(total_checks) FILTER (WHERE day >= $2::date) AS total_7,
	SUM(available_checks) FILTER (WHERE day >= $2::date) AS available_7,
	SUM(degraded_checks) FILTER (WHERE day >= $2::date) AS degraded_7,
	SUM(total_checks) FILTER (WHERE day >= $3::date) AS total_15,
	SUM(available_checks) FILTER (WHERE day >= $3::date) AS available_15,
	SUM(degraded_checks) FILTER (WHERE day >= $3::date) AS degraded_15,
	SUM(total_checks) FILTER (WHERE day >= $4::date) AS total_30,
	SUM(available_checks) FILTER (WHERE day >= $4::date) AS available_30,
	SUM(degraded_checks) FILTER (WHERE day >= $4::date) AS degraded_30
FROM channel_monitor_daily_rollups
WHERE monitor_id = $1
  AND day >= $4::date
GROUP BY model_id
`, monitorID, start7.UTC().Format("2006-01-02"), start15.UTC().Format("2006-01-02"), start30.UTC().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]*service.ChannelMonitorAvailabilityWindows{}
	for rows.Next() {
		var (
			modelID                      string
			total7, avail7, degraded7    sql.NullInt64
			total15, avail15, degraded15 sql.NullInt64
			total30, avail30, degraded30 sql.NullInt64
		)
		if err := rows.Scan(
			&modelID,
			&total7,
			&avail7,
			&degraded7,
			&total15,
			&avail15,
			&degraded15,
			&total30,
			&avail30,
			&degraded30,
		); err != nil {
			return nil, err
		}

		out[modelID] = &service.ChannelMonitorAvailabilityWindows{
			Last7: service.ChannelMonitorAvailabilityCounts{
				TotalChecks:     nullInt64ToInt64(total7),
				AvailableChecks: nullInt64ToInt64(avail7),
				DegradedChecks:  nullInt64ToInt64(degraded7),
			},
			Last15: service.ChannelMonitorAvailabilityCounts{
				TotalChecks:     nullInt64ToInt64(total15),
				AvailableChecks: nullInt64ToInt64(avail15),
				DegradedChecks:  nullInt64ToInt64(degraded15),
			},
			Last30: service.ChannelMonitorAvailabilityCounts{
				TotalChecks:     nullInt64ToInt64(total30),
				AvailableChecks: nullInt64ToInt64(avail30),
				DegradedChecks:  nullInt64ToInt64(degraded30),
			},
		}
	}
	return out, rows.Err()
}

func (r *channelMonitorRollupRepository) PruneBeforeDay(ctx context.Context, beforeDay time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
DELETE FROM channel_monitor_daily_rollups
WHERE day < $1::date
`, beforeDay.UTC().Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func nullInt64ToInt64(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}
