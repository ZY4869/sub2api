package service

import (
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// RateLimitService 处理限流和过载状态管理
type RateLimitService struct {
	accountRepo           AccountRepository
	usageRepo             UsageLogRepository
	cfg                   *config.Config
	geminiQuotaService    *GeminiQuotaService
	tempUnschedCache      TempUnschedCache
	timeoutCounterCache   TimeoutCounterCache
	settingService        *SettingService
	tokenCacheInvalidator TokenCacheInvalidator
	usageCacheMu          sync.RWMutex
	usageCache            map[int64]*geminiUsageCacheEntry
}

// SuccessfulTestRecoveryResult 表示测试成功后恢复了哪些运行时状态。
type SuccessfulTestRecoveryResult struct {
	ClearedError     bool
	ClearedRateLimit bool
}

// AccountRecoveryOptions 控制账号恢复时的附加行为。
type AccountRecoveryOptions struct {
	InvalidateToken bool
}

// NewRateLimitService 创建RateLimitService实例
func NewRateLimitService(accountRepo AccountRepository, usageRepo UsageLogRepository, cfg *config.Config, geminiQuotaService *GeminiQuotaService, tempUnschedCache TempUnschedCache) *RateLimitService {
	return &RateLimitService{
		accountRepo:        accountRepo,
		usageRepo:          usageRepo,
		cfg:                cfg,
		geminiQuotaService: geminiQuotaService,
		tempUnschedCache:   tempUnschedCache,
		usageCache:         make(map[int64]*geminiUsageCacheEntry),
	}
}

// SetTimeoutCounterCache 设置超时计数器缓存（可选依赖）
func (s *RateLimitService) SetTimeoutCounterCache(cache TimeoutCounterCache) {
	s.timeoutCounterCache = cache
}

// SetSettingService 设置系统设置服务（可选依赖）
func (s *RateLimitService) SetSettingService(settingService *SettingService) {
	s.settingService = settingService
}

// SetTokenCacheInvalidator 设置 token 缓存清理器（可选依赖）
func (s *RateLimitService) SetTokenCacheInvalidator(invalidator TokenCacheInvalidator) {
	s.tokenCacheInvalidator = invalidator
}
