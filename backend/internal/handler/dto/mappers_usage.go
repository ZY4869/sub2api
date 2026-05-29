package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

// AccountSummaryFromService returns a minimal AccountSummary for usage log display.
// Only includes ID and Name - no sensitive fields like Credentials, Proxy, etc.
func AccountSummaryFromService(a *service.Account) *AccountSummary {
	if a == nil {
		return nil
	}
	return &AccountSummary{
		ID:   a.ID,
		Name: a.Name,
	}
}

func usageLogFromServiceUser(l *service.UsageLog) UsageLog {
	// 普通用户 DTO：严禁包含管理员字段（例如 account_rate_multiplier、ip_address、account）。
	requestType := l.EffectiveRequestType()
	stream, openAIWSMode := service.ApplyLegacyRequestFields(requestType, l.Stream, l.OpenAIWSMode)
	legacyTotalCost := usageLegacyUSDCost(l.TotalCost, l.TotalCostUSDEquivalent, l.BillingCurrency)
	legacyActualCost := usageLegacyUSDCost(l.ActualCost, l.ActualCostUSDEquivalent, l.BillingCurrency)
	requestedModel := l.RequestedModel
	if requestedModel == "" {
		requestedModel = l.Model
	}
	return UsageLog{
		ID:                         l.ID,
		UserID:                     l.UserID,
		APIKeyID:                   l.APIKeyID,
		AccountID:                  l.AccountID,
		RequestID:                  l.RequestID,
		Model:                      requestedModel,
		UpstreamModel:              l.UpstreamModel,
		ServiceTier:                l.ServiceTier,
		ReasoningEffort:            l.ReasoningEffort,
		ReasoningEffortRaw:         l.ReasoningEffortRaw,
		ReasoningEffortEffective:   l.ReasoningEffortEffective,
		RequestedModelRaw:          l.RequestedModelRaw,
		RequestedModelNormalized:   l.RequestedModelNormalized,
		RequestContextLengthTokens: l.RequestContextLengthTokens,
		MillionContextRequested:    l.MillionContextRequested,
		MillionContextEffective:    l.MillionContextEffective,
		MillionContextSource:       l.MillionContextSource,
		MillionContextBetaToken:    l.MillionContextBetaToken,
		ThinkingEnabled:            l.ThinkingEnabled,
		InboundEndpoint:            l.InboundEndpoint,
		UpstreamEndpoint:           l.UpstreamEndpoint,
		ChannelID:                  l.ChannelID,
		ModelMappingChain:          l.ModelMappingChain,
		BillingTier:                l.BillingTier,
		BillingMode:                l.BillingMode,
		GroupID:                    l.GroupID,
		SubscriptionID:             l.SubscriptionID,
		InputTokens:                l.InputTokens,
		OutputTokens:               l.OutputTokens,
		CacheCreationTokens:        l.CacheCreationTokens,
		CacheReadTokens:            l.CacheReadTokens,
		CacheCreation5mTokens:      l.CacheCreation5mTokens,
		CacheCreation1hTokens:      l.CacheCreation1hTokens,
		InputCost:                  l.InputCost,
		OutputCost:                 l.OutputCost,
		CacheCreationCost:          l.CacheCreationCost,
		CacheReadCost:              l.CacheReadCost,
		TotalCost:                  legacyTotalCost,
		ActualCost:                 legacyActualCost,
		BillingCurrency:            service.NormalizeUsageBillingCurrency(l.BillingCurrency),
		TotalCostUSDEquivalent:     l.TotalCostUSDEquivalent,
		ActualCostUSDEquivalent:    l.ActualCostUSDEquivalent,
		CostByCurrency:             cloneUsageCostByCurrency(l.CostByCurrency),
		ActualCostByCurrency:       cloneUsageCostByCurrency(l.ActualCostByCurrency),
		RateMultiplier:             l.RateMultiplier,
		BillingType:                l.BillingType,
		RequestType:                requestType.String(),
		Status:                     service.NormalizeUsageLogStatus(l.Status),
		Stream:                     stream,
		OpenAIWSMode:               openAIWSMode,
		DurationMs:                 l.DurationMs,
		FirstTokenMs:               l.FirstTokenMs,
		HTTPStatus:                 l.HTTPStatus,
		ErrorCode:                  l.ErrorCode,
		ErrorMessage:               l.ErrorMessage,
		SimulatedClient:            l.SimulatedClient,
		OperationType:              l.OperationType,
		ChargeSource:               l.ChargeSource,
		ImageCount:                 l.ImageCount,
		ImageSize:                  l.ImageSize,
		ImageOutputTokens:          l.ImageOutputTokens,
		ImageOutputCost:            l.ImageOutputCost,
		UserAgent:                  l.UserAgent,
		CacheTTLOverridden:         l.CacheTTLOverridden,
		BillingExemptReason:        l.BillingExemptReason,
		CreatedAt:                  l.CreatedAt,
		User:                       UserFromServiceShallow(l.User),
		APIKey:                     APIKeyFromService(l.APIKey),
		Group:                      GroupFromServiceShallow(l.Group),
		Subscription:               UserSubscriptionFromService(l.Subscription),
	}
}

func usageLegacyUSDCost(sourceAmount, usdEquivalent float64, currency string) float64 {
	if usdEquivalent != 0 {
		return usdEquivalent
	}
	if service.NormalizeUsageBillingCurrency(currency) == service.ModelPricingCurrencyUSD {
		return sourceAmount
	}
	return 0
}

func cloneUsageCostByCurrency(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]float64, len(values))
	for currency, amount := range values {
		normalized := service.NormalizeUsageBillingCurrency(currency)
		if normalized == "" {
			continue
		}
		out[normalized] += amount
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// UsageLogFromService converts a service UsageLog to DTO for regular users.
// It excludes Account details and IP address - users should not see these.
func UsageLogFromService(l *service.UsageLog) *UsageLog {
	if l == nil {
		return nil
	}
	u := usageLogFromServiceUser(l)
	return &u
}

// UsageLogFromServiceAdmin converts a service UsageLog to DTO for admin users.
// It includes minimal Account info (ID, Name only) and IP address.
func UsageLogFromServiceAdmin(l *service.UsageLog) *AdminUsageLog {
	if l == nil {
		return nil
	}
	return &AdminUsageLog{
		UsageLog:              usageLogFromServiceUser(l),
		AccountRateMultiplier: l.AccountRateMultiplier,
		IPAddress:             l.IPAddress,
		Account:               AccountSummaryFromService(l.Account),
	}
}

func UsageRequestPreviewFromService(preview *service.UsageRequestPreview) *UsageRequestPreview {
	if preview == nil {
		return nil
	}
	return &UsageRequestPreview{
		Available:             preview.Available,
		RequestID:             preview.RequestID,
		CapturedAt:            preview.CapturedAt,
		InboundRequestJSON:    preview.InboundRequestJSON,
		NormalizedRequestJSON: preview.NormalizedRequestJSON,
		UpstreamRequestJSON:   preview.UpstreamRequestJSON,
		UpstreamResponseJSON:  preview.UpstreamResponseJSON,
		GatewayResponseJSON:   preview.GatewayResponseJSON,
		ToolTraceJSON:         preview.ToolTraceJSON,
	}
}

func requestTypeStringPtr(requestType *int16) *string {
	if requestType == nil {
		return nil
	}
	value := service.RequestTypeFromInt16(*requestType).String()
	return &value
}
