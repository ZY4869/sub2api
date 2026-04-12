package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"sync/atomic"
	"time"
)

const gatewayCompatibilityMetricsLogInterval = 1024

var gatewayCompatibilityMetricsLogCounter atomic.Uint64

type GatewayHandler struct {
	gatewayService            *service.GatewayService
	geminiNativeService       *service.GeminiNativeGatewayService
	geminiCompatService       *service.GeminiCompatGatewayService
	geminiLiveService         *service.GeminiLiveGatewayService
	geminiInteractionsService *service.GeminiInteractionsGatewayService
	antigravityGatewayService *service.AntigravityGatewayService
	userService               *service.UserService
	billingCacheService       *service.BillingCacheService
	usageService              *service.UsageService
	apiKeyService             *service.APIKeyService
	usageRecordWorkerPool     *service.UsageRecordWorkerPool
	errorPassthroughService   *service.ErrorPassthroughService
	concurrencyHelper         *ConcurrencyHelper
	userMsgQueueHelper        *UserMsgQueueHelper
	maxAccountSwitches        int
	maxAccountSwitchesGemini  int
	cfg                       *config.Config
	settingService            *service.SettingService
	modelRegistryService      *service.ModelRegistryService
}

func NewGatewayHandler(gatewayService *service.GatewayService, geminiNativeService *service.GeminiNativeGatewayService, geminiCompatService *service.GeminiCompatGatewayService, geminiLiveService *service.GeminiLiveGatewayService, geminiInteractionsService *service.GeminiInteractionsGatewayService, antigravityGatewayService *service.AntigravityGatewayService, userService *service.UserService, concurrencyService *service.ConcurrencyService, billingCacheService *service.BillingCacheService, usageService *service.UsageService, apiKeyService *service.APIKeyService, usageRecordWorkerPool *service.UsageRecordWorkerPool, errorPassthroughService *service.ErrorPassthroughService, userMsgQueueService *service.UserMessageQueueService, cfg *config.Config, settingService *service.SettingService) *GatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 10
	maxAccountSwitchesGemini := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
		if cfg.Gateway.MaxAccountSwitchesGemini > 0 {
			maxAccountSwitchesGemini = cfg.Gateway.MaxAccountSwitchesGemini
		}
	}
	var umqHelper *UserMsgQueueHelper
	if userMsgQueueService != nil && cfg != nil {
		umqHelper = NewUserMsgQueueHelper(userMsgQueueService, SSEPingFormatClaude, pingInterval)
	}
	return &GatewayHandler{gatewayService: gatewayService, geminiNativeService: geminiNativeService, geminiCompatService: geminiCompatService, geminiLiveService: geminiLiveService, geminiInteractionsService: geminiInteractionsService, antigravityGatewayService: antigravityGatewayService, userService: userService, billingCacheService: billingCacheService, usageService: usageService, apiKeyService: apiKeyService, usageRecordWorkerPool: usageRecordWorkerPool, errorPassthroughService: errorPassthroughService, concurrencyHelper: NewConcurrencyHelper(concurrencyService, SSEPingFormatClaude, pingInterval), userMsgQueueHelper: umqHelper, maxAccountSwitches: maxAccountSwitches, maxAccountSwitchesGemini: maxAccountSwitchesGemini, cfg: cfg, settingService: settingService}
}
func (h *GatewayHandler) SetModelRegistryService(modelRegistryService *service.ModelRegistryService) {
	h.modelRegistryService = modelRegistryService
}
func cloneAPIKeyWithGroup(apiKey *service.APIKey, group *service.Group) *service.APIKey {
	if apiKey == nil || group == nil {
		return apiKey
	}
	cloned := *apiKey
	groupID := group.ID
	cloned.GroupID = &groupID
	cloned.Group = group
	return &cloned
}
func (h *GatewayHandler) getUserMsgQueueMode(account *service.Account, parsed *service.ParsedRequest) string {
	if h.userMsgQueueHelper == nil {
		return ""
	}
	if !account.IsAnthropicOAuthOrSetupToken() {
		return ""
	}
	if !service.IsRealUserMessage(parsed) {
		return ""
	}
	mode := account.GetUserMsgQueueMode()
	if mode == "" {
		mode = h.cfg.Gateway.UserMessageQueue.GetEffectiveMode()
	}
	return mode
}
