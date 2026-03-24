package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

var errAPIKeyGroupSchemaOutdated = infraerrors.ServiceUnavailable(
	"API_KEY_GROUP_SCHEMA_OUTDATED",
	"api key group bindings require the latest database migration; restart the service or apply the latest migrations",
)

type apiKeyGroupBindingWriter interface {
	GetAPIKeyGroups(ctx context.Context, keyID int64) ([]service.APIKeyGroupBinding, error)
	SetAPIKeyGroups(ctx context.Context, keyID int64, bindings []service.APIKeyGroupBinding) error
	IncrementAPIKeyGroupQuotaUsed(ctx context.Context, keyID, groupID int64, amount float64) error
	RecomputeShadowGroupIDs(ctx context.Context, keyIDs []int64) error
}

type sqlTxStarter interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (r *apiKeyRepository) GetAPIKeyGroups(ctx context.Context, keyID int64) ([]service.APIKeyGroupBinding, error) {
	exec := apiKeyGroupSQLExecutor(ctx, r)
	if exec == nil {
		return nil, nil
	}
	groupMap, err := r.loadAPIKeyGroupBindingsMap(ctx, exec, []int64{keyID})
	if err != nil {
		return nil, err
	}
	return append([]service.APIKeyGroupBinding(nil), groupMap[keyID]...), nil
}

func (r *apiKeyRepository) SetAPIKeyGroups(ctx context.Context, keyID int64, bindings []service.APIKeyGroupBinding) error {
	exec := apiKeyGroupSQLExecutor(ctx, r)
	if exec == nil {
		return fmt.Errorf("sql executor is not configured")
	}

	supported, err := supportsAPIKeyGroupBindingsSchema(ctx, exec)
	if err != nil {
		return err
	}
	if !supported {
		if !canUseLegacyAPIKeyGroupShadow(bindings) {
			return errAPIKeyGroupSchemaOutdated.WithMetadata(map[string]string{
				"feature": "api_key_group_bindings",
			})
		}
		exec, commit, rollback, err := beginAPIKeyGroupSQLTx(ctx, exec)
		if err != nil {
			return err
		}
		defer rollback()
		if err := setLegacyAPIKeyGroupShadow(ctx, exec, keyID, bindings); err != nil {
			return err
		}
		return commit()
	}

	exec, commit, rollback, err := beginAPIKeyGroupSQLTx(ctx, exec)
	if err != nil {
		return err
	}
	defer rollback()

	if _, err := exec.ExecContext(ctx, `DELETE FROM api_key_groups WHERE api_key_id = $1`, keyID); err != nil {
		return err
	}
	for _, binding := range bindings {
		modelPatternsJSON, err := json.Marshal(binding.ModelPatterns)
		if err != nil {
			return fmt.Errorf("marshal model patterns: %w", err)
		}
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO api_key_groups (api_key_id, group_id, quota, quota_used, model_patterns, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5::jsonb, NOW(), NOW())`,
			keyID,
			binding.GroupID,
			binding.Quota,
			binding.QuotaUsed,
			string(modelPatternsJSON),
		); err != nil {
			return err
		}
	}
	if err := recomputeAPIKeyShadowGroup(ctx, exec, keyID); err != nil {
		return err
	}
	return commit()
}

func (r *apiKeyRepository) IncrementAPIKeyGroupQuotaUsed(ctx context.Context, keyID, groupID int64, amount float64) error {
	exec := apiKeyGroupSQLExecutor(ctx, r)
	if exec == nil {
		return fmt.Errorf("sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
		UPDATE api_key_groups
		SET quota_used = quota_used + $1,
			updated_at = NOW()
		WHERE api_key_id = $2 AND group_id = $3
	`, amount, keyID, groupID)
	return err
}

func (r *apiKeyRepository) RecomputeShadowGroupIDs(ctx context.Context, keyIDs []int64) error {
	exec := apiKeyGroupSQLExecutor(ctx, r)
	if exec == nil {
		return nil
	}
	if len(keyIDs) == 0 {
		return nil
	}
	supported, err := supportsAPIKeyGroupBindingsSchema(ctx, exec)
	if err != nil {
		return err
	}
	if !supported {
		return nil
	}
	for _, keyID := range keyIDs {
		if err := recomputeAPIKeyShadowGroup(ctx, exec, keyID); err != nil {
			return err
		}
	}
	return nil
}

func (r *apiKeyRepository) hydrateAPIKeyGroups(ctx context.Context, keys []*service.APIKey) error {
	exec := apiKeyGroupSQLExecutor(ctx, r)
	if exec == nil {
		for _, key := range keys {
			if key != nil {
				key.SyncLegacyGroupShadow()
			}
		}
		return nil
	}
	if len(keys) == 0 {
		return nil
	}
	keyIDs := make([]int64, 0, len(keys))
	keyByID := make(map[int64]*service.APIKey, len(keys))
	for _, key := range keys {
		if key == nil {
			continue
		}
		keyIDs = append(keyIDs, key.ID)
		keyByID[key.ID] = key
	}
	groupMap, err := r.loadAPIKeyGroupBindingsMap(ctx, exec, keyIDs)
	if err != nil {
		return err
	}
	for keyID, apiKey := range keyByID {
		apiKey.GroupBindings = append([]service.APIKeyGroupBinding(nil), groupMap[keyID]...)
		apiKey.SyncLegacyGroupShadow()
	}
	return nil
}

func (r *apiKeyRepository) loadAPIKeyGroupBindingsMap(ctx context.Context, exec sqlExecutor, keyIDs []int64) (map[int64][]service.APIKeyGroupBinding, error) {
	out := make(map[int64][]service.APIKeyGroupBinding, len(keyIDs))
	if r == nil || exec == nil || len(keyIDs) == 0 {
		return out, nil
	}

	rows, err := exec.QueryContext(ctx, `
		SELECT api_key_id, group_id, quota, quota_used, model_patterns, created_at, updated_at
		FROM api_key_groups
		WHERE api_key_id = ANY($1)
		ORDER BY api_key_id ASC, group_id ASC
	`, pq.Array(keyIDs))
	if err != nil {
		if isAPIKeyGroupSchemaMissingError(err) {
			return out, nil
		}
		return nil, err
	}
	defer rows.Close()

	groupIDs := make([]int64, 0)
	seenGroupIDs := make(map[int64]struct{})
	for rows.Next() {
		var (
			apiKeyID         int64
			groupID          int64
			quota            float64
			quotaUsed        float64
			modelPatternsRaw []byte
			createdAt        sql.NullTime
			updatedAt        sql.NullTime
		)
		if err := rows.Scan(&apiKeyID, &groupID, &quota, &quotaUsed, &modelPatternsRaw, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		binding := service.APIKeyGroupBinding{
			APIKeyID:  apiKeyID,
			GroupID:   groupID,
			Quota:     quota,
			QuotaUsed: quotaUsed,
			CreatedAt: createdAt.Time,
			UpdatedAt: updatedAt.Time,
		}
		if len(modelPatternsRaw) > 0 {
			_ = json.Unmarshal(modelPatternsRaw, &binding.ModelPatterns)
		}
		out[apiKeyID] = append(out[apiKeyID], binding)
		if _, exists := seenGroupIDs[groupID]; !exists {
			seenGroupIDs[groupID] = struct{}{}
			groupIDs = append(groupIDs, groupID)
		}
	}
	if err := rows.Err(); err != nil {
		if isAPIKeyGroupSchemaMissingError(err) {
			return out, nil
		}
		return nil, err
	}

	groupMap, err := r.loadAPIKeyBindingGroups(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	for apiKeyID := range out {
		filtered := out[apiKeyID][:0]
		for _, binding := range out[apiKeyID] {
			binding.Group = groupMap[binding.GroupID]
			if binding.Group == nil {
				continue
			}
			filtered = append(filtered, binding)
		}
		sort.Slice(filtered, func(i, j int) bool {
			left := filtered[i]
			right := filtered[j]
			leftPriority := 1
			rightPriority := 1
			if left.Group != nil {
				leftPriority = left.Group.Priority
			}
			if right.Group != nil {
				rightPriority = right.Group.Priority
			}
			if leftPriority != rightPriority {
				return leftPriority < rightPriority
			}
			return left.GroupID < right.GroupID
		})
		out[apiKeyID] = filtered
	}

	return out, nil
}

func apiKeyGroupSQLExecutor(ctx context.Context, repo *apiKeyRepository) sqlExecutor {
	if repo == nil {
		return nil
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		if exec, ok := any(tx.Client()).(sqlExecutor); ok {
			return exec
		}
		if exec, ok := any(tx).(sqlExecutor); ok {
			return exec
		}
	}
	return repo.sql
}

func (r *apiKeyRepository) loadAPIKeyBindingGroups(ctx context.Context, groupIDs []int64) (map[int64]*service.Group, error) {
	out := make(map[int64]*service.Group, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	groups, err := clientFromContext(ctx, r.client).Group.Query().
		Where(group.IDIn(groupIDs...), group.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range groups {
		out[item.ID] = groupEntityToService(item)
	}
	return out, nil
}

func beginAPIKeyGroupSQLTx(ctx context.Context, exec sqlExecutor) (sqlExecutor, func() error, func(), error) {
	if tx, ok := exec.(*sql.Tx); ok {
		return tx, func() error { return nil }, func() {}, nil
	}
	starter, ok := exec.(sqlTxStarter)
	if !ok {
		return nil, nil, nil, fmt.Errorf("sql executor does not support transactions")
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return tx, tx.Commit, func() { _ = tx.Rollback() }, nil
}

func recomputeAPIKeyShadowGroup(ctx context.Context, exec sqlExecutor, keyID int64) error {
	if _, err := exec.ExecContext(ctx, `
		UPDATE api_keys AS ak
		SET group_id = sub.group_id,
			updated_at = NOW()
		FROM (
			SELECT ag.group_id
			FROM api_key_groups ag
			JOIN groups g ON g.id = ag.group_id
			WHERE ag.api_key_id = $1 AND g.deleted_at IS NULL
			ORDER BY g.priority ASC, ag.group_id ASC
			LIMIT 1
		) AS sub
		WHERE ak.id = $1 AND ak.deleted_at IS NULL
	`, keyID); err != nil {
		return err
	}
	_, err := exec.ExecContext(ctx, `
		UPDATE api_keys
		SET group_id = NULL,
			updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM api_key_groups WHERE api_key_id = $1)
	`, keyID)
	return err
}

func listAPIKeyIDsByGroupID(ctx context.Context, exec sqlExecutor, groupID int64, params pagination.PaginationParams) ([]int64, int64, error) {
	var total int64
	if err := exec.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ag.api_key_id)
		FROM api_key_groups ag
		JOIN api_keys ak ON ak.id = ag.api_key_id
		WHERE ag.group_id = $1 AND ak.deleted_at IS NULL
	`, groupID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT DISTINCT ag.api_key_id
		FROM api_key_groups ag
		JOIN api_keys ak ON ak.id = ag.api_key_id
		WHERE ag.group_id = $1 AND ak.deleted_at IS NULL
		ORDER BY ag.api_key_id DESC
		OFFSET $2 LIMIT $3
	`, groupID, params.Offset(), params.Limit())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return ids, total, nil
}

func listAPIKeyIDsByUserAndGroup(ctx context.Context, exec sqlExecutor, userID, oldGroupID int64) ([]int64, error) {
	rows, err := exec.QueryContext(ctx, `
		SELECT DISTINCT ag.api_key_id
		FROM api_key_groups ag
		JOIN api_keys ak ON ak.id = ag.api_key_id
		WHERE ak.user_id = $1
		  AND ag.group_id = $2
		  AND ak.deleted_at IS NULL
	`, userID, oldGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func supportsAPIKeyGroupBindingsSchema(ctx context.Context, exec sqlExecutor) (bool, error) {
	exists, err := tableExistsSQL(ctx, exec, "api_key_groups")
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	requiredColumns := map[string][]string{
		"api_key_groups": {"api_key_id", "group_id", "quota", "quota_used", "model_patterns"},
		"groups":         {"priority"},
	}
	for tableName, columns := range requiredColumns {
		for _, columnName := range columns {
			ok, err := columnExistsSQL(ctx, exec, tableName, columnName)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}
	return true, nil
}

func tableExistsSQL(ctx context.Context, exec sqlExecutor, tableName string) (bool, error) {
	var exists bool
	if err := exec.QueryRowContext(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, tableName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func columnExistsSQL(ctx context.Context, exec sqlExecutor, tableName, columnName string) (bool, error) {
	var exists bool
	if err := exec.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func canUseLegacyAPIKeyGroupShadow(bindings []service.APIKeyGroupBinding) bool {
	if len(bindings) > 1 {
		return false
	}
	if len(bindings) == 0 {
		return true
	}
	binding := bindings[0]
	return binding.Quota <= 0 && binding.QuotaUsed <= 0 && len(binding.ModelPatterns) == 0
}

func setLegacyAPIKeyGroupShadow(ctx context.Context, exec sqlExecutor, keyID int64, bindings []service.APIKeyGroupBinding) error {
	var groupID any
	if len(bindings) > 0 {
		groupID = bindings[0].GroupID
	}
	_, err := exec.ExecContext(ctx, `
		UPDATE api_keys
		SET group_id = $2,
			updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`, keyID, groupID)
	return err
}

func isAPIKeyGroupSchemaMissingError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "42P01" && pgErr.Code != "42703" {
		return false
	}
	msg := strings.ToLower(pgErr.Message)
	return strings.Contains(msg, "api_key_groups") ||
		(strings.Contains(msg, "priority") && strings.Contains(msg, "group"))
}
