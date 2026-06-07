package repository

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *apiKeyRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	row, err := client.APIKey.Query().
		Where(apikey.IDEQ(id), apikey.DeletedAtIsNil()).
		Select(apikey.FieldID, apikey.FieldUserID, apikey.FieldKey, apikey.FieldName).
		Only(ctx)
	if err != nil {
		if !dbent.IsNotFound(err) {
			return err
		}
		exists, err := client.APIKey.Query().
			Where(apikey.IDEQ(id)).
			Exist(mixins.SkipSoftDelete(ctx))
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		return service.ErrAPIKeyNotFound
	}

	deletedAt := time.Now()
	// 存在唯一键约束，生成 tombstone key 用来释放原 key。
	tombstoneKey := fmt.Sprintf("__deleted__%d__%d", id, deletedAt.UnixNano())
	affected, err := client.APIKey.Update().
		Where(apikey.IDEQ(id), apikey.DeletedAtIsNil()).
		SetKey(tombstoneKey).
		SetDeletedAt(deletedAt).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrAPIKeyNotFound
		}
		return err
	}
	if affected == 0 {
		exists, err := client.APIKey.Query().
			Where(apikey.IDEQ(id)).
			Exist(mixins.SkipSoftDelete(ctx))
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		return service.ErrAPIKeyNotFound
	}
	if err := r.recordDeletedAPIKeyAudit(ctx, row.ID, row.UserID, row.Name, row.Key, deletedAt); err != nil {
		return err
	}
	return nil
}

func (r *apiKeyRepository) DeleteByUserID(ctx context.Context, userID int64) ([]string, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.APIKey.Query().
		Where(apikey.UserIDEQ(userID), apikey.DeletedAtIsNil()).
		Select(apikey.FieldID, apikey.FieldKey, apikey.FieldName, apikey.FieldUserID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.Key)
		if err := r.Delete(ctx, row.ID); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func (r *apiKeyRepository) recordDeletedAPIKeyAudit(ctx context.Context, id, userID int64, name, key string, deletedAt time.Time) error {
	exec := apiKeyGroupSQLExecutor(ctx, r)
	if exec == nil || id <= 0 || userID <= 0 {
		return nil
	}
	_, err := exec.ExecContext(ctx, `
		INSERT INTO deleted_api_key_audits (api_key_id, user_id, name, key_prefix, deleted_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (api_key_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			name = EXCLUDED.name,
			key_prefix = EXCLUDED.key_prefix,
			deleted_at = EXCLUDED.deleted_at
	`, id, userID, name, safeDeletedAPIKeyAuditPrefix(key), deletedAt.UTC())
	return err
}

func safeDeletedAPIKeyAuditPrefix(key string) string {
	const maxPrefix = 12
	if len(key) <= maxPrefix {
		return key
	}
	return key[:maxPrefix]
}
