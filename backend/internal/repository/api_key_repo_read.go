package repository

import (
	"context"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *apiKeyRepository) GetByID(ctx context.Context, id int64) (*service.APIKey, error) {
	m, err := r.activeQuery().
		Where(apikey.IDEQ(id)).
		WithUser().
		WithGroup().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	out := apiKeyEntityToService(m)
	if err := r.hydrateAPIKeyUserBalances(ctx, []*service.APIKey{out}); err != nil {
		return nil, err
	}
	if err := r.hydrateAPIKeyGroups(ctx, []*service.APIKey{out}); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *apiKeyRepository) GetByIDAllowDeleted(ctx context.Context, id int64) (*service.APIKey, error) {
	m, err := r.client.APIKey.Query().
		Where(apikey.IDEQ(id)).
		WithUser().
		WithGroup().
		Only(mixins.SkipSoftDelete(ctx))
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	out := apiKeyEntityToService(m)
	if err := r.hydrateAPIKeyUserBalances(mixins.SkipSoftDelete(ctx), []*service.APIKey{out}); err != nil {
		return nil, err
	}
	if err := r.hydrateAPIKeyGroups(mixins.SkipSoftDelete(ctx), []*service.APIKey{out}); err != nil {
		return nil, err
	}
	return out, nil
}

// GetKeyAndOwnerID 根据 API Key ID 获取其 key 与所有者（用户）ID。
// 相比 GetByID，此方法性能更优，因为：
//   - 使用 Select() 只查询必要字段，减少数据传输量
//   - 不加载完整的 API Key 实体及其关联数据（User、Group 等）
//   - 适用于删除等只需 key 与用户 ID 的场景

func (r *apiKeyRepository) GetKeyAndOwnerID(ctx context.Context, id int64) (string, int64, error) {
	m, err := r.activeQuery().
		Where(apikey.IDEQ(id)).
		Select(apikey.FieldKey, apikey.FieldUserID).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return "", 0, service.ErrAPIKeyNotFound
		}
		return "", 0, err
	}
	return m.Key, m.UserID, nil
}

func (r *apiKeyRepository) GetByKey(ctx context.Context, key string) (*service.APIKey, error) {
	m, err := r.activeQuery().
		Where(apikey.KeyEQ(key)).
		WithUser().
		WithGroup().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	out := apiKeyEntityToService(m)
	if err := r.hydrateAPIKeyUserBalances(ctx, []*service.APIKey{out}); err != nil {
		return nil, err
	}
	if err := r.hydrateAPIKeyGroups(ctx, []*service.APIKey{out}); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *apiKeyRepository) GetByKeyForAuth(ctx context.Context, key string) (*service.APIKey, error) {
	m, err := r.activeQuery().
		Where(apikey.KeyEQ(key)).
		Select(
			apikey.FieldID,
			apikey.FieldUserID,
			apikey.FieldGroupID,
			apikey.FieldModelDisplayMode,
			apikey.FieldStatus,
			apikey.FieldIPWhitelist,
			apikey.FieldIPBlacklist,
			apikey.FieldImageOnlyEnabled,
			apikey.FieldImageCountBillingEnabled,
			apikey.FieldImageMaxCount,
			apikey.FieldImageCountWeights,
			apikey.FieldQuota,
			apikey.FieldQuotaUsed,
			apikey.FieldQuotaUsedByCurrency,
			apikey.FieldExpiresAt,
			apikey.FieldStartsAt,
			apikey.FieldAccessTimePolicy,
			apikey.FieldRateLimit5h,
			apikey.FieldRateLimit1d,
			apikey.FieldRateLimit7d,
		).
		WithUser(func(q *dbent.UserQuery) {
			q.Select(
				user.FieldID,
				user.FieldStatus,
				user.FieldRole,
				user.FieldBalance,
				user.FieldConcurrency,
				user.FieldAPIKeyAccessTimePolicy,
			)
		}).
		WithGroup(func(q *dbent.GroupQuery) {
			q.Select(
				group.FieldID,
				group.FieldName,
				group.FieldPlatform,
				group.FieldPriority,
				group.FieldStatus,
				group.FieldSubscriptionType,
				group.FieldRateMultiplier,
				group.FieldDailyLimitUsd,
				group.FieldWeeklyLimitUsd,
				group.FieldMonthlyLimitUsd,
				group.FieldImagePrice1k,
				group.FieldImagePrice2k,
				group.FieldImagePrice4k,
				group.FieldImageProtocolMode,
				group.FieldClaudeCodeOnly,
				group.FieldFallbackGroupID,
				group.FieldFallbackGroupIDOnInvalidRequest,
				group.FieldModelRoutingEnabled,
				group.FieldGeminiMixedProtocolEnabled,
				group.FieldModelRouting,
				group.FieldMcpXMLInject,
				group.FieldSupportedModelScopes,
				group.FieldAllowMessagesDispatch,
				group.FieldDefaultMappedModel,
			)
		}).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	out := apiKeyEntityToService(m)
	if err := r.hydrateAPIKeyUserBalances(ctx, []*service.APIKey{out}); err != nil {
		return nil, err
	}
	if err := r.hydrateAPIKeyGroups(ctx, []*service.APIKey{out}); err != nil {
		return nil, err
	}
	return out, nil
}
