package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	chatgptCodexURL                       = "https://chatgpt.com/backend-api/codex/responses"
	openaiPlatformAPIURL                  = "https://api.openai.com/v1/responses"
	openaiStickySessionTTL                = time.Hour
	codexCLIUserAgent                     = "codex_cli_rs/0.104.0"
	codexCLIOnlyHeaderValueMaxBytes       = 256
	OpenAIParsedRequestBodyKey            = "openai_parsed_request_body"
	openAIWSReconnectRetryLimit           = 5
	openAIWSRetryBackoffInitialDefault    = 120 * time.Millisecond
	openAIWSRetryBackoffMaxDefault        = 2 * time.Second
	openAIWSRetryJitterRatioDefault       = 0.2
	openAICompactSessionSeedKey           = "openai_compact_session_seed"
	codexCLIVersion                       = "0.104.0"
	openAICodexSnapshotPersistMinInterval = 30 * time.Second
)

var openaiAllowedHeaders = map[string]bool{
	"accept-language":       true,
	"content-type":          true,
	"conversation_id":       true,
	"user-agent":            true,
	"originator":            true,
	"session_id":            true,
	"x-codex-turn-state":    true,
	"x-codex-turn-metadata": true,
}

var openaiPassthroughAllowedHeaders = map[string]bool{
	"accept":                true,
	"accept-language":       true,
	"content-type":          true,
	"conversation_id":       true,
	"openai-beta":           true,
	"user-agent":            true,
	"originator":            true,
	"session_id":            true,
	"x-codex-turn-state":    true,
	"x-codex-turn-metadata": true,
}

var codexCLIOnlyDebugHeaderWhitelist = []string{
	"User-Agent",
	"Content-Type",
	"Accept",
	"Accept-Language",
	"OpenAI-Beta",
	"Originator",
	"Session_ID",
	"Conversation_ID",
	"X-Request-ID",
	"X-Client-Request-ID",
	"X-Forwarded-For",
	"X-Real-IP",
}

type OpenAIGatewayService struct {
	accountRepo                   AccountRepository
	usageLogRepo                  UsageLogRepository
	usageBillingRepo              UsageBillingRepository
	userRepo                      UserRepository
	userSubRepo                   UserSubscriptionRepository
	cache                         GatewayCache
	cfg                           *config.Config
	settingService                *SettingService
	modelRegistryService          *ModelRegistryService
	codexDetector                 CodexClientRestrictionDetector
	schedulerSnapshot             *SchedulerSnapshotService
	concurrencyService            *ConcurrencyService
	billingService                *BillingService
	rateLimitService              *RateLimitService
	billingCacheService           *BillingCacheService
	userGroupRateResolver         *userGroupRateResolver
	channelService                *ChannelService
	httpUpstream                  HTTPUpstream
	deferredService               *DeferredService
	openAITokenProvider           *OpenAITokenProvider
	toolCorrector                 *CodexToolCorrector
	openaiWSResolver              OpenAIWSProtocolResolver
	openaiWSPoolOnce              sync.Once
	openaiWSStateStoreOnce        sync.Once
	openaiSchedulerOnce           sync.Once
	openaiWSPassthroughDialerOnce sync.Once
	openaiWSPool                  *openAIWSConnPool
	openaiWSStateStore            OpenAIWSStateStore
	openaiScheduler               OpenAIAccountScheduler
	openaiWSPassthroughDialer     openAIWSClientDialer
	openaiAccountStats            *openAIAccountRuntimeStats
	openaiWSFallbackUntil         sync.Map
	openaiWSRetryMetrics          openAIWSRetryMetrics
	responseHeaderFilter          *responseheaders.CompiledHeaderFilter
	codexSnapshotThrottle         *accountWriteThrottle
}

func NewOpenAIGatewayService(accountRepo AccountRepository, usageLogRepo UsageLogRepository, usageBillingRepo UsageBillingRepository, userRepo UserRepository, userSubRepo UserSubscriptionRepository, userGroupRateRepo UserGroupRateRepository, cache GatewayCache, cfg *config.Config, schedulerSnapshot *SchedulerSnapshotService, concurrencyService *ConcurrencyService, billingService *BillingService, rateLimitService *RateLimitService, billingCacheService *BillingCacheService, httpUpstream HTTPUpstream, deferredService *DeferredService, openAITokenProvider *OpenAITokenProvider, settingService *SettingService) *OpenAIGatewayService {
	svc := &OpenAIGatewayService{
		accountRepo:           accountRepo,
		usageLogRepo:          usageLogRepo,
		usageBillingRepo:      usageBillingRepo,
		userRepo:              userRepo,
		userSubRepo:           userSubRepo,
		cache:                 cache,
		cfg:                   cfg,
		settingService:        settingService,
		codexDetector:         NewOpenAICodexClientRestrictionDetector(cfg),
		schedulerSnapshot:     schedulerSnapshot,
		concurrencyService:    concurrencyService,
		billingService:        billingService,
		rateLimitService:      rateLimitService,
		billingCacheService:   billingCacheService,
		userGroupRateResolver: newUserGroupRateResolver(userGroupRateRepo, nil, resolveUserGroupRateCacheTTL(cfg), nil, "service.openai_gateway"),
		httpUpstream:          httpUpstream,
		deferredService:       deferredService,
		openAITokenProvider:   openAITokenProvider,
		toolCorrector:         NewCodexToolCorrector(),
		openaiWSResolver:      NewOpenAIWSProtocolResolver(cfg),
		responseHeaderFilter:  compileResponseHeaderFilter(cfg),
		codexSnapshotThrottle: newAccountWriteThrottle(openAICodexSnapshotPersistMinInterval),
	}
	svc.logOpenAIWSModeBootstrap()
	return svc
}

func (s *OpenAIGatewayService) getCodexSnapshotThrottle() *accountWriteThrottle {
	if s != nil && s.codexSnapshotThrottle != nil {
		return s.codexSnapshotThrottle
	}
	return defaultOpenAICodexSnapshotPersistThrottle
}

func (s *OpenAIGatewayService) billingDeps() *billingDeps {
	return &billingDeps{
		accountRepo:         s.accountRepo,
		userRepo:            s.userRepo,
		userSubRepo:         s.userSubRepo,
		billingCacheService: s.billingCacheService,
		deferredService:     s.deferredService,
	}
}

func (s *OpenAIGatewayService) SetChannelService(channelService *ChannelService) {
	s.channelService = channelService
}

func (s *OpenAIGatewayService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
}

func (s *OpenAIGatewayService) CloseOpenAIWSPool() {
	if s != nil && s.openaiWSPool != nil {
		s.openaiWSPool.Close()
	}
}

func (s *OpenAIGatewayService) shouldFailoverUpstreamError(statusCode int) bool {
	switch statusCode {
	case 401, 402, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}

func (s *OpenAIGatewayService) shouldFailoverOpenAIUpstreamResponse(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if s.shouldFailoverUpstreamError(statusCode) {
		return true
	}
	return isOpenAITransientProcessingError(statusCode, upstreamMsg, upstreamBody)
}

func (s *OpenAIGatewayService) handleFailoverSideEffects(ctx context.Context, resp *http.Response, account *Account) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
}

func (s *OpenAIGatewayService) replaceModelInResponseBody(body []byte, fromModel, toModel string) []byte {
	m := gjson.GetBytes(body, "model")
	if !m.Exists() {
		return body
	}

	shouldReplace := m.Str == fromModel
	if !shouldReplace {
		// Compat 图片 host 模型可能返回带版本后缀的 model，例如：
		// "gpt-5.4-mini-2026-03-17"。对 `gpt-image-2` 简写路由场景，
		// 需要把对外的 model 固定回 `gpt-image-2`。
		shouldReplace = fromModel == OpenAICompatImageHostModel &&
			toModel == OpenAICompatImageTargetModel &&
			strings.HasPrefix(m.Str, fromModel+"-")
	}
	if shouldReplace {
		if newBody, err := sjson.SetBytes(body, "model", toModel); err == nil {
			return newBody
		}
	}
	return body
}
