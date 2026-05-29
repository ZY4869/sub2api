package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func saveGroupVisibleModelPatterns(ctx context.Context, exec sqlExecutor, groupID int64, patterns []string) error {
	if exec == nil || groupID <= 0 {
		return nil
	}
	normalized := service.NormalizeGroupVisibleModelPatterns(patterns)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `
		UPDATE groups
		SET visible_model_patterns = $2::jsonb
		WHERE id = $1 AND deleted_at IS NULL
	`, groupID, string(payload))
	if err != nil && isGroupVisibleModelPatternsMissingError(err) && len(normalized) == 0 {
		return nil
	}
	return err
}

func hydrateVisibleModelPatternsForGroups(ctx context.Context, exec sqlExecutor, groups []*service.Group) error {
	if exec == nil || len(groups) == 0 {
		return nil
	}
	groupByID := make(map[int64]*service.Group, len(groups))
	ids := make([]int64, 0, len(groups))
	for _, group := range groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		if _, exists := groupByID[group.ID]; exists {
			continue
		}
		groupByID[group.ID] = group
		ids = append(ids, group.ID)
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := exec.QueryContext(ctx, `
		SELECT id, COALESCE(visible_model_patterns, '[]'::jsonb)
		FROM groups
		WHERE id = ANY($1) AND deleted_at IS NULL
	`, pq.Array(ids))
	if err != nil {
		if isGroupVisibleModelPatternsMissingError(err) {
			return nil
		}
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int64
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		group := groupByID[id]
		if group == nil {
			continue
		}
		var values []string
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &values)
		}
		group.VisibleModelPatterns = service.NormalizeGroupVisibleModelPatterns(values)
	}
	return rows.Err()
}

func hydrateVisibleModelPatternsForGroupValues(ctx context.Context, exec sqlExecutor, groups []service.Group) error {
	if len(groups) == 0 {
		return nil
	}
	ptrs := make([]*service.Group, 0, len(groups))
	for i := range groups {
		ptrs = append(ptrs, &groups[i])
	}
	return hydrateVisibleModelPatternsForGroups(ctx, exec, ptrs)
}

func isGroupVisibleModelPatternsMissingError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) || pgErr.Code != "42703" {
		return false
	}
	msg := strings.ToLower(pgErr.Message)
	return strings.Contains(msg, "visible_model_patterns")
}
