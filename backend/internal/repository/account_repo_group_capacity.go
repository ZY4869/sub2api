package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *accountRepository) ListSchedulableCapacityByGroupIDs(ctx context.Context, groupIDs []int64) ([]service.GroupAccountCapacityRow, error) {
	groupIDs = uniquePositiveInt64s(groupIDs)
	if len(groupIDs) == 0 {
		return []service.GroupAccountCapacityRow{}, nil
	}
	if r == nil || r.sql == nil {
		return r.listSchedulableCapacityByGroupIDsFallback(ctx, groupIDs)
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			ag.group_id,
			a.id AS account_id,
			a.concurrency,
			COALESCE(a.extra, '{}'::jsonb)::text AS extra,
			a.session_window_start,
			a.session_window_end,
			COALESCE(a.session_window_status, '') AS session_window_status
		FROM account_groups ag
		JOIN accounts a ON a.id = ag.account_id
		WHERE ag.group_id = ANY($1)
			AND a.deleted_at IS NULL
			AND a.status = $2
			AND a.schedulable = TRUE
			AND a.platform <> ALL($3)
			AND (a.lifecycle_state IS NULL OR a.lifecycle_state <> $4)
			AND (a.temp_unschedulable_until IS NULL OR a.temp_unschedulable_until <= $5)
			AND (a.expires_at IS NULL OR a.expires_at > $5 OR a.auto_pause_on_expired = FALSE)
			AND (a.overload_until IS NULL OR a.overload_until <= $5)
			AND (a.rate_limit_reset_at IS NULL OR a.rate_limit_reset_at <= $5)
		ORDER BY ag.group_id ASC, ag.priority ASC, a.priority ASC, a.id ASC
	`,
		pq.Array(groupIDs),
		service.StatusActive,
		pq.Array(service.UnsupportedPrimaryAccountPredicateValues()),
		service.AccountLifecycleBlacklisted,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupAccountCapacityRow, 0)
	for rows.Next() {
		var row service.GroupAccountCapacityRow
		var extraRaw string
		if err := rows.Scan(
			&row.GroupID,
			&row.AccountID,
			&row.Concurrency,
			&extraRaw,
			&row.SessionWindowStart,
			&row.SessionWindowEnd,
			&row.SessionWindowStatus,
		); err != nil {
			return nil, err
		}
		if extraRaw != "" && extraRaw != "null" {
			var extra map[string]any
			if err := json.Unmarshal([]byte(extraRaw), &extra); err != nil {
				return nil, err
			}
			row.Extra = extra
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *accountRepository) listSchedulableCapacityByGroupIDsFallback(ctx context.Context, groupIDs []int64) ([]service.GroupAccountCapacityRow, error) {
	rows := make([]service.GroupAccountCapacityRow, 0)
	if r == nil {
		return rows, nil
	}
	for _, groupID := range groupIDs {
		accounts, err := r.ListSchedulableByGroupID(ctx, groupID)
		if err != nil {
			return nil, err
		}
		for i := range accounts {
			acc := &accounts[i]
			rows = append(rows, service.GroupAccountCapacityRow{
				GroupID:             groupID,
				AccountID:           acc.ID,
				Concurrency:         acc.Concurrency,
				Extra:               copyJSONMap(acc.Extra),
				SessionWindowStart:  acc.SessionWindowStart,
				SessionWindowEnd:    acc.SessionWindowEnd,
				SessionWindowStatus: acc.SessionWindowStatus,
			})
		}
	}
	return rows, nil
}

func uniquePositiveInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
