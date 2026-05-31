package service

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"
	"sync"
)

type APIKeyService struct {
	apiKeyRepo            APIKeyRepository
	userRepo              UserRepository
	groupRepo             GroupRepository
	userSubRepo           UserSubscriptionRepository
	userGroupRateRepo     UserGroupRateRepository
	modelCatalogService   *ModelCatalogService
	gatewayService        *GatewayService
	cache                 APIKeyCache
	rateLimitCacheInvalid RateLimitCacheInvalidator // optional: invalidate Redis rate limit cache
	billingCacheService   *BillingCacheService
	settingService        *SettingService
	cfg                   *config.Config
	authCacheL1           *ristretto.Cache
	authCfg               apiKeyAuthCacheConfig
	authGroup             singleflight.Group
	lastUsedTouchL1       sync.Map // keyID -> nextAllowedAt(time.Time)
	lastUsedTouchSF       singleflight.Group
}

// NewAPIKeyService 创建API Key服务实例

func NewAPIKeyService(
	apiKeyRepo APIKeyRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache APIKeyCache,
	cfg *config.Config,
) *APIKeyService {
	svc := &APIKeyService{
		apiKeyRepo:        apiKeyRepo,
		userRepo:          userRepo,
		groupRepo:         groupRepo,
		userSubRepo:       userSubRepo,
		userGroupRateRepo: userGroupRateRepo,
		cache:             cache,
		cfg:               cfg,
	}
	svc.initAuthCache(cfg)
	return svc
}

// SetRateLimitCacheInvalidator sets the optional rate limit cache invalidator.
// Called after construction (e.g. in wire) to avoid circular dependencies.

func (s *APIKeyService) SetRateLimitCacheInvalidator(inv RateLimitCacheInvalidator) {
	s.rateLimitCacheInvalid = inv
}

func (s *APIKeyService) SetBillingCacheService(billingCacheService *BillingCacheService) {
	s.billingCacheService = billingCacheService
}

func (s *APIKeyService) SetSettingService(settingService *SettingService) {
	s.settingService = settingService
}

func (s *APIKeyService) SetModelCatalogService(modelCatalogService *ModelCatalogService) {
	s.modelCatalogService = modelCatalogService
}

func (s *APIKeyService) SetGatewayService(gatewayService *GatewayService) {
	s.gatewayService = gatewayService
}

func (s *APIKeyService) compileAPIKeyIPRules(apiKey *APIKey) {
	if apiKey == nil {
		return
	}
	apiKey.CompiledIPWhitelist = ip.CompileIPRules(apiKey.IPWhitelist)
	apiKey.CompiledIPBlacklist = ip.CompileIPRules(apiKey.IPBlacklist)
}
