package service

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	OpenAIAlphaSearchUsageModel     = "alpha/search"
	DefaultWebSearchPricePerCallUSD = 0.01
)

type OpenAIRecordWebSearchUsageInput struct {
	Result             *OpenAIAlphaSearchForwardResult
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
	RequestPayloadHash string
	APIKeyService      APIKeyQuotaUpdater
}

func ResolveWebSearchPricePerCallUSD(group *Group) float64 {
	if group == nil || group.WebSearchPricePerCall == nil || *group.WebSearchPricePerCall < 0 {
		return DefaultWebSearchPricePerCallUSD
	}
	return *group.WebSearchPricePerCall
}

func (s *OpenAIGatewayService) RecordWebSearchUsage(ctx context.Context, input *OpenAIRecordWebSearchUsageInput) error {
	if input == nil || input.Result == nil || input.APIKey == nil || input.User == nil || input.Account == nil {
		return nil
	}
	if input.Result.StatusCode < http.StatusOK || input.Result.StatusCode >= http.StatusMultipleChoices {
		return nil
	}
	apiKey := input.APIKey
	user := input.User
	account := input.Account
	group := apiKey.Group
	baseMultiplier := defaultGatewayRateMultiplier(s.cfg)
	if apiKey.GroupID != nil && group != nil {
		resolver := s.userGroupRateResolver
		if resolver == nil {
			resolver = newUserGroupRateResolver(nil, nil, resolveUserGroupRateCacheTTL(s.cfg), nil, "service.openai_gateway")
		}
		baseMultiplier = resolver.Resolve(ctx, user.ID, *apiKey.GroupID, group.RateMultiplier)
	}
	price := ResolveWebSearchPricePerCallUSD(group)
	multiplier := effectiveFlatRateMultiplier(baseMultiplier, group)
	totalCost := price
	actualCostBeforeExemption := price * multiplier
	cost := &CostBreakdown{
		Currency:                ModelPricingCurrencyUSD,
		TotalCost:               totalCost,
		ActualCost:              actualCostBeforeExemption,
		TotalCostUSDEquivalent:  totalCost,
		ActualCostUSDEquivalent: actualCostBeforeExemption,
		CostByCurrency:          normalizedBillingCostMap(ModelPricingCurrencyUSD, totalCost),
		ActualCostByCurrency:    normalizedBillingCostMap(ModelPricingCurrencyUSD, actualCostBeforeExemption),
	}
	actualCost, billingExemptReason, skipUserBilling := applyBillingExemption(cost, user)
	cost.ActualCost = actualCost
	cost.ActualCostUSDEquivalent = actualCost
	cost.ActualCostByCurrency = normalizedBillingCostMap(ModelPricingCurrencyUSD, actualCost)

	isSubscriptionBilling := input.Subscription != nil && group != nil && group.IsSubscriptionType()
	billingType := BillingTypeBalance
	if isSubscriptionBilling {
		billingType = BillingTypeSubscription
	}
	durationMs := int(input.Result.Duration.Milliseconds())
	accountRateMultiplier := account.BillingRateMultiplier()
	requestID := resolveUsageBillingRequestIDForAPIKey(ctx, input.Result.RequestID, apiKey)
	usageLog := &UsageLog{
		UserID:                  user.ID,
		APIKeyID:                apiKey.ID,
		AccountID:               account.ID,
		RequestID:               requestID,
		Model:                   OpenAIAlphaSearchUsageModel,
		RequestedModel:          OpenAIAlphaSearchUsageModel,
		InboundEndpoint:         optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:        optionalTrimmedStringPtr(input.UpstreamEndpoint),
		UpstreamURL:             optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, input.UpstreamURL)),
		UpstreamService:         optionalTrimmedStringPtr(ResolveUsageLogUpstreamService(account, input.UpstreamService)),
		TotalCost:               cost.TotalCost,
		ActualCost:              cost.ActualCost,
		BillingCurrency:         ModelPricingCurrencyUSD,
		TotalCostUSDEquivalent:  cost.TotalCostUSDEquivalent,
		ActualCostUSDEquivalent: cost.ActualCostUSDEquivalent,
		CostByCurrency:          cloneBillingStringMapFloat64(cost.CostByCurrency),
		ActualCostByCurrency:    cloneBillingStringMapFloat64(cost.ActualCostByCurrency),
		BillingExemptReason:     billingExemptReason,
		RateMultiplier:          multiplier,
		AccountRateMultiplier:   &accountRateMultiplier,
		BillingType:             billingType,
		RequestType:             RequestTypeSync,
		Status:                  UsageLogStatusSucceeded,
		DurationMs:              &durationMs,
		HTTPStatus:              &input.Result.StatusCode,
		CreatedAt:               time.Now(),
	}
	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}
	if apiKey.GroupID != nil {
		usageLog.GroupID = apiKey.GroupID
	}
	if input.Subscription != nil {
		usageLog.SubscriptionID = &input.Subscription.ID
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
		logger.LegacyPrintf("service.openai_gateway", "[SIMPLE MODE] Web search usage recorded (not billed): user=%d", usageLog.UserID)
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}
	if _, billingErr := applyUsageBilling(ctx, requestID, usageLog, &postUsageBillingParams{
		Cost:                  cost,
		User:                  user,
		APIKey:                apiKey,
		Account:               account,
		Subscription:          input.Subscription,
		RequestPayloadHash:    resolveUsageBillingPayloadFingerprint(ctx, input.RequestPayloadHash),
		IsSubscriptionBill:    isSubscriptionBilling,
		SkipUserBilling:       skipUserBilling,
		AccountRateMultiplier: accountRateMultiplier,
		APIKeyService:         input.APIKeyService,
		CurrencyConversion:    billingCurrencyConversionFromSettings(ctx, s.settingService),
	}, s.billingDeps(), s.usageBillingRepo); billingErr != nil {
		markUsageLogBillingFailure(usageLog, billingErr)
		writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
		return billingErr
	}
	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
	return nil
}
