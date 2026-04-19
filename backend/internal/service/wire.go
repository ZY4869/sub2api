package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// BuildInfo contains build information
type BuildInfo struct {
	Version   string
	BuildType string
}

// ProvidePricingService creates and initializes PricingService
func ProvidePricingService(cfg *config.Config, remoteClient PricingRemoteClient) (*PricingService, error) {
	svc := NewPricingService(cfg, remoteClient)
	if err := svc.Initialize(); err != nil {
		// Pricing service initialization failure should not block startup, use fallback prices
		println("[Service] Warning: Pricing service initialization failed:", err.Error())
	}
	return svc, nil
}

// ProvideUpdateService creates UpdateService with BuildInfo
func ProvideUpdateService(cache UpdateCache, githubClient GitHubReleaseClient, buildInfo BuildInfo) *UpdateService {
	return NewUpdateService(cache, githubClient, buildInfo.Version, buildInfo.BuildType)
}

// ProvideEmailQueueService creates EmailQueueService with default worker count
func ProvideEmailQueueService(emailService *EmailService) *EmailQueueService {
	return NewEmailQueueService(emailService, 3)
}

// ProvideTokenRefreshService creates and starts TokenRefreshService
func ProvideTokenRefreshService(
	accountRepo AccountRepository,
	oauthService *OAuthService,
	kiroOAuthService *KiroOAuthService,
	openaiOAuthService *OpenAIOAuthService,
	geminiOAuthService *GeminiOAuthService,
	antigravityOAuthService *AntigravityOAuthService,
	cacheInvalidator TokenCacheInvalidator,
	schedulerCache SchedulerCache,
	cfg *config.Config,
	tempUnschedCache TempUnschedCache,
	privacyClientFactory PrivacyClientFactory,
	proxyRepo ProxyRepository,
	refreshAPI *OAuthRefreshAPI,
) *TokenRefreshService {
	svc := NewTokenRefreshService(accountRepo, oauthService, openaiOAuthService, geminiOAuthService, antigravityOAuthService, cacheInvalidator, schedulerCache, cfg, tempUnschedCache, kiroOAuthService)
	// 注入 OpenAI privacy opt-out 依赖
	svc.SetPrivacyDeps(privacyClientFactory, proxyRepo)
	// 注入统一 OAuth 刷新 API（消除 TokenRefreshService 与 TokenProvider 之间的竞争条件）
	svc.SetRefreshAPI(refreshAPI)
	// 调用侧显式注入后台刷新策略，避免策略漂移
	svc.SetRefreshPolicy(DefaultBackgroundRefreshPolicy())
	svc.Start()
	return svc
}

// ProvideClaudeTokenProvider creates ClaudeTokenProvider with OAuthRefreshAPI injection
func ProvideClaudeTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	oauthService *OAuthService,
	kiroOAuthService *KiroOAuthService,
	refreshAPI *OAuthRefreshAPI,
) *ClaudeTokenProvider {
	p := NewClaudeTokenProvider(accountRepo, tokenCache, oauthService)
	claudeExecutor := NewClaudeTokenRefresher(oauthService)
	if kiroOAuthService != nil {
		kiroExecutor := NewKiroTokenRefresher(kiroOAuthService)
		p.SetRefreshAPI(refreshAPI, claudeExecutor, kiroExecutor)
	} else {
		p.SetRefreshAPI(refreshAPI, claudeExecutor)
	}
	p.SetRefreshPolicy(ClaudeProviderRefreshPolicy())
	return p
}

// ProvideOpenAITokenProvider creates OpenAITokenProvider with OAuthRefreshAPI injection
func ProvideOpenAITokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	openaiOAuthService *OpenAIOAuthService,
	copilotOAuthService *CopilotOAuthService,
	refreshAPI *OAuthRefreshAPI,
) *OpenAITokenProvider {
	p := NewOpenAITokenProvider(accountRepo, tokenCache, openaiOAuthService)
	executor := NewOpenAITokenRefresher(openaiOAuthService, accountRepo)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetCopilotOAuthService(copilotOAuthService)
	p.SetRefreshPolicy(OpenAIProviderRefreshPolicy())
	return p
}

// ProvideGeminiTokenProvider creates GeminiTokenProvider with OAuthRefreshAPI injection
func ProvideGeminiTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	geminiOAuthService *GeminiOAuthService,
	refreshAPI *OAuthRefreshAPI,
) *GeminiTokenProvider {
	p := NewGeminiTokenProvider(accountRepo, tokenCache, geminiOAuthService)
	executor := NewGeminiTokenRefresher(geminiOAuthService)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(GeminiProviderRefreshPolicy())
	return p
}

// ProvideAntigravityTokenProvider creates AntigravityTokenProvider with OAuthRefreshAPI injection
func ProvideAntigravityTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	antigravityOAuthService *AntigravityOAuthService,
	refreshAPI *OAuthRefreshAPI,
	tempUnschedCache TempUnschedCache,
) *AntigravityTokenProvider {
	p := NewAntigravityTokenProvider(accountRepo, tokenCache, antigravityOAuthService)
	executor := NewAntigravityTokenRefresher(antigravityOAuthService)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(AntigravityProviderRefreshPolicy())
	p.SetTempUnschedCache(tempUnschedCache)
	return p
}

func ProvideOAuthRefreshAPI(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
) *OAuthRefreshAPI {
	return NewOAuthRefreshAPI(accountRepo, tokenCache)
}

func ProvideGeminiMessagesCompatService(
	accountRepo AccountRepository,
	apiKeyRepo APIKeyRepository,
	groupRepo GroupRepository,
	resourceBindingRepo UpstreamResourceBindingRepository,
	googleBatchQuotaReservationRepo GoogleBatchQuotaReservationRepository,
	googleBatchArchiveJobRepo GoogleBatchArchiveJobRepository,
	googleBatchArchiveObjectRepo GoogleBatchArchiveObjectRepository,
	cache GatewayCache,
	schedulerSnapshot *SchedulerSnapshotService,
	tokenProvider *GeminiTokenProvider,
	rateLimitService *RateLimitService,
	billingService *BillingService,
	usageLogRepo UsageLogRepository,
	usageBillingRepo UsageBillingRepository,
	settingService *SettingService,
	googleBatchArchiveStorage *GoogleBatchArchiveStorage,
	httpUpstream HTTPUpstream,
	antigravityGatewayService *AntigravityGatewayService,
	cfg *config.Config,
) *GeminiMessagesCompatService {
	svc := NewGeminiMessagesCompatService(accountRepo, groupRepo, cache, schedulerSnapshot, tokenProvider, rateLimitService, httpUpstream, antigravityGatewayService, cfg)
	svc.SetAPIKeyRepository(apiKeyRepo)
	svc.SetUpstreamResourceBindingRepository(resourceBindingRepo)
	svc.SetGoogleBatchQuotaReservationRepository(googleBatchQuotaReservationRepo)
	svc.SetGoogleBatchArchiveRepositories(googleBatchArchiveJobRepo, googleBatchArchiveObjectRepo)
	svc.SetBillingService(billingService)
	svc.SetUsageLogRepository(usageLogRepo)
	svc.SetUsageBillingRepository(usageBillingRepo)
	svc.SetSettingService(settingService)
	svc.SetGoogleBatchArchiveStorage(googleBatchArchiveStorage)
	return svc
}

func ProvideGeminiNativeGatewayService(compatService *GeminiMessagesCompatService) *GeminiNativeGatewayService {
	return NewGeminiNativeGatewayService(compatService)
}

func ProvideGeminiCompatGatewayService(compatService *GeminiMessagesCompatService) *GeminiCompatGatewayService {
	return NewGeminiCompatGatewayService(compatService)
}

func ProvideGeminiLiveGatewayService(compatService *GeminiMessagesCompatService) *GeminiLiveGatewayService {
	return NewGeminiLiveGatewayService(compatService)
}

func ProvideGeminiInteractionsGatewayService(compatService *GeminiMessagesCompatService) *GeminiInteractionsGatewayService {
	return NewGeminiInteractionsGatewayService(compatService)
}

// ProvideDashboardAggregationService 创建并启动仪表盘聚合服务
func ProvideDashboardAggregationService(repo DashboardAggregationRepository, timingWheel *TimingWheelService, cfg *config.Config) *DashboardAggregationService {
	svc := NewDashboardAggregationService(repo, timingWheel, cfg)
	svc.Start()
	return svc
}

// ProvideUsageCleanupService 创建并启动使用记录清理任务服务
func ProvideUsageCleanupService(repo UsageCleanupRepository, timingWheel *TimingWheelService, dashboardAgg *DashboardAggregationService, cfg *config.Config) *UsageCleanupService {
	svc := NewUsageCleanupService(repo, timingWheel, dashboardAgg, cfg)
	svc.Start()
	return svc
}

func ProvideUsageRepairService(repo UsageRepairRepository, timingWheel *TimingWheelService) *UsageRepairService {
	svc := NewUsageRepairService(repo, timingWheel)
	svc.Start()
	return svc
}

// ProvideAccountExpiryService creates and starts AccountExpiryService.
func ProvideAccountExpiryService(accountRepo AccountRepository) *AccountExpiryService {
	svc := NewAccountExpiryService(accountRepo, time.Minute)
	svc.Start()
	return svc
}

func ProvideAccountBlacklistCleanupService(accountRepo AccountRepository) *AccountBlacklistCleanupService {
	svc := NewAccountBlacklistCleanupService(accountRepo, time.Hour)
	svc.Start()
	return svc
}

func ProvideAccountRateLimitRecoveryProbeService(
	accountRepo AccountRepository,
	accountTestService *AccountTestService,
	rateLimitService *RateLimitService,
) *AccountRateLimitRecoveryProbeService {
	svc := NewAccountRateLimitRecoveryProbeService(accountRepo, accountTestService, rateLimitService, time.Minute)
	svc.Start()
	return svc
}

// ProvideSubscriptionExpiryService creates and starts SubscriptionExpiryService.
func ProvideSubscriptionExpiryService(userSubRepo UserSubscriptionRepository) *SubscriptionExpiryService {
	svc := NewSubscriptionExpiryService(userSubRepo, time.Minute)
	svc.Start()
	return svc
}

// ProvideTimingWheelService creates and starts TimingWheelService
func ProvideTimingWheelService() (*TimingWheelService, error) {
	svc, err := NewTimingWheelService()
	if err != nil {
		return nil, err
	}
	svc.Start()
	return svc, nil
}

// ProvideDeferredService creates and starts DeferredService
func ProvideDeferredService(accountRepo AccountRepository, timingWheel *TimingWheelService) *DeferredService {
	svc := NewDeferredService(accountRepo, timingWheel, 10*time.Second)
	svc.Start()
	return svc
}

// ProvideConcurrencyService creates ConcurrencyService and starts slot cleanup worker.
func ProvideConcurrencyService(cache ConcurrencyCache, accountRepo AccountRepository, cfg *config.Config) *ConcurrencyService {
	svc := NewConcurrencyService(cache)
	if err := svc.CleanupStaleProcessSlots(context.Background()); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: startup cleanup stale process slots failed: %v", err)
	}
	if cfg != nil {
		svc.StartSlotCleanupWorker(accountRepo, cfg.Gateway.Scheduling.SlotCleanupInterval)
	}
	return svc
}

// ProvideUserMessageQueueService 创建用户消息串行队列服务并启动清理 worker
func ProvideUserMessageQueueService(cache UserMsgQueueCache, rpmCache RPMCache, cfg *config.Config) *UserMessageQueueService {
	svc := NewUserMessageQueueService(cache, rpmCache, &cfg.Gateway.UserMessageQueue)
	if cfg.Gateway.UserMessageQueue.CleanupIntervalSeconds > 0 {
		svc.StartCleanupWorker(time.Duration(cfg.Gateway.UserMessageQueue.CleanupIntervalSeconds) * time.Second)
	}
	return svc
}

// ProvideSchedulerSnapshotService creates and starts SchedulerSnapshotService.
func ProvideSchedulerSnapshotService(
	cache SchedulerCache,
	outboxRepo SchedulerOutboxRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	cfg *config.Config,
) *SchedulerSnapshotService {
	svc := NewSchedulerSnapshotService(cache, outboxRepo, accountRepo, groupRepo, cfg)
	svc.Start()
	return svc
}

// ProvideRateLimitService creates RateLimitService with optional dependencies.
func ProvideRateLimitService(
	accountRepo AccountRepository,
	usageRepo UsageLogRepository,
	cfg *config.Config,
	geminiQuotaService *GeminiQuotaService,
	tempUnschedCache TempUnschedCache,
	timeoutCounterCache TimeoutCounterCache,
	settingService *SettingService,
	tokenCacheInvalidator TokenCacheInvalidator,
) *RateLimitService {
	svc := NewRateLimitService(accountRepo, usageRepo, cfg, geminiQuotaService, tempUnschedCache)
	svc.SetTimeoutCounterCache(timeoutCounterCache)
	svc.SetSettingService(settingService)
	svc.SetTokenCacheInvalidator(tokenCacheInvalidator)
	return svc
}

// ProvideOpsMetricsCollector creates and starts OpsMetricsCollector.
func ProvideOpsMetricsCollector(
	opsRepo OpsRepository,
	settingRepo SettingRepository,
	accountRepo AccountRepository,
	concurrencyService *ConcurrencyService,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsMetricsCollector {
	collector := NewOpsMetricsCollector(opsRepo, settingRepo, accountRepo, concurrencyService, db, redisClient, cfg)
	collector.Start()
	return collector
}

// ProvideOpsAggregationService creates and starts OpsAggregationService (hourly/daily pre-aggregation).
func ProvideOpsAggregationService(
	opsRepo OpsRepository,
	settingRepo SettingRepository,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsAggregationService {
	svc := NewOpsAggregationService(opsRepo, settingRepo, db, redisClient, cfg)
	svc.Start()
	return svc
}

// ProvideOpsAlertEvaluatorService creates and starts OpsAlertEvaluatorService.
func ProvideOpsAlertEvaluatorService(
	opsService *OpsService,
	opsRepo OpsRepository,
	emailService *EmailService,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsAlertEvaluatorService {
	svc := NewOpsAlertEvaluatorService(opsService, opsRepo, emailService, redisClient, cfg)
	svc.Start()
	return svc
}

// ProvideOpsCleanupService creates and starts OpsCleanupService (cron scheduled).
func ProvideOpsCleanupService(
	opsRepo OpsRepository,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsCleanupService {
	svc := NewOpsCleanupService(opsRepo, db, redisClient, cfg)
	svc.Start()
	return svc
}

func ProvideOpsSystemLogSink(opsRepo OpsRepository) *OpsSystemLogSink {
	sink := NewOpsSystemLogSink(opsRepo)
	sink.Start()
	logger.SetSink(sink)
	return sink
}

func ProvideGoogleBatchArchiveStorage() *GoogleBatchArchiveStorage {
	return NewGoogleBatchArchiveStorage()
}

func ProvideGoogleBatchArchivePollerService(
	jobRepo GoogleBatchArchiveJobRepository,
	compatService *GeminiMessagesCompatService,
	settingService *SettingService,
) *GoogleBatchArchivePollerService {
	svc := NewGoogleBatchArchivePollerService(jobRepo, compatService, settingService)
	svc.Start()
	return svc
}

func ProvideGoogleBatchArchivePrefetchService(
	jobRepo GoogleBatchArchiveJobRepository,
	objectRepo GoogleBatchArchiveObjectRepository,
	compatService *GeminiMessagesCompatService,
	settingService *SettingService,
) *GoogleBatchArchivePrefetchService {
	svc := NewGoogleBatchArchivePrefetchService(jobRepo, objectRepo, compatService, settingService)
	svc.Start()
	return svc
}

func ProvideGoogleBatchArchiveCleanupService(
	jobRepo GoogleBatchArchiveJobRepository,
	objectRepo GoogleBatchArchiveObjectRepository,
	compatService *GeminiMessagesCompatService,
	settingService *SettingService,
) *GoogleBatchArchiveCleanupService {
	svc := NewGoogleBatchArchiveCleanupService(jobRepo, objectRepo, compatService, settingService)
	svc.Start()
	return svc
}

func buildIdempotencyConfig(cfg *config.Config) IdempotencyConfig {
	idempotencyCfg := DefaultIdempotencyConfig()
	if cfg != nil {
		if cfg.Idempotency.DefaultTTLSeconds > 0 {
			idempotencyCfg.DefaultTTL = time.Duration(cfg.Idempotency.DefaultTTLSeconds) * time.Second
		}
		if cfg.Idempotency.SystemOperationTTLSeconds > 0 {
			idempotencyCfg.SystemOperationTTL = time.Duration(cfg.Idempotency.SystemOperationTTLSeconds) * time.Second
		}
		if cfg.Idempotency.ProcessingTimeoutSeconds > 0 {
			idempotencyCfg.ProcessingTimeout = time.Duration(cfg.Idempotency.ProcessingTimeoutSeconds) * time.Second
		}
		if cfg.Idempotency.FailedRetryBackoffSeconds > 0 {
			idempotencyCfg.FailedRetryBackoff = time.Duration(cfg.Idempotency.FailedRetryBackoffSeconds) * time.Second
		}
		if cfg.Idempotency.MaxStoredResponseLen > 0 {
			idempotencyCfg.MaxStoredResponseLen = cfg.Idempotency.MaxStoredResponseLen
		}
		idempotencyCfg.ObserveOnly = cfg.Idempotency.ObserveOnly
	}
	return idempotencyCfg
}

func ProvideIdempotencyCoordinator(repo IdempotencyRepository, cfg *config.Config) *IdempotencyCoordinator {
	coordinator := NewIdempotencyCoordinator(repo, buildIdempotencyConfig(cfg))
	SetDefaultIdempotencyCoordinator(coordinator)
	return coordinator
}

func ProvideSystemOperationLockService(repo IdempotencyRepository, cfg *config.Config) *SystemOperationLockService {
	return NewSystemOperationLockService(repo, buildIdempotencyConfig(cfg))
}

func ProvideIdempotencyCleanupService(repo IdempotencyRepository, cfg *config.Config) *IdempotencyCleanupService {
	svc := NewIdempotencyCleanupService(repo, cfg)
	svc.Start()
	return svc
}

// ProvideScheduledTestService creates ScheduledTestService.
func ProvideScheduledTestService(
	planRepo ScheduledTestPlanRepository,
	resultRepo ScheduledTestResultRepository,
) *ScheduledTestService {
	return NewScheduledTestService(planRepo, resultRepo)
}

// ProvideScheduledTestRunnerService creates and starts ScheduledTestRunnerService.
func ProvideScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	accountRepo AccountRepository,
	telegramNotifier *TelegramNotifierService,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	svc := NewScheduledTestRunnerService(planRepo, scheduledSvc, accountTestSvc, rateLimitSvc, accountRepo, telegramNotifier, cfg)
	svc.Start()
	return svc
}

// ProvideOpsScheduledReportService creates and starts OpsScheduledReportService.
func ProvideOpsScheduledReportService(
	opsService *OpsService,
	userService *UserService,
	emailService *EmailService,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsScheduledReportService {
	svc := NewOpsScheduledReportService(opsService, userService, emailService, redisClient, cfg)
	svc.Start()
	return svc
}

// ProvideAPIKeyAuthCacheInvalidator 提供 API Key 认证缓存失效能力
func ProvideAPIKeyAuthCacheInvalidator(apiKeyService *APIKeyService) APIKeyAuthCacheInvalidator {
	// Start Pub/Sub subscriber for L1 cache invalidation across instances
	apiKeyService.StartAuthCacheInvalidationSubscriber(context.Background())
	return apiKeyService
}

func ProvideAPIKeyService(
	apiKeyRepo APIKeyRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache APIKeyCache,
	modelCatalogService *ModelCatalogService,
	cfg *config.Config,
) *APIKeyService {
	svc := NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, userSubRepo, userGroupRateRepo, cache, cfg)
	svc.SetModelCatalogService(modelCatalogService)
	return svc
}

func ProvideModelCatalogService(
	settingRepo SettingRepository,
	adminService AdminService,
	billingService *BillingService,
	pricingService *PricingService,
	docsService *APIDocsService,
	modelRegistryService *ModelRegistryService,
	cfg *config.Config,
) *ModelCatalogService {
	if billingService != nil {
		billingService.SetModelRegistryService(modelRegistryService)
	}
	svc := NewModelCatalogService(settingRepo, adminService, billingService, pricingService, cfg)
	svc.SetModelRegistryService(modelRegistryService)
	svc.SetDocsService(docsService)
	return svc
}

func ProvideTLSFingerprintProfileService(
	repo TLSFingerprintProfileRepository,
	cache TLSFingerprintProfileCache,
) *TLSFingerprintProfileService {
	return NewTLSFingerprintProfileService(repo, cache)
}

func ProvideVertexUpstreamCatalogService(
	httpUpstream HTTPUpstream,
	geminiTokenProvider *GeminiTokenProvider,
	proxyRepo ProxyRepository,
	cfg *config.Config,
) *VertexUpstreamCatalogService {
	return NewVertexUpstreamCatalogService(httpUpstream, geminiTokenProvider, proxyRepo, cfg)
}

func ProvideAccountModelImportService(
	modelCatalogService *ModelCatalogService,
	modelRegistryService *ModelRegistryService,
	geminiCompatService *GeminiMessagesCompatService,
	vertexCatalogService *VertexUpstreamCatalogService,
	openAITokenProvider *OpenAITokenProvider,
	kiroRuntimeService *KiroRuntimeService,
	httpUpstream HTTPUpstream,
	proxyRepo ProxyRepository,
	tlsFingerprintProfileService *TLSFingerprintProfileService,
) *AccountModelImportService {
	svc := NewAccountModelImportService(modelCatalogService, geminiCompatService, httpUpstream, proxyRepo)
	svc.SetModelRegistryService(modelRegistryService)
	svc.SetOpenAITokenProvider(openAITokenProvider)
	svc.SetKiroRuntimeService(kiroRuntimeService)
	svc.SetVertexCatalogService(vertexCatalogService)
	svc.SetTLSFingerprintProfileService(tlsFingerprintProfileService)
	if geminiCompatService != nil {
		geminiCompatService.SetVertexCatalogService(vertexCatalogService)
	}
	return svc
}

func ProvideAccountModelDiagnosticsService(
	accountRepo AccountRepository,
	apiKeyRepo APIKeyRepository,
	groupRepo GroupRepository,
	accountModelImportService *AccountModelImportService,
) *AccountModelDiagnosticsService {
	return NewAccountModelDiagnosticsService(accountRepo, apiKeyRepo, groupRepo, accountModelImportService)
}

func ProvideAccountTestService(
	accountRepo AccountRepository,
	accountModelImportService *AccountModelImportService,
	claudeTokenProvider *ClaudeTokenProvider,
	openAITokenProvider *OpenAITokenProvider,
	geminiTokenProvider *GeminiTokenProvider,
	antigravityGatewayService *AntigravityGatewayService,
	gatewayService *GatewayService,
	grokGatewayService *GrokGatewayService,
	openAIGatewayService *OpenAIGatewayService,
	geminiCompatService *GeminiMessagesCompatService,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	tlsFingerprintProfileService *TLSFingerprintProfileService,
) *AccountTestService {
	svc := NewAccountTestService(accountRepo, accountModelImportService, geminiTokenProvider, antigravityGatewayService, httpUpstream, cfg)
	svc.SetClaudeTokenProvider(claudeTokenProvider)
	svc.SetOpenAITokenProvider(openAITokenProvider)
	if gatewayService != nil {
		gatewayService.SetAccountModelImportService(accountModelImportService)
	}
	svc.SetGatewayService(gatewayService)
	svc.SetGrokGatewayService(grokGatewayService)
	svc.SetOpenAIGatewayService(openAIGatewayService)
	svc.SetGeminiCompatService(geminiCompatService)
	svc.SetTLSFingerprintProfileService(tlsFingerprintProfileService)
	return svc
}

func ProvideGatewayService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	usageLogRepo UsageLogRepository,
	usageBillingRepo UsageBillingRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache GatewayCache,
	cfg *config.Config,
	schedulerSnapshot *SchedulerSnapshotService,
	concurrencyService *ConcurrencyService,
	billingService *BillingService,
	rateLimitService *RateLimitService,
	billingCacheService *BillingCacheService,
	identityService *IdentityService,
	httpUpstream HTTPUpstream,
	deferredService *DeferredService,
	claudeTokenProvider *ClaudeTokenProvider,
	sessionLimitCache SessionLimitCache,
	rpmCache RPMCache,
	digestStore *DigestSessionStore,
	settingService *SettingService,
	modelCatalogService *ModelCatalogService,
	apiKeyService *APIKeyService,
	channelService *ChannelService,
	vertexCatalogService *VertexUpstreamCatalogService,
	tlsFingerprintProfileService *TLSFingerprintProfileService,
) *GatewayService {
	svc := NewGatewayService(accountRepo, groupRepo, usageLogRepo, usageBillingRepo, userRepo, userSubRepo, userGroupRateRepo, cache, cfg, schedulerSnapshot, concurrencyService, billingService, rateLimitService, billingCacheService, identityService, httpUpstream, deferredService, claudeTokenProvider, sessionLimitCache, rpmCache, digestStore, settingService)
	svc.SetChannelService(channelService)
	svc.SetVertexCatalogService(vertexCatalogService)
	svc.SetTLSFingerprintProfileService(tlsFingerprintProfileService)
	if modelCatalogService != nil {
		modelCatalogService.SetGatewayService(svc)
	}
	if apiKeyService != nil {
		apiKeyService.SetGatewayService(svc)
	}
	return svc
}

func ProvideOpenAIGatewayService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	usageBillingRepo UsageBillingRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache GatewayCache,
	cfg *config.Config,
	schedulerSnapshot *SchedulerSnapshotService,
	concurrencyService *ConcurrencyService,
	billingService *BillingService,
	rateLimitService *RateLimitService,
	billingCacheService *BillingCacheService,
	httpUpstream HTTPUpstream,
	deferredService *DeferredService,
	openAITokenProvider *OpenAITokenProvider,
	channelService *ChannelService,
) *OpenAIGatewayService {
	svc := NewOpenAIGatewayService(accountRepo, usageLogRepo, usageBillingRepo, userRepo, userSubRepo, userGroupRateRepo, cache, cfg, schedulerSnapshot, concurrencyService, billingService, rateLimitService, billingCacheService, httpUpstream, deferredService, openAITokenProvider)
	svc.SetChannelService(channelService)
	return svc
}

func ProvideAccountUsageService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	usageFetcher ClaudeUsageFetcher,
	geminiQuotaService *GeminiQuotaService,
	antigravityQuotaFetcher *AntigravityQuotaFetcher,
	cache *UsageCache,
	identityCache IdentityCache,
	tlsFingerprintProfileService *TLSFingerprintProfileService,
) *AccountUsageService {
	svc := NewAccountUsageService(accountRepo, usageLogRepo, usageFetcher, geminiQuotaService, antigravityQuotaFetcher, cache, identityCache)
	svc.SetTLSFingerprintProfileService(tlsFingerprintProfileService)
	return svc
}

func ProvideModelRegistryService(settingRepo SettingRepository, accountRepo AccountRepository) *ModelRegistryService {
	svc := NewModelRegistryService(settingRepo)
	svc.SetAccountRepository(accountRepo)
	return svc
}

// ProvideBackupService creates and starts BackupService
func ProvideBackupService(
	settingRepo SettingRepository,
	cfg *config.Config,
	encryptor SecretEncryptor,
	storeFactory BackupObjectStoreFactory,
	dumper DBDumper,
) *BackupService {
	svc := NewBackupService(settingRepo, cfg, encryptor, storeFactory, dumper)
	svc.Start()
	return svc
}

// ProvideSettingService wires SettingService with group reader for default subscription validation.
func ProvideSettingService(settingRepo SettingRepository, groupRepo GroupRepository, cfg *config.Config) *SettingService {
	svc := NewSettingService(settingRepo, cfg)
	svc.SetDefaultSubscriptionGroupReader(groupRepo)
	return svc
}

// ProviderSet is the Wire provider set for all services
var ProviderSet = wire.NewSet(
	// Core services
	NewAuthService,
	NewUserService,
	ProvideAPIKeyService,
	ProvideAPIKeyAuthCacheInvalidator,
	NewGroupService,
	NewChannelService,
	NewAccountService,
	NewProxyService,
	NewRedeemService,
	NewPromoService,
	NewUsageService,
	NewDashboardService,
	ProvidePricingService,
	NewBillingService,
	NewBillingCacheService,
	NewAnnouncementService,
	NewAPIDocsService,
	NewAdminService,
	ProvideDocumentAIService,
	ProvideModelRegistryService,
	ProvideModelCatalogService,
	ProvideTLSFingerprintProfileService,
	ProvideVertexUpstreamCatalogService,
	ProvideAccountModelImportService,
	ProvideAccountModelDiagnosticsService,
	ProvideGatewayService,
	ProvideGoogleBatchArchiveStorage,
	ProvideGoogleBatchArchivePollerService,
	ProvideGoogleBatchArchivePrefetchService,
	ProvideGoogleBatchArchiveCleanupService,
	ProvideOpenAIGatewayService,
	NewGrokGatewayService,
	NewGrokReverseClient,
	NewOAuthService,
	NewOpenAIOAuthService,
	NewCopilotOAuthService,
	NewKiroOAuthService,
	NewGeminiOAuthService,
	NewGeminiQuotaService,
	NewCompositeTokenCacheInvalidator,
	wire.Bind(new(TokenCacheInvalidator), new(*CompositeTokenCacheInvalidator)),
	NewAntigravityOAuthService,
	ProvideOAuthRefreshAPI,
	ProvideGeminiTokenProvider,
	ProvideGeminiMessagesCompatService,
	ProvideGeminiNativeGatewayService,
	ProvideGeminiCompatGatewayService,
	ProvideGeminiLiveGatewayService,
	ProvideGeminiInteractionsGatewayService,
	ProvideAntigravityTokenProvider,
	ProvideOpenAITokenProvider,
	ProvideClaudeTokenProvider,
	NewAntigravityGatewayService,
	ProvideRateLimitService,
	ProvideAccountUsageService,
	ProvideAccountTestService,
	ProvideSettingService,
	NewDataManagementService,
	ProvideBackupService,
	ProvideOpsSystemLogSink,
	NewOpsService,
	ProvideOpsMetricsCollector,
	ProvideOpsAggregationService,
	ProvideOpsAlertEvaluatorService,
	ProvideOpsCleanupService,
	ProvideOpsScheduledReportService,
	NewEmailService,
	ProvideEmailQueueService,
	NewTurnstileService,
	NewSubscriptionService,
	wire.Bind(new(DefaultSubscriptionAssigner), new(*SubscriptionService)),
	ProvideConcurrencyService,
	ProvideUserMessageQueueService,
	NewUsageRecordWorkerPool,
	ProvideSchedulerSnapshotService,
	NewIdentityService,
	NewCRSSyncService,
	ProvideUpdateService,
	ProvideTokenRefreshService,
	ProvideAccountExpiryService,
	ProvideAccountBlacklistCleanupService,
	ProvideAccountRateLimitRecoveryProbeService,
	ProvideSubscriptionExpiryService,
	ProvideTimingWheelService,
	ProvideDashboardAggregationService,
	ProvideUsageCleanupService,
	ProvideUsageRepairService,
	ProvideDeferredService,
	NewAntigravityQuotaFetcher,
	NewUserAttributeService,
	NewUsageCache,
	NewKiroRuntimeService,
	NewTotpService,
	NewErrorPassthroughService,
	NewDigestSessionStore,
	ProvideIdempotencyCoordinator,
	ProvideSystemOperationLockService,
	ProvideIdempotencyCleanupService,
	ProvideScheduledTestService,
	NewTelegramNotifierService,
	ProvideScheduledTestRunnerService,
	NewGroupCapacityService,
)
