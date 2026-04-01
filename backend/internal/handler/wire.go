package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
)

// ProvideAdminHandlers creates the AdminHandlers struct
func ProvideAdminHandlers(
	dashboardHandler *admin.DashboardHandler,
	userHandler *admin.UserHandler,
	groupHandler *admin.GroupHandler,
	accountHandler *admin.AccountHandler,
	announcementHandler *admin.AnnouncementHandler,
	dataManagementHandler *admin.DataManagementHandler,
	backupHandler *admin.BackupHandler,
	oauthHandler *admin.OAuthHandler,
	openaiOAuthHandler *admin.OpenAIOAuthHandler,
	kiroOAuthHandler *admin.KiroOAuthHandler,
	geminiOAuthHandler *admin.GeminiOAuthHandler,
	antigravityOAuthHandler *admin.AntigravityOAuthHandler,
	proxyHandler *admin.ProxyHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	opsHandler *admin.OpsHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	usageHandler *admin.UsageHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	errorPassthroughHandler *admin.ErrorPassthroughHandler,
	apiKeyHandler *admin.AdminAPIKeyHandler,
	modelCatalogHandler *admin.ModelCatalogHandler,
	modelRegistryHandler *admin.ModelRegistryHandler,
	scheduledTestHandler *admin.ScheduledTestHandler,
	tlsFingerprintProfileHandler *admin.TLSFingerprintProfileHandler,
) *AdminHandlers {
	return &AdminHandlers{
		Dashboard:             dashboardHandler,
		User:                  userHandler,
		Group:                 groupHandler,
		Account:               accountHandler,
		Announcement:          announcementHandler,
		DataManagement:        dataManagementHandler,
		Backup:                backupHandler,
		OAuth:                 oauthHandler,
		OpenAIOAuth:           openaiOAuthHandler,
		KiroOAuth:             kiroOAuthHandler,
		GeminiOAuth:           geminiOAuthHandler,
		AntigravityOAuth:      antigravityOAuthHandler,
		Proxy:                 proxyHandler,
		Redeem:                redeemHandler,
		Promo:                 promoHandler,
		Setting:               settingHandler,
		Ops:                   opsHandler,
		System:                systemHandler,
		Subscription:          subscriptionHandler,
		Usage:                 usageHandler,
		UserAttribute:         userAttributeHandler,
		ErrorPassthrough:      errorPassthroughHandler,
		APIKey:                apiKeyHandler,
		ModelCatalog:          modelCatalogHandler,
		ModelRegistry:         modelRegistryHandler,
		ScheduledTest:         scheduledTestHandler,
		TLSFingerprintProfile: tlsFingerprintProfileHandler,
	}
}

// ProvideAdminAccountHandler creates AccountHandler and wires optional import dependencies.
func ProvideAdminAccountHandler(
	adminService service.AdminService,
	oauthService *service.OAuthService,
	openaiOAuthService *service.OpenAIOAuthService,
	copilotOAuthService *service.CopilotOAuthService,
	kiroOAuthService *service.KiroOAuthService,
	geminiOAuthService *service.GeminiOAuthService,
	antigravityOAuthService *service.AntigravityOAuthService,
	rateLimitService *service.RateLimitService,
	accountUsageService *service.AccountUsageService,
	accountTestService *service.AccountTestService,
	concurrencyService *service.ConcurrencyService,
	crsSyncService *service.CRSSyncService,
	sessionLimitCache service.SessionLimitCache,
	rpmCache service.RPMCache,
	tokenCacheInvalidator service.TokenCacheInvalidator,
	accountModelImportService *service.AccountModelImportService,
	accountModelDiagnosticsService *service.AccountModelDiagnosticsService,
	modelRegistryService *service.ModelRegistryService,
) *admin.AccountHandler {
	handler := admin.NewAccountHandler(
		adminService,
		oauthService,
		openaiOAuthService,
		geminiOAuthService,
		antigravityOAuthService,
		rateLimitService,
		accountUsageService,
		accountTestService,
		concurrencyService,
		crsSyncService,
		sessionLimitCache,
		rpmCache,
		tokenCacheInvalidator,
	)
	handler.SetAccountModelImportService(accountModelImportService)
	handler.SetAccountModelDiagnosticsService(accountModelDiagnosticsService)
	handler.SetModelRegistryService(modelRegistryService)
	handler.SetCopilotOAuthService(copilotOAuthService)
	handler.SetKiroOAuthService(kiroOAuthService)
	return handler
}

func ProvideOpenAIOAuthHandler(
	openaiOAuthService *service.OpenAIOAuthService,
	copilotOAuthService *service.CopilotOAuthService,
	adminService service.AdminService,
) *admin.OpenAIOAuthHandler {
	handler := admin.NewOpenAIOAuthHandler(openaiOAuthService, adminService)
	handler.SetCopilotOAuthService(copilotOAuthService)
	return handler
}

func ProvideMetaHandler(modelCatalogService *service.ModelCatalogService, modelRegistryService *service.ModelRegistryService) *MetaHandler {
	handler := NewMetaHandler(modelCatalogService)
	handler.SetModelRegistryService(modelRegistryService)
	return handler
}

func ProvideGatewayHandler(
	gatewayService *service.GatewayService,
	geminiCompatService *service.GeminiMessagesCompatService,
	antigravityGatewayService *service.AntigravityGatewayService,
	userService *service.UserService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	userMsgQueueService *service.UserMessageQueueService,
	cfg *config.Config,
	settingService *service.SettingService,
	modelRegistryService *service.ModelRegistryService,
) *GatewayHandler {
	gatewayService.SetModelRegistryService(modelRegistryService)
	handler := NewGatewayHandler(gatewayService, geminiCompatService, antigravityGatewayService, userService, concurrencyService, billingCacheService, usageService, apiKeyService, usageRecordWorkerPool, errorPassthroughService, userMsgQueueService, cfg, settingService)
	handler.SetModelRegistryService(modelRegistryService)
	return handler
}

func ProvideGrokGatewayHandler(
	gatewayService *service.GatewayService,
	grokGatewayService *service.GrokGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	cfg *config.Config,
	settingService *service.SettingService,
) *GrokGatewayHandler {
	handler := NewGrokGatewayHandler(gatewayService, grokGatewayService, concurrencyService, billingCacheService, apiKeyService, usageRecordWorkerPool, cfg)
	handler.SetSettingService(settingService)
	return handler
}

// ProvideSystemHandler creates admin.SystemHandler with UpdateService
func ProvideSystemHandler(updateService *service.UpdateService, lockService *service.SystemOperationLockService) *admin.SystemHandler {
	return admin.NewSystemHandler(updateService, lockService)
}

// ProvideSettingHandler creates SettingHandler with version from BuildInfo
func ProvideSettingHandler(settingService *service.SettingService, buildInfo BuildInfo) *SettingHandler {
	return NewSettingHandler(settingService, buildInfo.Version)
}

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	metaHandler *MetaHandler,
	apiKeyHandler *APIKeyHandler,
	usageHandler *UsageHandler,
	redeemHandler *RedeemHandler,
	subscriptionHandler *SubscriptionHandler,
	announcementHandler *AnnouncementHandler,
	adminHandlers *AdminHandlers,
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	grokGatewayHandler *GrokGatewayHandler,
	soraGatewayHandler *SoraGatewayHandler,
	soraClientHandler *SoraClientHandler,
	settingHandler *SettingHandler,
	totpHandler *TotpHandler,
	_ *service.IdempotencyCoordinator,
	_ *service.IdempotencyCleanupService,
) *Handlers {
	return &Handlers{
		Auth:          authHandler,
		User:          userHandler,
		Meta:          metaHandler,
		APIKey:        apiKeyHandler,
		Usage:         usageHandler,
		Redeem:        redeemHandler,
		Subscription:  subscriptionHandler,
		Announcement:  announcementHandler,
		Admin:         adminHandlers,
		Gateway:       gatewayHandler,
		OpenAIGateway: openaiGatewayHandler,
		GrokGateway:   grokGatewayHandler,
		SoraGateway:   soraGatewayHandler,
		SoraClient:    soraClientHandler,
		Setting:       settingHandler,
		Totp:          totpHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Top-level handlers
	NewAuthHandler,
	NewUserHandler,
	ProvideMetaHandler,
	NewAPIKeyHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	ProvideGatewayHandler,
	NewOpenAIGatewayHandler,
	ProvideGrokGatewayHandler,
	NewSoraGatewayHandler,
	NewSoraClientHandler,
	NewTotpHandler,
	ProvideSettingHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	admin.NewUserHandler,
	admin.NewGroupHandler,
	ProvideAdminAccountHandler,
	admin.NewAnnouncementHandler,
	admin.NewDataManagementHandler,
	admin.NewBackupHandler,
	admin.NewOAuthHandler,
	ProvideOpenAIOAuthHandler,
	admin.NewKiroOAuthHandler,
	admin.NewGeminiOAuthHandler,
	admin.NewAntigravityOAuthHandler,
	admin.NewProxyHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	admin.NewSettingHandler,
	admin.NewOpsHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUsageHandler,
	admin.NewUserAttributeHandler,
	admin.NewErrorPassthroughHandler,
	admin.NewAdminAPIKeyHandler,
	admin.NewModelCatalogHandler,
	admin.NewModelRegistryHandler,
	admin.NewScheduledTestHandler,
	admin.NewTLSFingerprintProfileHandler,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
