package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelMonitorHistoryRepository struct {
	db *sql.DB
}

func NewChannelMonitorHistoryRepository(db *sql.DB) service.ChannelMonitorHistoryRepository {
	return &channelMonitorHistoryRepository{db: db}
}

func (r *channelMonitorHistoryRepository) Create(ctx context.Context, history *service.ChannelMonitorHistory) (*service.ChannelMonitorHistory, error) {
	if history == nil {
		return nil, errors.New("nil history")
	}

	var httpStatus sql.NullInt64
	if history.HTTPStatus != nil {
		httpStatus = sql.NullInt64{Int64: int64(*history.HTTPStatus), Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO channel_monitor_histories (
	monitor_id,
	model_id,
	status,
	response_text,
	error_message,
	http_status,
	latency_ms,
	started_at,
	finished_at,
	created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
RETURNING
	id,
	created_at
`,
		history.MonitorID,
		history.ModelID,
		history.Status,
		history.ResponseText,
		history.ErrorMessage,
		nullInt64Value(httpStatus),
		history.LatencyMs,
		history.StartedAt,
		history.FinishedAt,
	)
	if err := row.Scan(&history.ID, &history.CreatedAt); err != nil {
		return nil, err
	}
	return history, nil
}

func (r *channelMonitorHistoryRepository) ListByMonitorID(ctx context.Context, monitorID int64, limit int) ([]*service.ChannelMonitorHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
	id,
	monitor_id,
	model_id,
	status,
	response_text,
	error_message,
	http_status,
	latency_ms,
	started_at,
	finished_at,
	created_at
FROM channel_monitor_histories
WHERE monitor_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2
`, monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorHistoryRows(rows)
}

func (r *channelMonitorHistoryRepository) ListLatestByMonitorIDs(ctx context.Context, monitorIDs []int64) ([]*service.ChannelMonitorHistory, error) {
	if len(monitorIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT ON (monitor_id, model_id)
	id,
	monitor_id,
	model_id,
	status,
	response_text,
	error_message,
	http_status,
	latency_ms,
	started_at,
	finished_at,
	created_at
FROM channel_monitor_histories
WHERE monitor_id = ANY($1)
ORDER BY monitor_id, model_id, created_at DESC, id DESC
`, pq.Array(monitorIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorHistoryRows(rows)
}

func (r *channelMonitorHistoryRepository) ListPrimaryTimelineByMonitorIDs(ctx context.Context, monitorIDs []int64, limitPerMonitor int) ([]*service.ChannelMonitorHistory, error) {
	if len(monitorIDs) == 0 {
		return nil, nil
	}
	if limitPerMonitor <= 0 {
		limitPerMonitor = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
	id,
	monitor_id,
	model_id,
	status,
	response_text,
	error_message,
	http_status,
	latency_ms,
	started_at,
	finished_at,
	created_at
FROM (
	SELECT
		h.*,
		ROW_NUMBER() OVER (PARTITION BY h.monitor_id ORDER BY h.created_at DESC, h.id DESC) AS rn
	FROM channel_monitor_histories h
	INNER JOIN channel_monitors m
		ON m.id = h.monitor_id
	   AND h.model_id = m.primary_model_id
	WHERE h.monitor_id = ANY($1)
) ranked
WHERE rn <= $2
ORDER BY monitor_id, created_at DESC, id DESC
`, pq.Array(monitorIDs), limitPerMonitor)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorHistoryRows(rows)
}

func (r *channelMonitorHistoryRepository) ListLatestByMonitorID(ctx context.Context, monitorID int64) ([]*service.ChannelMonitorHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT ON (model_id)
	id,
	monitor_id,
	model_id,
	status,
	response_text,
	error_message,
	http_status,
	latency_ms,
	started_at,
	finished_at,
	created_at
FROM channel_monitor_histories
WHERE monitor_id = $1
ORDER BY model_id, created_at DESC, id DESC
`, monitorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorHistoryRows(rows)
}

func (r *channelMonitorHistoryRepository) ListForAggregation(ctx context.Context, afterID int64, limit int) ([]*service.ChannelMonitorHistory, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
	id,
	monitor_id,
	model_id,
	status,
	response_text,
	error_message,
	http_status,
	latency_ms,
	started_at,
	finished_at,
	created_at
FROM channel_monitor_histories
WHERE id > $1
ORDER BY id ASC
LIMIT $2
`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorHistoryRows(rows)
}

func (r *channelMonitorHistoryRepository) PruneBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
DELETE FROM channel_monitor_histories
WHERE created_at < $1
`, before)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func scanChannelMonitorHistoryRows(rows *sql.Rows) ([]*service.ChannelMonitorHistory, error) {
	var out []*service.ChannelMonitorHistory
	for rows.Next() {
		h := &service.ChannelMonitorHistory{}
		var httpStatus sql.NullInt64
		if err := rows.Scan(
			&h.ID,
			&h.MonitorID,
			&h.ModelID,
			&h.Status,
			&h.ResponseText,
			&h.ErrorMessage,
			&httpStatus,
			&h.LatencyMs,
			&h.StartedAt,
			&h.FinishedAt,
			&h.CreatedAt,
		); err != nil {
			return nil, err
		}
		if httpStatus.Valid {
			v := int(httpStatus.Int64)
			h.HTTPStatus = &v
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func nullInt64Value(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}
