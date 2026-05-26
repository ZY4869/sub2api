package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

type BillingRuntimeInput struct {
	Model                         string
	Provider                      string
	Layer                         string
	PublicCatalogEntryID          string
	PublicCatalogPublicModelID    string
	PublicCatalogSourceAccountID  int64
	PublicCatalogCurrency         string
	PublicCatalogRuntimePriceSpec PublicModelCatalogRuntimePriceSpec
	PublicCatalogSalePriceDisplay PublicModelCatalogPriceDisplay
	InboundEndpoint               string
	RawInboundPath                string
	RequestBody                   []byte
	Tokens                        UsageTokens
	Charges                       BillingSimulationCharges
	ImageCount                    int
	ImageSize                     string
	VideoRequests                 int
	MediaType                     string
	ServiceTier                   string
	RequestedServiceTier          string
	ResolvedServiceTier           string
	BatchMode                     string
	RateMultiplier                float64
	ImagePriceConfig              *ImagePriceConfig
	LongContextThreshold          int
	LongContextMultiplier         float64
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
	if result := r.resolvePublicCatalogEntryRuntime(ctx, input, provider); result != nil {
		return result, nil
	}
	if provider == PlatformGemini || isGeminiBillingEndpoint(input.InboundEndpoint) {
		return r.resolveGeminiRuntime(ctx, input)
	}
	if result := r.resolveRuleBasedRuntime(ctx, input, provider); result != nil {
		return result, nil
	}
	return r.resolveLegacyRuntime(ctx, input)
}

func normalizeBillingRuntimeInput(input BillingRuntimeInput) BillingRuntimeInput {
	input.Model = strings.TrimSpace(input.Model)
	input.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	input.Layer = normalizeBillingDimension(input.Layer, BillingLayerSale)
	input.PublicCatalogEntryID = strings.TrimSpace(input.PublicCatalogEntryID)
	input.PublicCatalogPublicModelID = NormalizeModelCatalogModelID(input.PublicCatalogPublicModelID)
	input.PublicCatalogCurrency = normalizeModelPricingCurrency(input.PublicCatalogCurrency)
	input.PublicCatalogRuntimePriceSpec = normalizePublicModelCatalogRuntimePriceSpec(input.PublicCatalogRuntimePriceSpec)
	input.ServiceTier = normalizeBillingDimension(input.ServiceTier, "")
	input.RequestedServiceTier = normalizeBillingDimension(input.RequestedServiceTier, "")
	input.ResolvedServiceTier = normalizeBillingDimension(input.ResolvedServiceTier, "")
	input.RawInboundPath = strings.TrimSpace(input.RawInboundPath)
	input.BatchMode = normalizeBillingActualBatchMode(input.BatchMode)
	input.MediaType = strings.TrimSpace(strings.ToLower(input.MediaType))
	if input.RateMultiplier <= 0 {
		input.RateMultiplier = 1
	}
	input.PublicCatalogSalePriceDisplay = normalizePublicModelCatalogPriceDisplay(input.PublicCatalogSalePriceDisplay)
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

func (r *BillingRuntimeResolver) resolvePublicCatalogEntryRuntime(ctx context.Context, input BillingRuntimeInput, provider string) *BillingRuntimeResult {
	entryID := strings.TrimSpace(input.PublicCatalogEntryID)
	display := input.PublicCatalogSalePriceDisplay
	if entryID == "" {
		if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
			entryID = entry.EntryID
			display = entry.SalePriceDisplay
			input.PublicCatalogPublicModelID = firstNonEmptyBillingRuntime(input.PublicCatalogPublicModelID, entry.PublicModelID)
			input.PublicCatalogSourceAccountID = firstNonZeroInt64(input.PublicCatalogSourceAccountID, entry.SourceAccountID)
			input.PublicCatalogCurrency = firstNonEmptyBillingRuntime(input.PublicCatalogCurrency, entry.Currency, entry.Item.Currency)
			if input.PublicCatalogRuntimePriceSpec.Currency == "" {
				input.PublicCatalogRuntimePriceSpec = entry.RuntimePriceSpec
			}
		}
	}
	input.PublicCatalogCurrency = firstNonEmptyBillingRuntime(input.PublicCatalogCurrency, input.PublicCatalogRuntimePriceSpec.Currency)
	if entryID == "" || (len(display.Primary) == 0 && len(display.Secondary) == 0) {
		if entryID != "" {
			protocolruntime.RecordBillingResolverFallback("public_catalog_price_empty")
			logger.FromContext(ctx).Warn(
				"public model catalog sale price missing; falling back to legacy pricing",
				zap.String("entry_id", entryID),
				zap.String("model", input.Model),
			)
		}
		return nil
	}
	var classification *GeminiRequestClassification
	if r != nil && r.billingCenterService != nil && (provider == PlatformGemini || isGeminiBillingEndpoint(input.InboundEndpoint)) {
		calcInput := GeminiBillingCalculationInput{
			Model:                input.Model,
			InboundEndpoint:      input.InboundEndpoint,
			RawInboundPath:       input.RawInboundPath,
			RequestBody:          input.RequestBody,
			Tokens:               input.Tokens,
			ImageCount:           input.ImageCount,
			VideoRequests:        input.VideoRequests,
			MediaType:            input.MediaType,
			RateMultiplier:       input.RateMultiplier,
			RequestedServiceTier: input.RequestedServiceTier,
			ResolvedServiceTier:  firstNonEmptyBillingRuntime(input.ResolvedServiceTier, input.ServiceTier),
			Charges:              input.Charges,
		}
		classification = r.billingCenterService.classifier.ClassifyRequest(calcInput)
		input.Charges = r.billingCenterService.buildGeminiCalculationCharges(calcInput, classification)
		input.BatchMode = firstNonEmptyBillingRuntime(classification.BatchMode, input.BatchMode)
		input.ServiceTier = firstNonEmptyBillingRuntime(classification.ServiceTier, input.ServiceTier)
	}
	cost := calculatePublicCatalogEntryRuntimeCost(display, input)
	if cost == nil {
		protocolruntime.RecordBillingResolverFallback("public_catalog_price_empty")
		logger.FromContext(ctx).Warn(
			"public model catalog sale price produced zero cost; falling back to legacy pricing",
			zap.String("entry_id", entryID),
			zap.String("model", input.Model),
		)
		return nil
	}
	protocolruntime.RecordBillingResolver("public_catalog_entry")
	logger.FromContext(ctx).Info(
		"public model catalog sale price matched",
		zap.String("entry_id", entryID),
		zap.String("public_model_id", input.PublicCatalogPublicModelID),
		zap.Int64("source_account_id", input.PublicCatalogSourceAccountID),
		zap.String("pricing_source", "public_catalog_entry"),
	)
	return &BillingRuntimeResult{
		Cost:           cost,
		Classification: classification,
		MatchedItems:   []string{entryID},
		PricingSource:  "public_catalog_entry",
		ResolverPath:   "public_catalog_entry",
	}
}

func calculatePublicCatalogEntryRuntimeCost(display PublicModelCatalogPriceDisplay, input BillingRuntimeInput) *CostBreakdown {
	priceByID := map[string]float64{}
	for _, entry := range append(append([]PublicModelCatalogPriceEntry(nil), display.Primary...), display.Secondary...) {
		if id := strings.TrimSpace(entry.ID); id != "" {
			priceByID[id] = entry.Value
		}
	}
	if len(priceByID) == 0 {
		return nil
	}
	cost := &CostBreakdown{}
	matched := false
	missing := false
	add := func(target *float64, fieldID string, units float64) {
		if units <= 0 {
			return
		}
		price, ok := priceByID[fieldID]
		if !ok {
			missing = true
			return
		}
		matched = true
		*target += price * units * input.RateMultiplier
	}
	switch {
	case input.ImageCount > 0 || input.MediaType == "image":
		add(&cost.OutputCost, publicCatalogOutputFieldID(input), float64(input.ImageCount))
	case input.VideoRequests > 0 || input.MediaType == "video":
		add(&cost.OutputCost, publicCatalogOutputFieldID(input), float64(input.VideoRequests))
	default:
		inputField, outputField, cacheField := publicCatalogTextFieldIDs(input)
		add(&cost.InputCost, inputField, float64(input.Tokens.InputTokens))
		add(&cost.OutputCost, outputField, float64(input.Tokens.OutputTokens))
		add(&cost.CacheCreationCost, cacheField, float64(input.Tokens.CacheCreationTokens+input.Tokens.CacheCreation5mTokens))
		add(&cost.CacheCreationCost, cacheField, float64(input.Tokens.CacheCreation1hTokens))
		add(&cost.CacheReadCost, cacheField, float64(input.Tokens.CacheReadTokens))
		add(&cost.OutputCost, billingDiscountFieldGroundingSearch, input.Charges.GroundingSearchQueries)
		add(&cost.OutputCost, billingDiscountFieldGroundingMaps, input.Charges.GroundingMapsQueries)
		add(&cost.OutputCost, billingDiscountFieldFileSearchEmbedding, input.Charges.FileSearchEmbeddingTokens)
		add(&cost.OutputCost, billingDiscountFieldFileSearchRetrieval, input.Charges.FileSearchRetrievalTokens)
	}
	cost.TotalCost = cost.InputCost + cost.OutputCost + cost.CacheCreationCost + cost.CacheReadCost
	cost.ActualCost = cost.TotalCost
	if missing || !matched || cost.TotalCost == 0 {
		return nil
	}
	return finalizeCostBreakdownCurrency(cost, &ModelPricing{Currency: defaultModelPricingCurrency(input.PublicCatalogCurrency)})
}

func publicCatalogTextFieldIDs(input BillingRuntimeInput) (string, string, string) {
	if normalizeBillingActualBatchMode(input.BatchMode) == BillingBatchModeBatch {
		return billingDiscountFieldBatchInputPrice, billingDiscountFieldBatchOutputPrice, billingDiscountFieldBatchCachePrice
	}
	spec := normalizePublicModelCatalogRuntimePriceSpec(input.PublicCatalogRuntimePriceSpec)
	longContext := spec.LongContextInputTokenThreshold > 0 &&
		(input.Tokens.InputTokens+input.Tokens.CacheReadTokens) > spec.LongContextInputTokenThreshold
	inputField := billingDiscountFieldInputPrice
	outputField := billingDiscountFieldOutputPrice
	if longContext && spec.LongContextInputCostMultiplier > 1 {
		inputField = billingDiscountFieldInputPriceAboveThreshold
	}
	if longContext && spec.LongContextOutputCostMultiplier > 1 {
		outputField = billingDiscountFieldOutputPriceAboveThreshold
	}
	return inputField, outputField, billingDiscountFieldCachePrice
}

func publicCatalogOutputFieldID(input BillingRuntimeInput) string {
	if normalizeBillingActualBatchMode(input.BatchMode) == BillingBatchModeBatch {
		return billingDiscountFieldBatchOutputPrice
	}
	return billingDiscountFieldOutputPrice
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
		return r.resolveLegacyRuntime(ctx, input)
	}
	result, err := r.billingCenterService.CalculateGeminiCost(ctx, GeminiBillingCalculationInput{
		Model:                input.Model,
		InboundEndpoint:      input.InboundEndpoint,
		RawInboundPath:       input.RawInboundPath,
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
		Cost:          costBreakdownFromSimulationWithMetadata(result, r.billingCenterService.resolveModelPricingCurrencyMetadata(ctx, input.Model, input.Layer)),
		MatchedItems:  append([]string(nil), result.MatchedRuleIDs...),
		PricingSource: "billing_rules",
		ResolverPath:  "billing_rules",
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

func (r *BillingRuntimeResolver) resolveLegacyRuntime(ctx context.Context, input BillingRuntimeInput) (*BillingRuntimeResult, error) {
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
		cost = r.billingService.CalculateImageCostWithServiceTierWithContext(ctx, input.Model, input.ImageSize, input.ImageCount, input.ImagePriceConfig, input.RateMultiplier, serviceTier)
		path = "legacy_image"
	case input.VideoRequests > 0 || input.MediaType == "video":
		cost = r.billingService.CalculateVideoRequestCostWithContext(ctx, input.Model, input.RateMultiplier)
		path = "legacy_video"
	case input.LongContextThreshold > 0 && input.LongContextMultiplier > 1:
		cost, err = r.billingService.CalculateCostWithLongContextWithContext(ctx, input.Model, input.Tokens, input.RateMultiplier, input.LongContextThreshold, input.LongContextMultiplier)
		path = "legacy_long_context"
	case serviceTier != "":
		cost, err = r.billingService.CalculateCostWithServiceTierWithContext(ctx, input.Model, input.Tokens, input.RateMultiplier, serviceTier)
		path = "legacy_service_tier"
	default:
		cost, err = r.billingService.CalculateCostWithServiceTierWithContext(ctx, input.Model, input.Tokens, input.RateMultiplier, "")
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

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
