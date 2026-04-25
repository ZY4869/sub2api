package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type channelMonitorAggregationRepository struct {
	db *sql.DB
}

func NewChannelMonitorAggregationRepository(db *sql.DB) service.ChannelMonitorAggregationRepository {
	return &channelMonitorAggregationRepository{db: db}
}

func (r *channelMonitorAggregationRepository) GetWatermark(ctx context.Context) (int64, error) {
	var v int64
	if err := r.db.QueryRowContext(ctx, `
SELECT last_history_id
FROM channel_monitor_aggregation_watermark
WHERE id = 1
`).Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

func (r *channelMonitorAggregationRepository) SetWatermark(ctx context.Context, lastHistoryID int64) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO channel_monitor_aggregation_watermark (id, last_history_id, updated_at)
VALUES (1, $1, NOW())
ON CONFLICT (id)
DO UPDATE SET last_history_id = EXCLUDED.last_history_id, updated_at = NOW()
`, lastHistoryID)
	return err
}
