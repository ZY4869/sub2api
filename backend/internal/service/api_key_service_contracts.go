package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByID(ctx context.Context, id int64) (*APIKey, error)
	// GetKeyAndOwnerID 仅获取 API Key 的 key 与所有者 ID，用于删除等轻量场景
	GetKeyAndOwnerID(ctx context.Context, id int64) (string, int64, error)
	GetByKey(ctx context.Context, key string) (*APIKey, error)
	// GetByKeyForAuth 认证专用查询，返回最小字段集
	GetByKeyForAuth(ctx context.Context, key string) (*APIKey, error)
	Update(ctx context.Context, key *APIKey) error
	Delete(ctx context.Context, id int64) error

	ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error)
	VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)
	ExistsByKey(ctx context.Context, key string) (bool, error)
	ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error)
	SearchAPIKeys(ctx context.Context, userID int64, keyword string, limit int) ([]APIKey, error)
	ClearGroupIDByGroupID(ctx context.Context, groupID int64) (int64, error)
	// UpdateGroupIDByUserAndGroup 将用户下绑定 oldGroupID 的所有 Key 迁移到 newGroupID
	UpdateGroupIDByUserAndGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (int64, error)
	CountByGroupID(ctx context.Context, groupID int64) (int64, error)
	ListKeysByUserID(ctx context.Context, userID int64) ([]string, error)
	ListKeysByGroupID(ctx context.Context, groupID int64) ([]string, error)
	GetAPIKeyGroups(ctx context.Context, keyID int64) ([]APIKeyGroupBinding, error)
	SetAPIKeyGroups(ctx context.Context, keyID int64, bindings []APIKeyGroupBinding) error
	IncrementAPIKeyGroupQuotaUsed(ctx context.Context, keyID, groupID int64, amount float64) error

	IncrementQuotaUsed(ctx context.Context, id int64, amount float64) (float64, error)
	UpdateLastUsed(ctx context.Context, id int64, usedAt time.Time) error

	TryReserveImageCount(ctx context.Context, id int64, count int) (bool, error)
	RollbackImageCount(ctx context.Context, id int64, count int) error

	IncrementRateLimitUsage(ctx context.Context, id int64, cost float64) error
	ResetRateLimitWindows(ctx context.Context, id int64) error
	GetRateLimitData(ctx context.Context, id int64) (*APIKeyRateLimitData, error)
}

type billingHoldRepositoryProvider interface {
	BillingHoldRepository() BillingHoldRepository
}

type apiKeyDeletedReader interface {
	GetByIDAllowDeleted(ctx context.Context, id int64) (*APIKey, error)
}

type APIKeyCache interface {
	GetCreateAttemptCount(ctx context.Context, userID int64) (int, error)
	IncrementCreateAttemptCount(ctx context.Context, userID int64) error
	DeleteCreateAttemptCount(ctx context.Context, userID int64) error

	IncrementDailyUsage(ctx context.Context, apiKey string) error
	SetDailyUsageExpiry(ctx context.Context, apiKey string, ttl time.Duration) error

	GetAuthCache(ctx context.Context, key string) (*APIKeyAuthCacheEntry, error)
	SetAuthCache(ctx context.Context, key string, entry *APIKeyAuthCacheEntry, ttl time.Duration) error
	DeleteAuthCache(ctx context.Context, key string) error

	PublishAuthCacheInvalidation(ctx context.Context, cacheKey string) error
	SubscribeAuthCacheInvalidation(ctx context.Context, handler func(cacheKey string)) error
}

type APIKeyAuthCacheInvalidator interface {
	InvalidateAuthCacheByKey(ctx context.Context, key string)
	InvalidateAuthCacheByUserID(ctx context.Context, userID int64)
	InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64)
}

// RateLimitCacheInvalidator invalidates rate limit cache entries on manual reset.
type RateLimitCacheInvalidator interface {
	InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error
}
