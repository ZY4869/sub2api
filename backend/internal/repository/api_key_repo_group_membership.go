package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func (r *apiKeyRepository) ClearGroupIDByGroupID(ctx context.Context, groupID int64) (int64, error) {
	exec, commit, rollback, err := beginAPIKeyGroupSQLTx(ctx, apiKeyGroupSQLExecutor(ctx, r))
	if err != nil {
		return 0, err
	}
	defer rollback()
	keyIDs, _, err := listAPIKeyIDsByGroupID(ctx, exec, groupID, pagination.PaginationParams{Page: 1, PageSize: 1000000})
	if err != nil {
		return 0, err
	}
	res, err := exec.ExecContext(ctx, `DELETE FROM api_key_groups WHERE group_id = $1`, groupID)
	if err != nil {
		return 0, err
	}
	for _, keyID := range keyIDs {
		if err := recomputeAPIKeyShadowGroup(ctx, exec, keyID); err != nil {
			return 0, err
		}
	}
	if err := commit(); err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	return affected, err
}

// UpdateGroupIDByUserAndGroup 将用户下绑定 oldGroupID 的所有 Key 迁移到 newGroupID

func (r *apiKeyRepository) CountByGroupID(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := scanSingleRow(ctx, r.sql, `
		SELECT COUNT(DISTINCT ag.api_key_id)
		FROM api_key_groups ag
		JOIN api_keys ak ON ak.id = ag.api_key_id
		WHERE ag.group_id = $1 AND ak.deleted_at IS NULL
	`, []any{groupID}, &count)
	return count, err
}

func (r *apiKeyRepository) ListKeysByUserID(ctx context.Context, userID int64) ([]string, error) {
	keys, err := r.activeQuery().
		Where(apikey.UserIDEQ(userID)).
		Select(apikey.FieldKey).
		Strings(ctx)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *apiKeyRepository) ListKeysByGroupID(ctx context.Context, groupID int64) (keys []string, err error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT DISTINCT ak.key
		FROM api_key_groups ag
		JOIN api_keys ak ON ak.id = ag.api_key_id
		WHERE ag.group_id = $1 AND ak.deleted_at IS NULL
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	keys = make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	err = rows.Err()
	return keys, err
}

// IncrementQuotaUsed 使用 Ent 原子递增 quota_used 字段并返回新值
