package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"log/slog"
	"strings"
	"time"
)

func (s *GatewayService) getUserGroupRateMultiplier(ctx context.Context, userID, groupID int64, groupDefaultMultiplier float64) float64 {
	if s == nil {
		return groupDefaultMultiplier
	}
	resolver := s.userGroupRateResolver
	if resolver == nil {
		resolver = newUserGroupRateResolver(s.userGroupRateRepo, s.userGroupRateCache, resolveUserGroupRateCacheTTL(s.cfg), &s.userGroupRateSF, "service.gateway")
	}
	return resolver.Resolve(ctx, userID, groupID, groupDefaultMultiplier)
}

func usageLogThinkingEnabledFromContext(ctx context.Context) *bool {
	if enabled, ok := ThinkingEnabledFromContext(ctx); ok {
		v := enabled
		return &v
	}
	return nil
}

type RecordUsageInput struct {
	Result             *ForwardResult
	APIKey             *APIKey
	User               *User
	Account            *Account
	Subscription       *UserSubscription
	InboundEndpoint    string
	UpstreamEndpoint   string
	UpstreamURL        string
	UpstreamService    string
	UserAgent          string
	IPAddress          string
	RequestBody        []byte
	RequestPayloadHash string
	ForceCacheBilling  bool
	APIKeyService      APIKeyQuotaUpdater
}
type APIKeyQuotaUpdater interface {
	UpdateQuotaUsed(ctx context.Context, apiKeyID int64, cost float64) error
	UpdateRateLimitUsage(ctx context.Context, apiKeyID int64, cost float64) error
}

type apiKeyGroupQuotaUpdater interface {
	UpdateGroupQuotaUsed(ctx context.Context, apiKeyID, groupID int64, cost float64) error
}
type postUsageBillingParams struct {
	Cost                  *CostBreakdown
	User                  *User
	APIKey                *APIKey
	Account               *Account
	Subscription          *UserSubscription
	IsSubscriptionBill    bool
	RequestPayloadHash    string
	SkipUserBilling       bool
	AccountRateMultiplier float64
	APIKeyService         APIKeyQuotaUpdater
}

func applyBillingExemption(cost *CostBreakdown, user *User) (actualCost float64, billingExemptReason *string, skipUserBilling bool) {
	if user != nil && user.IsAdminFreeBillingEnabled() {
		return 0, BillingExemptReasonPtr(BillingExemptReasonAdminFree), true
	}
	if cost == nil {
		return 0, nil, false
	}
	return cost.ActualCost, nil, false
}

func postUsageBilling(ctx context.Context, p *postUsageBillingParams, deps *billingDeps) {
	billingCtx, cancel := detachedBillingContext(ctx)
	defer cancel()

	cost := p.Cost
	if p.IsSubscriptionBill {
		if !p.SkipUserBilling && cost.TotalCost > 0 && p.Subscription != nil {
			if err := deps.userSubRepo.IncrementUsage(billingCtx, p.Subscription.ID, cost.TotalCost); err != nil {
				slog.Error("increment subscription usage failed", "subscription_id", p.Subscription.ID, "error", err)
			}
		}
	} else {
		if !p.SkipUserBilling && cost.ActualCost > 0 {
			if err := deps.userRepo.DeductBalance(billingCtx, p.User.ID, cost.ActualCost); err != nil {
				slog.Error("deduct balance failed", "user_id", p.User.ID, "error", err)
			}
		}
	}
	if !p.SkipUserBilling && cost.ActualCost > 0 && p.APIKey.Quota > 0 && p.APIKeyService != nil {
		if err := p.APIKeyService.UpdateQuotaUsed(billingCtx, p.APIKey.ID, cost.ActualCost); err != nil {
			slog.Error("update api key quota failed", "api_key_id", p.APIKey.ID, "error", err)
		}
	}
	if !p.SkipUserBilling && cost.ActualCost > 0 && p.APIKeyService != nil && p.APIKey != nil && p.APIKey.GroupID != nil {
		if updater, ok := p.APIKeyService.(apiKeyGroupQuotaUpdater); ok {
			if err := updater.UpdateGroupQuotaUsed(billingCtx, p.APIKey.ID, *p.APIKey.GroupID, cost.ActualCost); err != nil {
				slog.Error("update api key group quota failed", "api_key_id", p.APIKey.ID, "group_id", *p.APIKey.GroupID, "error", err)
			}
		}
	}
	if !p.SkipUserBilling && cost.ActualCost > 0 && p.APIKey.HasRateLimits() && p.APIKeyService != nil {
		if err := p.APIKeyService.UpdateRateLimitUsage(billingCtx, p.APIKey.ID, cost.ActualCost); err != nil {
			slog.Error("update api key rate limit usage failed", "api_key_id", p.APIKey.ID, "error", err)
		}
	}
	if cost.TotalCost > 0 && CanParticipateInAccountQuota(p.Account) && p.Account.HasAnyQuotaLimit() {
		accountCost := cost.TotalCost * p.AccountRateMultiplier
		if err := deps.accountRepo.IncrementQuotaUsed(billingCtx, p.Account.ID, accountCost); err != nil {
			slog.Error("increment account quota used failed", "account_id", p.Account.ID, "cost", accountCost, "error", err)
		}
	}
	finalizePostUsageBilling(p, deps)
}

type billingDeps struct {
	accountRepo         AccountRepository
	userRepo            UserRepository
	userSubRepo         UserSubscriptionRepository
	billingCacheService *BillingCacheService
	deferredService     *DeferredService
}

func (s *GatewayService) billingDeps() *billingDeps {
	return &billingDeps{accountRepo: s.accountRepo, userRepo: s.userRepo, userSubRepo: s.userSubRepo, billingCacheService: s.billingCacheService, deferredService: s.deferredService}
}

func isGeminiBillingEndpoint(inboundEndpoint string) bool {
	normalized := NormalizeInboundEndpoint(inboundEndpoint)
	switch normalized {
	case EndpointGeminiModels,
		EndpointGeminiFiles,
		EndpointGeminiFilesUp,
		EndpointGeminiFilesDownload,
		EndpointGeminiBatches,
		EndpointGoogleBatchArchiveBatches,
		EndpointGoogleBatchArchiveFiles,
		EndpointVertexSyncModels,
		EndpointVertexBatchJobs:
		return true
	}

	path := strings.TrimSpace(strings.ToLower(inboundEndpoint))
	return strings.Contains(path, "/v1beta/openai/") ||
		strings.Contains(path, "/v1beta/live") ||
		strings.Contains(path, "/v1beta/interactions") ||
		strings.Contains(path, "/cachedcontents") ||
		strings.Contains(path, "/filesearchstores") ||
		strings.Contains(path, "/documents") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, "/embeddings")
}

func geminiVideoRequestsForUsage(result *ForwardResult) int {
	if result == nil || strings.TrimSpace(strings.ToLower(result.MediaType)) != "video" {
		return 0
	}
	return 1
}

func (s *GatewayService) calculateGeminiGatewayCost(
	ctx context.Context,
	inboundEndpoint string,
	requestBody []byte,
	billingModel string,
	result *ForwardResult,
	multiplier float64,
) (*GeminiBillingCalculationResult, error) {
	if s == nil ||
		s.billingService == nil ||
		s.billingService.billingCenterService == nil ||
		result == nil ||
		!isGeminiBillingEndpoint(inboundEndpoint) {
		return nil, nil
	}
	requestedServiceTier := ""
	if result.ServiceTier != nil {
		requestedServiceTier = strings.TrimSpace(*result.ServiceTier)
	}
	if requestedServiceTier == "" {
		if extracted := extractGeminiRequestedServiceTierFromBody(requestBody); extracted != nil {
			requestedServiceTier = strings.TrimSpace(*extracted)
		}
	}

	return s.billingService.billingCenterService.CalculateGeminiCost(ctx, GeminiBillingCalculationInput{
		Model:                billingModel,
		InboundEndpoint:      inboundEndpoint,
		RequestBody:          requestBody,
		RequestedServiceTier: requestedServiceTier,
		Tokens: UsageTokens{
			InputTokens:           result.Usage.InputTokens,
			OutputTokens:          result.Usage.OutputTokens,
			CacheCreationTokens:   result.Usage.CacheCreationInputTokens,
			CacheReadTokens:       result.Usage.CacheReadInputTokens,
			CacheCreation5mTokens: result.Usage.CacheCreation5mTokens,
			CacheCreation1hTokens: result.Usage.CacheCreation1hTokens,
		},
		ImageCount:     result.ImageCount,
		VideoRequests:  geminiVideoRequestsForUsage(result),
		MediaType:      result.MediaType,
		RateMultiplier: multiplier,
	})
}

func applyGeminiClassificationToUsageLog(usageLog *UsageLog, classification *GeminiRequestClassification, fallbackMediaType string) {
	if usageLog == nil || classification == nil {
		return
	}

	usageLog.OperationType = optionalTrimmedStringPtr(classification.OperationType)
	usageLog.ChargeSource = optionalTrimmedStringPtr(classification.ChargeSource)
	usageLog.ServiceTier = optionalTrimmedStringPtr(classification.ServiceTier)
	usageLog.GeminiSurface = optionalTrimmedStringPtr(classification.Surface)
	usageLog.GeminiBatchMode = optionalTrimmedStringPtr(classification.BatchMode)
	usageLog.GeminiCachePhase = optionalTrimmedStringPtr(classification.CachePhase)
	usageLog.GeminiGroundingKind = optionalTrimmedStringPtr(classification.GroundingKind)
	usageLog.GeminiInputModality = optionalTrimmedStringPtr(classification.InputModality)
	usageLog.GeminiOutputModality = optionalTrimmedStringPtr(classification.OutputModality)

	mediaType := classification.MediaType
	if strings.TrimSpace(mediaType) == "" {
		mediaType = fallbackMediaType
	}
	usageLog.MediaType = optionalTrimmedStringPtr(mediaType)
}

func (s *GatewayService) calculateGatewayMediaCost(result *ForwardResult, apiKey *APIKey, account *Account, multiplier float64) *CostBreakdown {
	if s == nil || s.billingService == nil || result == nil {
		return &CostBreakdown{}
	}

	platform := RoutingPlatformForAccount(account)
	if result.MediaType == "prompt" {
		return &CostBreakdown{}
	}
	if result.MediaType == "image" {
		var groupConfig *ImagePriceConfig
		if apiKey != nil && apiKey.Group != nil {
			groupConfig = &ImagePriceConfig{Price1K: apiKey.Group.ImagePrice1K, Price2K: apiKey.Group.ImagePrice2K, Price4K: apiKey.Group.ImagePrice4K}
		}
		return s.billingService.CalculateImageCost(result.Model, result.ImageSize, result.ImageCount, groupConfig, multiplier)
	}
	if result.MediaType == "video" {
		if platform == PlatformGrok {
			return s.billingService.CalculateVideoRequestCost(result.Model, multiplier)
		}
		return &CostBreakdown{}
	}
	if result.ImageCount > 0 {
		var groupConfig *ImagePriceConfig
		if apiKey != nil && apiKey.Group != nil {
			groupConfig = &ImagePriceConfig{Price1K: apiKey.Group.ImagePrice1K, Price2K: apiKey.Group.ImagePrice2K, Price4K: apiKey.Group.ImagePrice4K}
		}
		return s.billingService.CalculateImageCost(result.Model, result.ImageSize, result.ImageCount, groupConfig, multiplier)
	}
	return nil
}

func (s *GatewayService) RecordUsage(ctx context.Context, input *RecordUsageInput) error {
	result := input.Result
	apiKey := input.APIKey
	user := input.User
	account := input.Account
	subscription := input.Subscription
	if input.ForceCacheBilling && result.Usage.InputTokens > 0 {
		logger.LegacyPrintf("service.gateway", "force_cache_billing: %d input_tokens → cache_read_input_tokens (account=%d)", result.Usage.InputTokens, account.ID)
		result.Usage.CacheReadInputTokens += result.Usage.InputTokens
		result.Usage.InputTokens = 0
	}
	cacheTTLOverridden := false
	if account.IsCacheTTLOverrideEnabled() {
		applyCacheTTLOverride(&result.Usage, account.GetCacheTTLOverrideTarget())
		cacheTTLOverridden = (result.Usage.CacheCreation5mTokens + result.Usage.CacheCreation1hTokens) > 0
	}
	multiplier := 1.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if apiKey.GroupID != nil && apiKey.Group != nil {
		groupDefault := apiKey.Group.RateMultiplier
		multiplier = s.getUserGroupRateMultiplier(ctx, user.ID, *apiKey.GroupID, groupDefault)
	}
	tokens := UsageTokens{
		InputTokens:           result.Usage.InputTokens,
		OutputTokens:          result.Usage.OutputTokens,
		CacheCreationTokens:   result.Usage.CacheCreationInputTokens,
		CacheReadTokens:       result.Usage.CacheReadInputTokens,
		CacheCreation5mTokens: result.Usage.CacheCreation5mTokens,
		CacheCreation1hTokens: result.Usage.CacheCreation1hTokens,
	}
	channelResolution := resolveGatewayChannelBilling(ctx, s.channelService, result.Model, result.UpstreamModel, GatewayChannelUsage{
		TotalTokens:       tokens.InputTokens + tokens.OutputTokens + tokens.CacheCreationTokens + tokens.CacheReadTokens + tokens.CacheCreation5mTokens + tokens.CacheCreation1hTokens,
		ImageOutputTokens: result.Usage.OutputTokens,
		ImageCount:        result.ImageCount,
	})
	billingModel := forwardResultBillingModel(result.Model, result.UpstreamModel)
	if channelResolution != nil && channelResolution.BillingModel != "" {
		billingModel = channelResolution.BillingModel
	}
	var geminiBillingResult *GeminiBillingCalculationResult
	var cost *CostBreakdown
	if geminiCost, err := s.calculateGeminiGatewayCost(ctx, input.InboundEndpoint, input.RequestBody, billingModel, result, multiplier); err != nil {
		logger.LegacyPrintf("service.gateway", "Calculate Gemini billing center cost failed: %v", err)
	} else {
		geminiBillingResult = geminiCost
	}
	if geminiBillingResult != nil && geminiBillingResult.Cost != nil {
		cost = geminiBillingResult.Cost
		applyGeminiBillingMetadataToContext(ctx, geminiBillingResult)
	} else {
		cost = s.calculateGatewayMediaCost(result, apiKey, account, multiplier)
	}
	if cost == nil {
		var err error
		cost, err = s.billingService.CalculateCost(billingModel, tokens, multiplier)
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Calculate cost failed: %v", err)
			cost = &CostBreakdown{ActualCost: 0}
		}
	}
	var channelPricing *GatewayChannelResolvedPricing
	if channelResolution != nil {
		channelPricing = channelResolution.Pricing
	}
	cost, imageOutputTokens, imageOutputCost := applyChannelPricingOverride(cost, channelPricing, tokens, multiplier, result.ImageCount)
	isSubscriptionBilling := subscription != nil && apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	billingType := BillingTypeBalance
	if isSubscriptionBilling {
		billingType = BillingTypeSubscription
	}
	durationMs := int(result.Duration.Milliseconds())
	var imageSize *string
	if result.ImageSize != "" {
		imageSize = &result.ImageSize
	}
	accountRateMultiplier := account.BillingRateMultiplier()
	actualCost, billingExemptReason, skipUserBilling := applyBillingExemption(cost, user)
	requestID := resolveUsageBillingRequestID(ctx, result.RequestID)
	usageLog := &UsageLog{UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID, RequestID: requestID, Model: result.Model, RequestedModel: result.Model, UpstreamModel: optionalNonEqualStringPtr(result.UpstreamModel, result.Model), ReasoningEffort: result.ReasoningEffort, ThinkingEnabled: usageLogThinkingEnabledFromContext(ctx), InboundEndpoint: optionalTrimmedStringPtr(input.InboundEndpoint), UpstreamEndpoint: optionalTrimmedStringPtr(input.UpstreamEndpoint), UpstreamURL: optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, input.UpstreamURL)), UpstreamService: optionalTrimmedStringPtr(ResolveUsageLogUpstreamService(account, input.UpstreamService)), InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationTokens: result.Usage.CacheCreationInputTokens, CacheReadTokens: result.Usage.CacheReadInputTokens, CacheCreation5mTokens: result.Usage.CacheCreation5mTokens, CacheCreation1hTokens: result.Usage.CacheCreation1hTokens, InputCost: cost.InputCost, OutputCost: cost.OutputCost, CacheCreationCost: cost.CacheCreationCost, CacheReadCost: cost.CacheReadCost, TotalCost: cost.TotalCost, ActualCost: actualCost, BillingExemptReason: billingExemptReason, RateMultiplier: multiplier, AccountRateMultiplier: &accountRateMultiplier, BillingType: billingType, Status: UsageLogStatusSucceeded, Stream: result.Stream, DurationMs: &durationMs, FirstTokenMs: result.FirstTokenMs, ImageCount: result.ImageCount, ImageSize: imageSize, CacheTTLOverridden: cacheTTLOverridden, CreatedAt: time.Now()}
	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}
	if apiKey.GroupID != nil {
		usageLog.GroupID = apiKey.GroupID
	}
	if subscription != nil {
		usageLog.SubscriptionID = &subscription.ID
	}
	if simulatedClient := NormalizeUsageLogSimulatedClient(result.SimulatedClient); simulatedClient != nil {
		usageLog.SimulatedClient = simulatedClient
	}
	if geminiBillingResult != nil {
		applyGeminiClassificationToUsageLog(usageLog, geminiBillingResult.Classification, result.MediaType)
	}
	applyGatewayChannelUsageLogMetadata(usageLog, channelResolution, imageOutputTokens, imageOutputCost)
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.gateway")
		logger.LegacyPrintf("service.gateway", "[SIMPLE MODE] Usage recorded (not billed): user=%d, tokens=%d", usageLog.UserID, usageLog.TotalTokens())
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}
	if _, billingErr := applyUsageBilling(ctx, requestID, usageLog, &postUsageBillingParams{
		Cost:                  cost,
		User:                  user,
		APIKey:                apiKey,
		Account:               account,
		Subscription:          subscription,
		RequestPayloadHash:    resolveUsageBillingPayloadFingerprint(ctx, input.RequestPayloadHash),
		IsSubscriptionBill:    isSubscriptionBilling,
		SkipUserBilling:       skipUserBilling,
		AccountRateMultiplier: accountRateMultiplier,
		APIKeyService:         input.APIKeyService,
	}, s.billingDeps(), s.usageBillingRepo); billingErr != nil {
		return billingErr
	}
	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.gateway")
	return nil
}

type RecordUsageLongContextInput struct {
	Result                *ForwardResult
	APIKey                *APIKey
	User                  *User
	Account               *Account
	Subscription          *UserSubscription
	InboundEndpoint       string
	UpstreamEndpoint      string
	UpstreamURL           string
	UpstreamService       string
	UserAgent             string
	IPAddress             string
	RequestBody           []byte
	RequestPayloadHash    string
	LongContextThreshold  int
	LongContextMultiplier float64
	ForceCacheBilling     bool
	APIKeyService         APIKeyQuotaUpdater
}

func (s *GatewayService) RecordUsageWithLongContext(ctx context.Context, input *RecordUsageLongContextInput) error {
	result := input.Result
	apiKey := input.APIKey
	user := input.User
	account := input.Account
	subscription := input.Subscription
	if input.ForceCacheBilling && result.Usage.InputTokens > 0 {
		logger.LegacyPrintf("service.gateway", "force_cache_billing: %d input_tokens → cache_read_input_tokens (account=%d)", result.Usage.InputTokens, account.ID)
		result.Usage.CacheReadInputTokens += result.Usage.InputTokens
		result.Usage.InputTokens = 0
	}
	cacheTTLOverridden := false
	if account.IsCacheTTLOverrideEnabled() {
		applyCacheTTLOverride(&result.Usage, account.GetCacheTTLOverrideTarget())
		cacheTTLOverridden = (result.Usage.CacheCreation5mTokens + result.Usage.CacheCreation1hTokens) > 0
	}
	multiplier := 1.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if apiKey.GroupID != nil && apiKey.Group != nil {
		groupDefault := apiKey.Group.RateMultiplier
		multiplier = s.getUserGroupRateMultiplier(ctx, user.ID, *apiKey.GroupID, groupDefault)
	}
	tokens := UsageTokens{
		InputTokens:           result.Usage.InputTokens,
		OutputTokens:          result.Usage.OutputTokens,
		CacheCreationTokens:   result.Usage.CacheCreationInputTokens,
		CacheReadTokens:       result.Usage.CacheReadInputTokens,
		CacheCreation5mTokens: result.Usage.CacheCreation5mTokens,
		CacheCreation1hTokens: result.Usage.CacheCreation1hTokens,
	}
	channelResolution := resolveGatewayChannelBilling(ctx, s.channelService, result.Model, result.UpstreamModel, GatewayChannelUsage{
		TotalTokens:       tokens.InputTokens + tokens.OutputTokens + tokens.CacheCreationTokens + tokens.CacheReadTokens + tokens.CacheCreation5mTokens + tokens.CacheCreation1hTokens,
		ImageOutputTokens: result.Usage.OutputTokens,
		ImageCount:        result.ImageCount,
	})
	billingModel := forwardResultBillingModel(result.Model, result.UpstreamModel)
	if channelResolution != nil && channelResolution.BillingModel != "" {
		billingModel = channelResolution.BillingModel
	}
	var geminiBillingResult *GeminiBillingCalculationResult
	var cost *CostBreakdown
	if geminiCost, err := s.calculateGeminiGatewayCost(ctx, input.InboundEndpoint, input.RequestBody, billingModel, result, multiplier); err != nil {
		logger.LegacyPrintf("service.gateway", "Calculate Gemini billing center cost failed: %v", err)
	} else {
		geminiBillingResult = geminiCost
	}
	if geminiBillingResult != nil && geminiBillingResult.Cost != nil {
		cost = geminiBillingResult.Cost
		applyGeminiBillingMetadataToContext(ctx, geminiBillingResult)
	} else if result.ImageCount > 0 {
		var groupConfig *ImagePriceConfig
		if apiKey.Group != nil {
			groupConfig = &ImagePriceConfig{Price1K: apiKey.Group.ImagePrice1K, Price2K: apiKey.Group.ImagePrice2K, Price4K: apiKey.Group.ImagePrice4K}
		}
		cost = s.billingService.CalculateImageCost(billingModel, result.ImageSize, result.ImageCount, groupConfig, multiplier)
	} else {
		var err error
		cost, err = s.billingService.CalculateCostWithLongContext(billingModel, tokens, multiplier, input.LongContextThreshold, input.LongContextMultiplier)
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Calculate cost failed: %v", err)
			cost = &CostBreakdown{ActualCost: 0}
		}
	}
	var channelPricing *GatewayChannelResolvedPricing
	if channelResolution != nil {
		channelPricing = channelResolution.Pricing
	}
	cost, imageOutputTokens, imageOutputCost := applyChannelPricingOverride(cost, channelPricing, tokens, multiplier, result.ImageCount)
	isSubscriptionBilling := subscription != nil && apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	billingType := BillingTypeBalance
	if isSubscriptionBilling {
		billingType = BillingTypeSubscription
	}
	durationMs := int(result.Duration.Milliseconds())
	var imageSize *string
	if result.ImageSize != "" {
		imageSize = &result.ImageSize
	}
	accountRateMultiplier := account.BillingRateMultiplier()
	actualCost, billingExemptReason, skipUserBilling := applyBillingExemption(cost, user)
	requestID := resolveUsageBillingRequestID(ctx, result.RequestID)
	usageLog := &UsageLog{UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID, RequestID: requestID, Model: result.Model, RequestedModel: result.Model, UpstreamModel: optionalNonEqualStringPtr(result.UpstreamModel, result.Model), ReasoningEffort: result.ReasoningEffort, ThinkingEnabled: usageLogThinkingEnabledFromContext(ctx), InboundEndpoint: optionalTrimmedStringPtr(input.InboundEndpoint), UpstreamEndpoint: optionalTrimmedStringPtr(input.UpstreamEndpoint), UpstreamURL: optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, input.UpstreamURL)), UpstreamService: optionalTrimmedStringPtr(ResolveUsageLogUpstreamService(account, input.UpstreamService)), InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationTokens: result.Usage.CacheCreationInputTokens, CacheReadTokens: result.Usage.CacheReadInputTokens, CacheCreation5mTokens: result.Usage.CacheCreation5mTokens, CacheCreation1hTokens: result.Usage.CacheCreation1hTokens, InputCost: cost.InputCost, OutputCost: cost.OutputCost, CacheCreationCost: cost.CacheCreationCost, CacheReadCost: cost.CacheReadCost, TotalCost: cost.TotalCost, ActualCost: actualCost, BillingExemptReason: billingExemptReason, RateMultiplier: multiplier, AccountRateMultiplier: &accountRateMultiplier, BillingType: billingType, Status: UsageLogStatusSucceeded, Stream: result.Stream, DurationMs: &durationMs, FirstTokenMs: result.FirstTokenMs, ImageCount: result.ImageCount, ImageSize: imageSize, CacheTTLOverridden: cacheTTLOverridden, CreatedAt: time.Now()}
	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}
	if apiKey.GroupID != nil {
		usageLog.GroupID = apiKey.GroupID
	}
	if subscription != nil {
		usageLog.SubscriptionID = &subscription.ID
	}
	if simulatedClient := NormalizeUsageLogSimulatedClient(result.SimulatedClient); simulatedClient != nil {
		usageLog.SimulatedClient = simulatedClient
	}
	if geminiBillingResult != nil {
		applyGeminiClassificationToUsageLog(usageLog, geminiBillingResult.Classification, result.MediaType)
	}
	applyGatewayChannelUsageLogMetadata(usageLog, channelResolution, imageOutputTokens, imageOutputCost)
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.gateway")
		logger.LegacyPrintf("service.gateway", "[SIMPLE MODE] Usage recorded (not billed): user=%d, tokens=%d", usageLog.UserID, usageLog.TotalTokens())
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}
	if _, billingErr := applyUsageBilling(ctx, requestID, usageLog, &postUsageBillingParams{
		Cost:                  cost,
		User:                  user,
		APIKey:                apiKey,
		Account:               account,
		Subscription:          subscription,
		RequestPayloadHash:    resolveUsageBillingPayloadFingerprint(ctx, input.RequestPayloadHash),
		IsSubscriptionBill:    isSubscriptionBilling,
		SkipUserBilling:       skipUserBilling,
		AccountRateMultiplier: accountRateMultiplier,
		APIKeyService:         input.APIKeyService,
	}, s.billingDeps(), s.usageBillingRepo); billingErr != nil {
		return billingErr
	}
	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.gateway")
	return nil
}

func applyGeminiBillingMetadataToContext(ctx context.Context, result *GeminiBillingCalculationResult) {
	if result == nil {
		return
	}
	if result.Classification != nil {
		SetGeminiSurfaceMetadata(ctx, result.Classification.Surface)
		SetGeminiRequestedServiceTierMetadata(ctx, result.Classification.RequestedServiceTier)
		SetGeminiResolvedServiceTierMetadata(ctx, result.Classification.ServiceTier)
		SetGeminiBatchModeMetadata(ctx, result.Classification.BatchMode)
		SetGeminiCachePhaseMetadata(ctx, result.Classification.CachePhase)
	}
	if len(result.MatchedRuleIDs) > 0 {
		SetBillingRuleIDMetadata(ctx, result.MatchedRuleIDs[0])
	}
	if result.Fallback != nil {
		SetGeminiBillingFallbackReasonMetadata(ctx, result.Fallback.Reason)
	}
}
