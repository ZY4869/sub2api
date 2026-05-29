package repository

import (
	"context"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/apikeygroup"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *apiKeyRepository) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	q := r.activeQuery().Where(apikey.UserIDEQ(userID))

	// Apply filters
	if filters.Search != "" {
		q = q.Where(apikey.Or(
			apikey.NameContainsFold(filters.Search),
			apikey.KeyContainsFold(filters.Search),
		))
	}
	if filters.Status != "" {
		q = q.Where(apikey.StatusEQ(filters.Status))
	}
	if filters.GroupID != nil {
		if *filters.GroupID == 0 {
			q = q.Where(apikey.Not(apikey.HasAPIKeyGroups()))
		} else {
			q = q.Where(apikey.HasAPIKeyGroupsWith(apikeygroup.GroupIDEQ(*filters.GroupID)))
		}
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	keys, err := q.
		WithGroup().
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(apikey.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	outKeys := make([]service.APIKey, 0, len(keys))
	serviceKeys := make([]*service.APIKey, 0, len(keys))
	for i := range keys {
		item := apiKeyEntityToService(keys[i])
		serviceKeys = append(serviceKeys, item)
		outKeys = append(outKeys, *item)
	}
	if err := r.hydrateAPIKeyGroups(ctx, serviceKeys); err != nil {
		return nil, nil, err
	}
	for i := range serviceKeys {
		outKeys[i] = *serviceKeys[i]
	}

	return outKeys, paginationResultFromTotal(int64(total), params), nil
}

func (r *apiKeyRepository) VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	if len(apiKeyIDs) == 0 {
		return []int64{}, nil
	}

	ids, err := r.client.APIKey.Query().
		Where(apikey.UserIDEQ(userID), apikey.IDIn(apiKeyIDs...), apikey.DeletedAtIsNil()).
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *apiKeyRepository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	count, err := r.activeQuery().Where(apikey.UserIDEQ(userID)).Count(ctx)
	return int64(count), err
}

func (r *apiKeyRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	count, err := r.activeQuery().Where(apikey.KeyEQ(key)).Count(ctx)
	return count > 0, err
}

func (r *apiKeyRepository) ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error) {
	ids, total, err := listAPIKeyIDsByGroupID(ctx, r.sql, groupID, params)
	if err != nil {
		return nil, nil, err
	}
	keys, err := r.getAPIKeysByIDs(ctx, ids, true)
	if err != nil {
		return nil, nil, err
	}
	return keys, paginationResultFromTotal(total, params), nil
}

// SearchAPIKeys searches API keys by user ID and/or keyword (name)

func (r *apiKeyRepository) SearchAPIKeys(ctx context.Context, userID int64, keyword string, limit int) ([]service.APIKey, error) {
	q := r.activeQuery()
	if userID > 0 {
		q = q.Where(apikey.UserIDEQ(userID))
	}

	if keyword != "" {
		q = q.Where(apikey.NameContainsFold(keyword))
	}

	keys, err := q.Limit(limit).Order(dbent.Desc(apikey.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}

	outKeys := make([]service.APIKey, 0, len(keys))
	serviceKeys := make([]*service.APIKey, 0, len(keys))
	for i := range keys {
		item := apiKeyEntityToService(keys[i])
		serviceKeys = append(serviceKeys, item)
		outKeys = append(outKeys, *item)
	}
	if err := r.hydrateAPIKeyGroups(ctx, serviceKeys); err != nil {
		return nil, err
	}
	for i := range serviceKeys {
		outKeys[i] = *serviceKeys[i]
	}
	return outKeys, nil
}

// ClearGroupIDByGroupID 将指定分组的所有 API Key 的 group_id 设为 nil
