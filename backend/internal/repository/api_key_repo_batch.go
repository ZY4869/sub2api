package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *apiKeyRepository) getAPIKeysByIDs(ctx context.Context, ids []int64, withUser bool) ([]service.APIKey, error) {
	if len(ids) == 0 {
		return []service.APIKey{}, nil
	}
	q := r.activeQuery().Where(apikey.IDIn(ids...)).WithGroup()
	if withUser {
		q = q.WithUser()
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	serviceKeys := make([]*service.APIKey, 0, len(rows))
	keyByID := make(map[int64]*service.APIKey, len(rows))
	for _, row := range rows {
		item := apiKeyEntityToService(row)
		serviceKeys = append(serviceKeys, item)
		keyByID[item.ID] = item
	}
	if err := r.hydrateAPIKeyUserBalances(ctx, serviceKeys); err != nil {
		return nil, err
	}
	if err := r.hydrateAPIKeyGroups(ctx, serviceKeys); err != nil {
		return nil, err
	}
	out := make([]service.APIKey, 0, len(ids))
	for _, id := range ids {
		if item := keyByID[id]; item != nil {
			out = append(out, *item)
		}
	}
	return out, nil
}
