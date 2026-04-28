package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// OpenAIGatewayHandler handles OpenAI API gateway requests.
type OpenAIGatewayHandler struct {
	gatewayService          *service.OpenAIGatewayService
	billingCacheService     *service.BillingCacheService
	apiKeyService           *service.APIKeyService
	usageRecordWorkerPool   *service.UsageRecordWorkerPool
	errorPassthroughService *service.ErrorPassthroughService
	concurrencyHelper       *ConcurrencyHelper
	maxAccountSwitches      int
	cfg                     *config.Config
	settingService          *service.SettingService
}

// NewOpenAIGatewayHandler creates a new OpenAIGatewayHandler.
func NewOpenAIGatewayHandler(
	gatewayService *service.OpenAIGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	cfg *config.Config,
) *OpenAIGatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
	}
	return &OpenAIGatewayHandler{
		gatewayService:          gatewayService,
		billingCacheService:     billingCacheService,
		apiKeyService:           apiKeyService,
		usageRecordWorkerPool:   usageRecordWorkerPool,
		errorPassthroughService: errorPassthroughService,
		concurrencyHelper:       NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		maxAccountSwitches:      maxAccountSwitches,
		cfg:                     cfg,
	}
}

func (h *OpenAIGatewayHandler) SetSettingService(settingService *service.SettingService) {
	h.settingService = settingService
}

func normalizeOpenAIGroupPlatform(platform string) string {
	switch platform {
	case service.PlatformCopilot:
		return service.PlatformCopilot
	case service.PlatformDeepSeek:
		return service.PlatformDeepSeek
	default:
		return service.PlatformOpenAI
	}
}

func applyOpenAIPlatformContext(c *gin.Context, groupPlatform string) {
	if c == nil || c.Request == nil {
		return
	}
	c.Request = c.Request.WithContext(service.WithOpenAIPlatform(c.Request.Context(), normalizeOpenAIGroupPlatform(groupPlatform)))
}
