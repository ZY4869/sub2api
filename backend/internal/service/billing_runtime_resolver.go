package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

type BillingRuntimeInput struct {
	Model                string
	Provider             string
	Layer                string
	InboundEndpoint      string
	RequestBody          []byte
	Tokens               UsageTokens
	ImageCount           int
	ImageSize            string
	VideoRequests        int
	MediaType            string
	ServiceTier          string
	RequestedServiceTier string
	ResolvedServiceTier  string
	BatchMode            string
	RateMultiplier       float64
	ImagePriceConfig     *ImagePriceConfig
	LongContextThreshold int
	LongContextMultiplier float64
}

type BillingRuntimeResult struct {
	Cost                   *CostBreakdown
	Classification         *GeminiRequestClassification
	MatchedItems           []string
	FallbackReason         string
	PricingSource          string
	ResolverPath           string
	ChannelOverrideApplied bool
}

type BillingRuntimeResolver struct {
	billingService       *BillingService
	billingCenterService *BillingCenterService
}

func NewBillingRuntimeResolver(billingCenterService *BillingCenterService, billingService *BillingService) *BillingRuntimeResolver {
	return &BillingRuntimeResolver{billingService: billingService, billingCenterService: billingCenterService}
}

func (s *BillingCenterService) RuntimeResolver() *BillingRuntimeResolver {
	if s == nil {
		return nil
	}
	if s.runtimeResolver == nil {
		s.runtimeResolver = NewBillingRuntimeResolver(s, s.billingService)
	}
	return s.runtimeResolver
}

func (s *BillingService) ResolveRuntime(ctx context.Context, input BillingRuntimeInput) (*BillingRuntimeResult, error) {
	if s == nil {
		return &BillingRuntimeResult{Cost: &CostBreakdown{}, PricingSource: "unavailable", ResolverPath: "service_unavailable"}, nil
	}
	if s.billingCenterService != nil {
		return s.billingCenterService.RuntimeResolver().Resolve(ctx, input)
	}
	return NewBillingRuntimeResolver(nil, s).Resolve(ctx, input)
}

func (r *BillingRuntimeResolver) Resolve(ctx context.Context, input BillingRuntimeInput) (*BillingRuntimeResult, error) {
	if r == nil {
		return &BillingRuntimeResult{Cost: &CostBreakdown{}, PricingSource: "unavailable", ResolverPath: "resolver_unavailable"}, nil
	}
	input = normalizeBillingRuntimeInput(input)
	provider := r.resolveProvider(ctx, input)
	if provider == PlatformGemini || isGeminiBillingEndpoint(input.InboundEndpoint) {
		return r.resolveGeminiRuntime(ctx, input)
	}
	if result := r.resolveRuleBasedRuntime(ctx, input, provider); result != nil {
		return result, nil
	}
	return r.resolveLegacyRuntime(input)
}

func normalizeBillingRuntimeInput(input BillingRuntimeInput) BillingRuntimeInput {
	input.Model = strings.TrimSpace(input.Model)
	input.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	input.Layer = normalizeBillingDimension(input.Layer, BillingLayerSale)
	input.ServiceTier = normalizeBillingDimension(input.ServiceTier, "")
	input.RequestedServiceTier = normalizeBillingDimension(input.RequestedServiceTier, "")
	input.ResolvedServiceTier = normalizeBillingDimension(input.ResolvedServiceTier, "")
	input.BatchMode = normalizeBillingActualBatchMode(input.BatchMode)
	input.InboundEndpoint = NormalizeInboundEndpoint(input.InboundEndpoint)
	input.MediaType = strings.TrimSpace(strings.ToLower(input.MediaType))
	if input.RateMultiplier <= 0 {
		input.RateMultiplier = 1
	}
	return input
}

func (r *BillingRuntimeResolver) resolveProvider(ctx context.Context, input BillingRuntimeInput) string {
	if input.Provider != "" {
		return input.Provider
	}
	if r == nil || r.billingCenterService == nil || r.billingCenterService.modelCatalogService == nil || input.Model == "" {
		return ""
	}
	record, err := r.billingCenterService.resolveBillingRecord(ctx, input.Model)
	if err != nil || record == nil {
		return ""
	}
	return strings.TrimSpace(record.provider)
}

func (r *BillingRuntimeResolver) resolveGeminiRuntime(ctx context.Context, input BillingRuntimeInput) (*BillingRuntimeResult, error) {
	if input.InboundEndpoint != "" && isGeminiNonBillablePassthroughEndpoint(input.InboundEndpoint) {
		protocolruntime.RecordBillingResolver("non_billable_passthrough")
		return &BillingRuntimeResult{
			Cost:          &CostBreakdown{},
			PricingSource: "non_billable",
			ResolverPath:  "non_billable_passthrough",
		}, nil
	}
	if r.billingCenterService == nil {
		return r.resolveLegacyRuntime(input)
	}
	result, err := r.billingCenterService.CalculateGeminiCost(ctx, GeminiBillingCalculationInput{
		Model:                input.Model,
		InboundEndpoint:      input.InboundEndpoint,
		RequestBody:          input.RequestBody,
		Tokens:               input.Tokens,
		ImageCount:           input.ImageCount,
		VideoRequests:        input.VideoRequests,
		MediaType:            input.MediaType,
		RateMultiplier:       input.RateMultiplier,
		RequestedServiceTier: input.RequestedServiceTier,
		ResolvedServiceTier:  firstNonEmptyBillingRuntime(input.ResolvedServiceTier, input.ServiceTier),
	})
	if err != nil {
		return nil, err
	}
	path := "gemini_rules"
	source := "billing_rules"
	fallbackReason := ""
	if result != nil && result.Fallback != nil && result.Fallback.Applied {
		path = "gemini_fallback"
		source = "legacy_model_pricing"
		fallbackReason = strings.TrimSpace(result.Fallback.Reason)
		protocolruntime.RecordBillingResolverFallback(fallbackReason)
	}
	protocolruntime.RecordBillingResolver(path)
	return &BillingRuntimeResult{
		Cost:           result.Cost,
		Classification: result.Classification,
		MatchedItems:   append([]string(nil), result.MatchedRuleIDs...),
		FallbackReason: fallbackReason,
		PricingSource:  source,
		ResolverPath:   path,
	}, nil
}

func (r *BillingRuntimeResolver) resolveRuleBasedRuntime(ctx context.Context, input BillingRuntimeInput, provider string) *BillingRuntimeResult {
	if r.billingCenterService == nil || provider == "" {
		return nil
	}
	sim := normalizeSimulationInput(BillingSimulationInput{
		Provider:       provider,
		Layer:          input.Layer,
		Model:          input.Model,
		ServiceTier:    input.ServiceTier,
		BatchMode:      input.BatchMode,
		OutputModality: resolveBillingRuntimeOutputModality(input),
		Charges: BillingSimulationCharges{
			TextInputTokens:   float64(input.Tokens.InputTokens),
			TextOutputTokens:  float64(input.Tokens.OutputTokens),
			CacheCreateTokens: float64(input.Tokens.CacheCreationTokens + input.Tokens.CacheCreation5mTokens + input.Tokens.CacheCreation1hTokens),
			CacheReadTokens:   float64(input.Tokens.CacheReadTokens),
			ImageOutputs:      float64(input.ImageCount),
			VideoRequests:     float64(input.VideoRequests),
		},
	})
	result := r.billingCenterService.evaluateSimulation(
		sim,
		nil,
		r.billingCenterService.ListRules(ctx),
		r.billingCenterService.resolveLongContextThreshold(ctx, input.Model),
		input.RateMultiplier,
	)
	if result == nil || len(result.Lines) == 0 || len(result.UnmatchedDemands) > 0 {
		if result != nil && len(result.UnmatchedDemands) > 0 {
			protocolruntime.RecordBillingResolverFallback("partial_rule_match")
		}
		return nil
	}
	protocolruntime.RecordBillingResolver("billing_rules")
	return &BillingRuntimeResult{
		Cost:         costBreakdownFromSimulation(result),
		MatchedItems: append([]string(nil), result.MatchedRuleIDs...),
		PricingSource: "billing_rules",
		ResolverPath: "billing_rules",
	}
}

func resolveBillingRuntimeOutputModality(input BillingRuntimeInput) string {
	switch {
	case input.ImageCount > 0 || input.MediaType == "image":
		return "image"
	case input.VideoRequests > 0 || input.MediaType == "video":
		return "video"
	default:
		return "text"
	}
}

func (r *BillingRuntimeResolver) resolveLegacyRuntime(input BillingRuntimeInput) (*BillingRuntimeResult, error) {
	if r == nil || r.billingService == nil {
		return &BillingRuntimeResult{Cost: &CostBreakdown{}, PricingSource: "unavailable", ResolverPath: "service_unavailable"}, nil
	}
	var (
		cost *CostBreakdown
		err  error
		path string
	)
	serviceTier := firstNonEmptyBillingRuntime(input.ResolvedServiceTier, input.ServiceTier)
	switch {
	case input.ImageCount > 0 || input.MediaType == "image":
		cost = r.billingService.CalculateImageCostWithServiceTier(input.Model, input.ImageSize, input.ImageCount, input.ImagePriceConfig, input.RateMultiplier, serviceTier)
		path = "legacy_image"
	case input.VideoRequests > 0 || input.MediaType == "video":
		cost = r.billingService.CalculateVideoRequestCost(input.Model, input.RateMultiplier)
		path = "legacy_video"
	case input.LongContextThreshold > 0 && input.LongContextMultiplier > 1:
		cost, err = r.billingService.CalculateCostWithLongContext(input.Model, input.Tokens, input.RateMultiplier, input.LongContextThreshold, input.LongContextMultiplier)
		path = "legacy_long_context"
	case serviceTier != "":
		cost, err = r.billingService.CalculateCostWithServiceTier(input.Model, input.Tokens, input.RateMultiplier, serviceTier)
		path = "legacy_service_tier"
	default:
		cost, err = r.billingService.CalculateCost(input.Model, input.Tokens, input.RateMultiplier)
		path = "legacy_base"
	}
	if err != nil {
		return nil, err
	}
	protocolruntime.RecordBillingResolver(path)
	return &BillingRuntimeResult{
		Cost:          cost,
		PricingSource: "legacy_model_pricing",
		ResolverPath:  path,
	}, nil
}

func firstNonEmptyBillingRuntime(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
