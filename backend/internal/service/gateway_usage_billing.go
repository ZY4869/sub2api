package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"log/slog"
	"strings"
	"time"

	"go.uber.org/zap"
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
	UserAgent          string
	IPAddress          string
	RequestPayloadHash string
	ForceCacheBilling  bool
	APIKeyService      APIKeyQuotaUpdater
}
type APIKeyQuotaUpdater interface {
	UpdateQuotaUsed(ctx context.Context, apiKeyID int64, cost float64) error
	UpdateRateLimitUsage(ctx context.Context, apiKeyID int64, cost float64) error
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
	if !p.SkipUserBilling && cost.ActualCost > 0 && p.APIKey.HasRateLimits() && p.APIKeyService != nil {
		if err := p.APIKeyService.UpdateRateLimitUsage(billingCtx, p.APIKey.ID, cost.ActualCost); err != nil {
			slog.Error("update api key rate limit usage failed", "api_key_id", p.APIKey.ID, "error", err)
		}
	}
	if cost.TotalCost > 0 && p.Account.IsAPIKeyOrBedrock() && p.Account.HasAnyQuotaLimit() {
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
	var cost *CostBreakdown
	if result.MediaType == "image" || result.MediaType == "video" {
		var soraConfig *SoraPriceConfig
		if apiKey.Group != nil {
			soraConfig = &SoraPriceConfig{ImagePrice360: apiKey.Group.SoraImagePrice360, ImagePrice540: apiKey.Group.SoraImagePrice540, VideoPricePerRequest: apiKey.Group.SoraVideoPricePerRequest, VideoPricePerRequestHD: apiKey.Group.SoraVideoPricePerRequestHD}
		}
		if result.MediaType == "image" {
			cost = s.billingService.CalculateSoraImageCost(result.ImageSize, result.ImageCount, soraConfig, multiplier)
		} else {
			cost = s.billingService.CalculateSoraVideoCost(result.Model, soraConfig, multiplier)
		}
	} else if result.MediaType == "prompt" {
		cost = &CostBreakdown{}
	} else if result.ImageCount > 0 {
		var groupConfig *ImagePriceConfig
		if apiKey.Group != nil {
			groupConfig = &ImagePriceConfig{Price1K: apiKey.Group.ImagePrice1K, Price2K: apiKey.Group.ImagePrice2K, Price4K: apiKey.Group.ImagePrice4K}
		}
		cost = s.billingService.CalculateImageCost(result.Model, result.ImageSize, result.ImageCount, groupConfig, multiplier)
	} else {
		tokens := UsageTokens{InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationTokens: result.Usage.CacheCreationInputTokens, CacheReadTokens: result.Usage.CacheReadInputTokens, CacheCreation5mTokens: result.Usage.CacheCreation5mTokens, CacheCreation1hTokens: result.Usage.CacheCreation1hTokens}
		var err error
		cost, err = s.billingService.CalculateCost(result.Model, tokens, multiplier)
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Calculate cost failed: %v", err)
			cost = &CostBreakdown{ActualCost: 0}
		}
	}
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
	var mediaType *string
	if strings.TrimSpace(result.MediaType) != "" {
		mediaType = &result.MediaType
	}
	accountRateMultiplier := account.BillingRateMultiplier()
	actualCost, billingExemptReason, skipUserBilling := applyBillingExemption(cost, user)
	usageLog := &UsageLog{UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID, RequestID: result.RequestID, Model: result.Model, ReasoningEffort: result.ReasoningEffort, ThinkingEnabled: usageLogThinkingEnabledFromContext(ctx), InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationTokens: result.Usage.CacheCreationInputTokens, CacheReadTokens: result.Usage.CacheReadInputTokens, CacheCreation5mTokens: result.Usage.CacheCreation5mTokens, CacheCreation1hTokens: result.Usage.CacheCreation1hTokens, InputCost: cost.InputCost, OutputCost: cost.OutputCost, CacheCreationCost: cost.CacheCreationCost, CacheReadCost: cost.CacheReadCost, TotalCost: cost.TotalCost, ActualCost: actualCost, BillingExemptReason: billingExemptReason, RateMultiplier: multiplier, AccountRateMultiplier: &accountRateMultiplier, BillingType: billingType, Stream: result.Stream, DurationMs: &durationMs, FirstTokenMs: result.FirstTokenMs, ImageCount: result.ImageCount, ImageSize: imageSize, MediaType: mediaType, CacheTTLOverridden: cacheTTLOverridden, CreatedAt: time.Now()}
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
	inserted, err := s.usageLogRepo.Create(ctx, usageLog)
	if err != nil {
		logger.LegacyPrintf("service.gateway", "Create usage log failed: %v", err)
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		logger.LegacyPrintf("service.gateway", "[SIMPLE MODE] Usage recorded (not billed): user=%d, tokens=%d", usageLog.UserID, usageLog.TotalTokens())
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}
	shouldBill := inserted || err != nil
	if shouldBill {
		if skipUserBilling {
			logger.With(zap.String("component", "billing.admin_free"), zap.Int64("user_id", user.ID), zap.String("reason", BillingExemptReasonAdminFree)).Info("admin free billing applied")
		}
		postUsageBilling(ctx, &postUsageBillingParams{Cost: cost, User: user, APIKey: apiKey, Account: account, Subscription: subscription, IsSubscriptionBill: isSubscriptionBilling, SkipUserBilling: skipUserBilling, AccountRateMultiplier: accountRateMultiplier, APIKeyService: input.APIKeyService}, s.billingDeps())
	} else {
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
	}
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
	UserAgent             string
	IPAddress             string
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
	var cost *CostBreakdown
	if result.ImageCount > 0 {
		var groupConfig *ImagePriceConfig
		if apiKey.Group != nil {
			groupConfig = &ImagePriceConfig{Price1K: apiKey.Group.ImagePrice1K, Price2K: apiKey.Group.ImagePrice2K, Price4K: apiKey.Group.ImagePrice4K}
		}
		cost = s.billingService.CalculateImageCost(result.Model, result.ImageSize, result.ImageCount, groupConfig, multiplier)
	} else {
		tokens := UsageTokens{InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationTokens: result.Usage.CacheCreationInputTokens, CacheReadTokens: result.Usage.CacheReadInputTokens, CacheCreation5mTokens: result.Usage.CacheCreation5mTokens, CacheCreation1hTokens: result.Usage.CacheCreation1hTokens}
		var err error
		cost, err = s.billingService.CalculateCostWithLongContext(result.Model, tokens, multiplier, input.LongContextThreshold, input.LongContextMultiplier)
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Calculate cost failed: %v", err)
			cost = &CostBreakdown{ActualCost: 0}
		}
	}
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
	usageLog := &UsageLog{UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID, RequestID: result.RequestID, Model: result.Model, ReasoningEffort: result.ReasoningEffort, ThinkingEnabled: usageLogThinkingEnabledFromContext(ctx), InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationTokens: result.Usage.CacheCreationInputTokens, CacheReadTokens: result.Usage.CacheReadInputTokens, CacheCreation5mTokens: result.Usage.CacheCreation5mTokens, CacheCreation1hTokens: result.Usage.CacheCreation1hTokens, InputCost: cost.InputCost, OutputCost: cost.OutputCost, CacheCreationCost: cost.CacheCreationCost, CacheReadCost: cost.CacheReadCost, TotalCost: cost.TotalCost, ActualCost: actualCost, BillingExemptReason: billingExemptReason, RateMultiplier: multiplier, AccountRateMultiplier: &accountRateMultiplier, BillingType: billingType, Stream: result.Stream, DurationMs: &durationMs, FirstTokenMs: result.FirstTokenMs, ImageCount: result.ImageCount, ImageSize: imageSize, CacheTTLOverridden: cacheTTLOverridden, CreatedAt: time.Now()}
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
	inserted, err := s.usageLogRepo.Create(ctx, usageLog)
	if err != nil {
		logger.LegacyPrintf("service.gateway", "Create usage log failed: %v", err)
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		logger.LegacyPrintf("service.gateway", "[SIMPLE MODE] Usage recorded (not billed): user=%d, tokens=%d", usageLog.UserID, usageLog.TotalTokens())
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}
	shouldBill := inserted || err != nil
	if shouldBill {
		if skipUserBilling {
			logger.With(zap.String("component", "billing.admin_free"), zap.Int64("user_id", user.ID), zap.String("reason", BillingExemptReasonAdminFree)).Info("admin free billing applied")
		}
		postUsageBilling(ctx, &postUsageBillingParams{Cost: cost, User: user, APIKey: apiKey, Account: account, Subscription: subscription, IsSubscriptionBill: isSubscriptionBilling, SkipUserBilling: skipUserBilling, AccountRateMultiplier: accountRateMultiplier, APIKeyService: input.APIKeyService}, s.billingDeps())
	} else {
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
	}
	return nil
}
