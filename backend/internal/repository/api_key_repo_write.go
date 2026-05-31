package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"time"
)

func (r *apiKeyRepository) Create(ctx context.Context, key *service.APIKey) error {
	client := clientFromContext(ctx, r.client)
	builder := client.APIKey.Create().
		SetUserID(key.UserID).
		SetKey(key.Key).
		SetName(key.Name).
		SetModelDisplayMode(service.NormalizeAPIKeyModelDisplayMode(key.ModelDisplayMode)).
		SetStatus(key.Status).
		SetNillableGroupID(key.GroupID).
		SetNillableLastUsedAt(key.LastUsedAt).
		SetImageOnlyEnabled(key.ImageOnlyEnabled).
		SetImageCountBillingEnabled(key.ImageCountBillingEnabled).
		SetImageMaxCount(key.ImageMaxCount).
		SetImageCountUsed(key.ImageCountUsed).
		SetImageCountWeights(service.NormalizeAPIKeyImageCountWeights(key.ImageCountWeights)).
		SetQuota(key.Quota).
		SetQuotaUsed(key.QuotaUsed).
		SetNillableExpiresAt(key.ExpiresAt).
		SetNillableStartsAt(key.StartsAt).
		SetAccessTimePolicy(timeAccessPolicyToMap(key.AccessTimePolicy)).
		SetRateLimit5h(key.RateLimit5h).
		SetRateLimit1d(key.RateLimit1d).
		SetRateLimit7d(key.RateLimit7d)

	if len(key.IPWhitelist) > 0 {
		builder.SetIPWhitelist(key.IPWhitelist)
	}
	if len(key.IPBlacklist) > 0 {
		builder.SetIPBlacklist(key.IPBlacklist)
	}
	if len(key.QuotaUsedByCurrency) > 0 {
		builder.SetQuotaUsedByCurrency(service.CloneBillingCurrencyMap(key.QuotaUsedByCurrency))
	}

	created, err := builder.Save(ctx)
	if err == nil {
		key.ID = created.ID
		key.LastUsedAt = created.LastUsedAt
		key.CreatedAt = created.CreatedAt
		key.UpdatedAt = created.UpdatedAt
		if syncErr := r.syncAPIKeyGroupShadow(ctx, key); syncErr != nil {
			return syncErr
		}
	}
	return translatePersistenceError(err, nil, service.ErrAPIKeyExists)
}

func (r *apiKeyRepository) syncAPIKeyGroupShadow(ctx context.Context, key *service.APIKey) error {
	if key == nil || key.ID == 0 {
		return nil
	}
	if len(key.GroupBindings) > 0 {
		return r.SetAPIKeyGroups(ctx, key.ID, key.GroupBindings)
	}
	if key.GroupID != nil {
		return r.SetAPIKeyGroups(ctx, key.ID, []service.APIKeyGroupBinding{{
			APIKeyID: key.ID,
			GroupID:  *key.GroupID,
		}})
	}
	existingBindings, err := r.GetAPIKeyGroups(ctx, key.ID)
	if err != nil {
		return err
	}
	if len(existingBindings) == 0 {
		return nil
	}
	return r.SetAPIKeyGroups(ctx, key.ID, nil)
}

func (r *apiKeyRepository) BeginTx(ctx context.Context) (*dbent.Tx, error) {
	return clientFromContext(ctx, r.client).Tx(ctx)
}

func (r *apiKeyRepository) Update(ctx context.Context, key *service.APIKey) error {
	// 使用原子操作：将软删除检查与更新合并到同一语句，避免竞态条件。
	// 之前的实现先检查 Exist 再 UpdateOneID，若在两步之间发生软删除，
	// 则会更新已删除的记录。
	// 这里选择 Update().Where()，确保只有未软删除记录能被更新。
	// 同时显式设置 updated_at，避免二次查询带来的并发可见性问题。
	client := clientFromContext(ctx, r.client)
	now := time.Now()
	builder := client.APIKey.Update().
		Where(apikey.IDEQ(key.ID), apikey.DeletedAtIsNil()).
		SetName(key.Name).
		SetModelDisplayMode(service.NormalizeAPIKeyModelDisplayMode(key.ModelDisplayMode)).
		SetStatus(key.Status).
		SetImageOnlyEnabled(key.ImageOnlyEnabled).
		SetImageCountBillingEnabled(key.ImageCountBillingEnabled).
		SetImageMaxCount(key.ImageMaxCount).
		SetImageCountWeights(service.NormalizeAPIKeyImageCountWeights(key.ImageCountWeights)).
		SetQuota(key.Quota).
		SetQuotaUsed(key.QuotaUsed).
		SetAccessTimePolicy(timeAccessPolicyToMap(key.AccessTimePolicy)).
		SetRateLimit5h(key.RateLimit5h).
		SetRateLimit1d(key.RateLimit1d).
		SetRateLimit7d(key.RateLimit7d).
		SetUsage5h(key.Usage5h).
		SetUsage1d(key.Usage1d).
		SetUsage7d(key.Usage7d).
		SetUpdatedAt(now)
	if len(key.QuotaUsedByCurrency) > 0 {
		builder.SetQuotaUsedByCurrency(service.CloneBillingCurrencyMap(key.QuotaUsedByCurrency))
	}
	if len(key.Usage5hByCurrency) > 0 {
		builder.SetUsage5hByCurrency(service.CloneBillingCurrencyMap(key.Usage5hByCurrency))
	}
	if len(key.Usage1dByCurrency) > 0 {
		builder.SetUsage1dByCurrency(service.CloneBillingCurrencyMap(key.Usage1dByCurrency))
	}
	if len(key.Usage7dByCurrency) > 0 {
		builder.SetUsage7dByCurrency(service.CloneBillingCurrencyMap(key.Usage7dByCurrency))
	}
	if key.GroupID != nil {
		builder.SetGroupID(*key.GroupID)
	} else {
		builder.ClearGroupID()
	}

	// Expiration time
	if key.ExpiresAt != nil {
		builder.SetExpiresAt(*key.ExpiresAt)
	} else {
		builder.ClearExpiresAt()
	}
	if key.StartsAt != nil {
		builder.SetStartsAt(*key.StartsAt)
	} else {
		builder.ClearStartsAt()
	}

	// Rate limit window start times
	if key.Window5hStart != nil {
		builder.SetWindow5hStart(*key.Window5hStart)
	} else {
		builder.ClearWindow5hStart()
	}
	if key.Window1dStart != nil {
		builder.SetWindow1dStart(*key.Window1dStart)
	} else {
		builder.ClearWindow1dStart()
	}
	if key.Window7dStart != nil {
		builder.SetWindow7dStart(*key.Window7dStart)
	} else {
		builder.ClearWindow7dStart()
	}

	// IP 限制字段
	if len(key.IPWhitelist) > 0 {
		builder.SetIPWhitelist(key.IPWhitelist)
	} else {
		builder.ClearIPWhitelist()
	}
	if len(key.IPBlacklist) > 0 {
		builder.SetIPBlacklist(key.IPBlacklist)
	} else {
		builder.ClearIPBlacklist()
	}

	affected, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		// 更新影响行数为 0，说明记录不存在或已被软删除。
		return service.ErrAPIKeyNotFound
	}
	if err := r.syncAPIKeyGroupShadow(ctx, key); err != nil {
		return err
	}

	// 使用同一时间戳回填，避免并发删除导致二次查询失败。
	key.UpdatedAt = now
	return nil
}
