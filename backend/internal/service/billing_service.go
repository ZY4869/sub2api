package service

import (
	"context"
	"fmt"

	"log"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// APIKeyRateLimitCacheData holds rate limit usage data cached in Redis.
type APIKeyRateLimitCacheData struct {
	Usage5h  float64 `json:"usage_5h"`
	Usage1d  float64 `json:"usage_1d"`
	Usage7d  float64 `json:"usage_7d"`
	Window5h int64   `json:"window_5h"` // unix timestamp, 0 = not started
	Window1d int64   `json:"window_1d"`
	Window7d int64   `json:"window_7d"`
}

// BillingCache defines cache operations for billing service
type BillingCache interface {
	// Balance operations
	GetUserBalance(ctx context.Context, userID int64) (float64, error)
	SetUserBalance(ctx context.Context, userID int64, balance float64) error
	DeductUserBalance(ctx context.Context, userID int64, amount float64) error
	InvalidateUserBalance(ctx context.Context, userID int64) error

	// Subscription operations
	GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error)
	SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error
	UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error
	InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error

	// API Key rate limit operations
	GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error)
	SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error
	UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error
	InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error
}

// ModelPricing defines per-token pricing, aligned with LiteLLM semantics.
type ModelPricing struct {
	InputPricePerToken                        float64
	InputPricePerTokenPriority                float64
	InputTokenThreshold                       int
	InputPricePerTokenAboveThreshold          float64
	InputPricePerTokenPriorityAboveThreshold  float64
	OutputPricePerToken                       float64
	OutputPricePerTokenPriority               float64
	OutputTokenThreshold                      int
	OutputPricePerTokenAboveThreshold         float64
	OutputPricePerTokenPriorityAboveThreshold float64
	OutputPricePerImage                       float64
	OutputPricePerImagePriority               float64
	OutputPricePerVideoRequest                float64
	CacheCreationPricePerToken                float64
	CacheReadPricePerToken                    float64
	CacheReadPricePerTokenPriority            float64
	CacheCreation5mPrice                      float64
	CacheCreation1hPrice                      float64
	SupportsCacheBreakdown                    bool
	LongContextInputThreshold                 int
	LongContextInputMultiplier                float64
	LongContextOutputMultiplier               float64
}

const (
	openAIGPT54LongContextInputThreshold   = 272000
	openAIGPT54LongContextInputMultiplier  = 2.0
	openAIGPT54LongContextOutputMultiplier = 1.5
)

func normalizeBillingServiceTier(serviceTier string) string {
	return strings.ToLower(strings.TrimSpace(serviceTier))
}

func usePriorityServiceTierPricing(serviceTier string, pricing *ModelPricing) bool {
	if pricing == nil || normalizeBillingServiceTier(serviceTier) != "priority" {
		return false
	}
	return pricing.InputPricePerTokenPriority > 0 || pricing.OutputPricePerTokenPriority > 0 || pricing.CacheReadPricePerTokenPriority > 0
}

func serviceTierCostMultiplier(serviceTier string) float64 {
	switch normalizeBillingServiceTier(serviceTier) {
	case "priority":
		return 2.0
	case "flex":
		return 0.5
	default:
		return 1.0
	}
}

func resolveTieredTokenPrice(tokenCount int, lowPrice float64, threshold int, highPrice float64) float64 {
	if threshold <= 0 || highPrice <= 0 || tokenCount <= threshold {
		return lowPrice
	}
	return highPrice
}

// UsageTokens captures token counts used for billing.
type UsageTokens struct {
	InputTokens           int
	OutputTokens          int
	CacheCreationTokens   int
	CacheReadTokens       int
	CacheCreation5mTokens int
	CacheCreation1hTokens int
}

// CostBreakdown contains the computed billing amounts.
type CostBreakdown struct {
	InputCost         float64
	OutputCost        float64
	CacheCreationCost float64
	CacheReadCost     float64
	TotalCost         float64
	ActualCost        float64 // Final billed amount after multipliers are applied.
}
// BillingService provides billing and pricing operations.
type BillingService struct {
	cfg                    *config.Config
	pricingService         *PricingService
	modelRegistryService   *ModelRegistryService
	billingCenterService   *BillingCenterService
	fallbackPrices         map[string]*ModelPricing // fallback pricing table
	overrideMu             sync.RWMutex
	officialPriceOverrides map[string]*ModelPricingOverride
	priceOverrides         map[string]*ModelPricingOverride
}

// NewBillingService creates a billing service instance.
func NewBillingService(cfg *config.Config, pricingService *PricingService) *BillingService {
	s := &BillingService{
		cfg:                    cfg,
		pricingService:         pricingService,
		fallbackPrices:         make(map[string]*ModelPricing),
		officialPriceOverrides: make(map[string]*ModelPricingOverride),
		priceOverrides:         make(map[string]*ModelPricingOverride),
	}

	// Initialize hardcoded fallback pricing when dynamic pricing is unavailable.
	s.initFallbackPricing()

	return s
}

func (s *BillingService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
}

func (s *BillingService) SetBillingCenterService(billingCenterService *BillingCenterService) {
	s.billingCenterService = billingCenterService
}

// initFallbackPricing initializes hardcoded fallback pricing.
// Price unit: USD per token, aligned with LiteLLM semantics.
func (s *BillingService) initFallbackPricing() {
	// Claude 4.5 Opus
	s.fallbackPrices["claude-opus-4.5"] = &ModelPricing{
		InputPricePerToken:         5e-6,    // $5 per MTok
		OutputPricePerToken:        25e-6,   // $25 per MTok
		CacheCreationPricePerToken: 6.25e-6, // $6.25 per MTok
		CacheReadPricePerToken:     0.5e-6,  // $0.50 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 4 Sonnet
	s.fallbackPrices["claude-sonnet-4"] = &ModelPricing{
		InputPricePerToken:         3e-6,    // $3 per MTok
		OutputPricePerToken:        15e-6,   // $15 per MTok
		CacheCreationPricePerToken: 3.75e-6, // $3.75 per MTok
		CacheReadPricePerToken:     0.3e-6,  // $0.30 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3.5 Sonnet
	s.fallbackPrices["claude-3-5-sonnet"] = &ModelPricing{
		InputPricePerToken:         3e-6,    // $3 per MTok
		OutputPricePerToken:        15e-6,   // $15 per MTok
		CacheCreationPricePerToken: 3.75e-6, // $3.75 per MTok
		CacheReadPricePerToken:     0.3e-6,  // $0.30 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3.5 Haiku
	s.fallbackPrices["claude-3-5-haiku"] = &ModelPricing{
		InputPricePerToken:         1e-6,    // $1 per MTok
		OutputPricePerToken:        5e-6,    // $5 per MTok
		CacheCreationPricePerToken: 1.25e-6, // $1.25 per MTok
		CacheReadPricePerToken:     0.1e-6,  // $0.10 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3 Opus
	s.fallbackPrices["claude-3-opus"] = &ModelPricing{
		InputPricePerToken:         15e-6,    // $15 per MTok
		OutputPricePerToken:        75e-6,    // $75 per MTok
		CacheCreationPricePerToken: 18.75e-6, // $18.75 per MTok
		CacheReadPricePerToken:     1.5e-6,   // $1.50 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3 Haiku
	s.fallbackPrices["claude-3-haiku"] = &ModelPricing{
		InputPricePerToken:         0.25e-6, // $0.25 per MTok
		OutputPricePerToken:        1.25e-6, // $1.25 per MTok
		CacheCreationPricePerToken: 0.3e-6,  // $0.30 per MTok
		CacheReadPricePerToken:     0.03e-6, // $0.03 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 4.6 Opus (currently priced the same as 4.5)
	s.fallbackPrices["claude-opus-4.6"] = s.fallbackPrices["claude-opus-4.5"]

	// Claude 4.7 Opus (暂与4.6同价，待官方定价更新)
	s.fallbackPrices["claude-opus-4.7"] = s.fallbackPrices["claude-opus-4.6"]

	// Gemini 3.1 Pro
	s.fallbackPrices["gemini-3.1-pro"] = &ModelPricing{
		InputPricePerToken:         2e-6,   // $2 per MTok
		OutputPricePerToken:        12e-6,  // $12 per MTok
		CacheCreationPricePerToken: 2e-6,   // $2 per MTok
		CacheReadPricePerToken:     0.2e-6, // $0.20 per MTok
		SupportsCacheBreakdown:     false,
	}

	// OpenAI GPT-5.1 local fallback to avoid billing failures when dynamic pricing is unavailable.
	s.fallbackPrices["gpt-5.1"] = &ModelPricing{
		InputPricePerToken:             1.25e-6, // $1.25 per MTok
		InputPricePerTokenPriority:     2.5e-6,  // $2.5 per MTok
		OutputPricePerToken:            10e-6,   // $10 per MTok
		OutputPricePerTokenPriority:    20e-6,   // $20 per MTok
		CacheCreationPricePerToken:     1.25e-6, // $1.25 per MTok
		CacheReadPricePerToken:         0.125e-6,
		CacheReadPricePerTokenPriority: 0.25e-6,
		SupportsCacheBreakdown:         false,
	}
	// OpenAI GPT-5.4 business baseline pricing.
	s.fallbackPrices["gpt-5.4"] = &ModelPricing{
		InputPricePerToken:             2.5e-6,  // $2.5 per MTok
		InputPricePerTokenPriority:     5e-6,    // $5 per MTok
		OutputPricePerToken:            15e-6,   // $15 per MTok
		OutputPricePerTokenPriority:    30e-6,   // $30 per MTok
		CacheCreationPricePerToken:     2.5e-6,  // $2.5 per MTok
		CacheReadPricePerToken:         0.25e-6, // $0.25 per MTok
		CacheReadPricePerTokenPriority: 0.5e-6,  // $0.5 per MTok
		SupportsCacheBreakdown:         false,
		LongContextInputThreshold:      openAIGPT54LongContextInputThreshold,
		LongContextInputMultiplier:     openAIGPT54LongContextInputMultiplier,
		LongContextOutputMultiplier:    openAIGPT54LongContextOutputMultiplier,
	}
	// OpenAI GPT-5.4 mini/nano/pro official fallback pricing.
	s.fallbackPrices["gpt-5.4-mini"] = &ModelPricing{
		InputPricePerToken:     7.5e-7,
		OutputPricePerToken:    4.5e-6,
		CacheReadPricePerToken: 7.5e-8,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["gpt-5.4-nano"] = &ModelPricing{
		InputPricePerToken:     2e-7,
		OutputPricePerToken:    1.25e-6,
		CacheReadPricePerToken: 2e-8,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["gpt-5.4-pro"] = &ModelPricing{
		InputPricePerToken:                3e-5, // $30 per MTok
		InputTokenThreshold:               openAIGPT54LongContextInputThreshold,
		InputPricePerTokenAboveThreshold:  6e-5,
		OutputPricePerToken:               1.8e-4, // $180 per MTok
		OutputTokenThreshold:              openAIGPT54LongContextInputThreshold,
		OutputPricePerTokenAboveThreshold: 2.7e-4,
		SupportsCacheBreakdown:            false,
		LongContextInputThreshold:         openAIGPT54LongContextInputThreshold,
		LongContextInputMultiplier:        openAIGPT54LongContextInputMultiplier,
		LongContextOutputMultiplier:       openAIGPT54LongContextOutputMultiplier,
	}
	// OpenAI GPT-5.2 local fallback pricing.
	s.fallbackPrices["gpt-5.2"] = &ModelPricing{
		InputPricePerToken:             1.75e-6,
		InputPricePerTokenPriority:     3.5e-6,
		OutputPricePerToken:            14e-6,
		OutputPricePerTokenPriority:    28e-6,
		CacheCreationPricePerToken:     1.75e-6,
		CacheReadPricePerToken:         0.175e-6,
		CacheReadPricePerTokenPriority: 0.35e-6,
		SupportsCacheBreakdown:         false,
	}
	// Codex family fallback pricing uses GPT-5.1 Codex as the baseline.
	s.fallbackPrices["gpt-5.1-codex"] = &ModelPricing{
		InputPricePerToken:             1.5e-6, // $1.5 per MTok
		InputPricePerTokenPriority:     3e-6,   // $3 per MTok
		OutputPricePerToken:            12e-6,  // $12 per MTok
		OutputPricePerTokenPriority:    24e-6,  // $24 per MTok
		CacheCreationPricePerToken:     1.5e-6, // $1.5 per MTok
		CacheReadPricePerToken:         0.15e-6,
		CacheReadPricePerTokenPriority: 0.3e-6,
		SupportsCacheBreakdown:         false,
	}
	s.fallbackPrices["gpt-5.2-codex"] = &ModelPricing{
		InputPricePerToken:             1.75e-6,
		InputPricePerTokenPriority:     3.5e-6,
		OutputPricePerToken:            14e-6,
		OutputPricePerTokenPriority:    28e-6,
		CacheCreationPricePerToken:     1.75e-6,
		CacheReadPricePerToken:         0.175e-6,
		CacheReadPricePerTokenPriority: 0.35e-6,
		SupportsCacheBreakdown:         false,
	}
	s.fallbackPrices["gpt-5.3-codex"] = s.fallbackPrices["gpt-5.1-codex"]
}

// getFallbackPricing returns fallback pricing by model family.
func (s *BillingService) getFallbackPricing(model string) *ModelPricing {
	modelLower := strings.ToLower(model)

	// Match by model family.
	if strings.Contains(modelLower, "opus") {
		if strings.Contains(modelLower, "4.7") || strings.Contains(modelLower, "4-7") {
			return s.fallbackPrices["claude-opus-4.7"]
		}
		if strings.Contains(modelLower, "4.6") || strings.Contains(modelLower, "4-6") {
			return s.fallbackPrices["claude-opus-4.6"]
		}
		if strings.Contains(modelLower, "4.5") || strings.Contains(modelLower, "4-5") {
			return s.fallbackPrices["claude-opus-4.5"]
		}
		return s.fallbackPrices["claude-3-opus"]
	}
	if strings.Contains(modelLower, "sonnet") {
		if strings.Contains(modelLower, "4") && !strings.Contains(modelLower, "3") {
			return s.fallbackPrices["claude-sonnet-4"]
		}
		return s.fallbackPrices["claude-3-5-sonnet"]
	}
	if strings.Contains(modelLower, "haiku") {
		if strings.Contains(modelLower, "3-5") || strings.Contains(modelLower, "3.5") {
			return s.fallbackPrices["claude-3-5-haiku"]
		}
		return s.fallbackPrices["claude-3-haiku"]
	}
	// Unknown Claude models fall back to Sonnet to avoid billing interruptions.
	if strings.Contains(modelLower, "claude") {
		return s.fallbackPrices["claude-sonnet-4"]
	}
	if strings.Contains(modelLower, "gemini-3.1-pro") || strings.Contains(modelLower, "gemini-3-1-pro") {
		return s.fallbackPrices["gemini-3.1-pro"]
	}

	// Only match known GPT-5/Codex families to avoid mispricing unknown OpenAI models.
	if strings.Contains(modelLower, "gpt-5") || strings.Contains(modelLower, "codex") {
		normalized := normalizeCodexModel(modelLower)
		switch {
		case strings.HasPrefix(normalized, "gpt-5.4-pro"):
			return s.fallbackPrices["gpt-5.4-pro"]
		case strings.HasPrefix(normalized, "gpt-5.4-mini"):
			return s.fallbackPrices["gpt-5.4-mini"]
		case strings.HasPrefix(normalized, "gpt-5.4-nano"):
			return s.fallbackPrices["gpt-5.4-nano"]
		case strings.HasPrefix(normalized, "gpt-5.4"):
			return s.fallbackPrices["gpt-5.4"]
		case strings.HasPrefix(normalized, "gpt-5.2-codex"):
			return s.fallbackPrices["gpt-5.2-codex"]
		case strings.HasPrefix(normalized, "gpt-5.2"):
			return s.fallbackPrices["gpt-5.2"]
		case strings.HasPrefix(normalized, "gpt-5.3-codex"):
			return s.fallbackPrices["gpt-5.3-codex"]
		case normalized == "gpt-5.1-codex", normalized == "gpt-5.1-codex-max", normalized == "gpt-5.1-codex-mini", normalized == "codex-mini-latest":
			return s.fallbackPrices["gpt-5.1-codex"]
		case strings.HasPrefix(normalized, "gpt-5.1"), strings.HasPrefix(normalized, "gpt-5-pro"):
			return s.fallbackPrices["gpt-5.1"]
		}
	}

	return nil
}

// GetModelPricing returns pricing for a model.
func (s *BillingService) GetModelPricing(model string) (*ModelPricing, error) {
	// Normalize the model name to lowercase.
	model = strings.ToLower(model)
	if s.modelRegistryService != nil {
		if pricingModel, ok, err := s.modelRegistryService.ResolvePricingModel(context.Background(), model); err == nil && ok && pricingModel != "" {
			model = pricingModel
		}
	}

	// 1. Prefer dynamic pricing when available.
	if s.pricingService != nil {
		litellmPricing := s.pricingService.GetModelPricing(model)
		if litellmPricing != nil {
			// Enable 5m/1h cache breakdown only when:
			// 1. 1h pricing exists.
			// 2. 1h pricing is greater than 5m pricing to avoid under-charging on bad upstream data.
			price5m := litellmPricing.CacheCreationInputTokenCost
			price1h := litellmPricing.CacheCreationInputTokenCostAbove1hr
			enableBreakdown := price1h > 0 && price1h > price5m
			return s.applyModelSpecificPricingPolicy(model, &ModelPricing{
				InputPricePerToken:                        litellmPricing.InputCostPerToken,
				InputPricePerTokenPriority:                litellmPricing.InputCostPerTokenPriority,
				InputTokenThreshold:                       litellmPricing.InputTokenThreshold,
				InputPricePerTokenAboveThreshold:          litellmPricing.InputCostPerTokenAboveThreshold,
				InputPricePerTokenPriorityAboveThreshold:  litellmPricing.InputCostPerTokenPriorityAboveThreshold,
				OutputPricePerToken:                       litellmPricing.OutputCostPerToken,
				OutputPricePerTokenPriority:               litellmPricing.OutputCostPerTokenPriority,
				OutputTokenThreshold:                      litellmPricing.OutputTokenThreshold,
				OutputPricePerTokenAboveThreshold:         litellmPricing.OutputCostPerTokenAboveThreshold,
				OutputPricePerTokenPriorityAboveThreshold: litellmPricing.OutputCostPerTokenPriorityAboveThreshold,
				OutputPricePerImage:                       litellmPricing.OutputCostPerImage,
				OutputPricePerImagePriority:               litellmPricing.OutputCostPerImagePriority,
				OutputPricePerVideoRequest:                litellmPricing.OutputCostPerVideoRequest,
				CacheCreationPricePerToken:                litellmPricing.CacheCreationInputTokenCost,
				CacheReadPricePerToken:                    litellmPricing.CacheReadInputTokenCost,
				CacheReadPricePerTokenPriority:            litellmPricing.CacheReadInputTokenCostPriority,
				CacheCreation5mPrice:                      price5m,
				CacheCreation1hPrice:                      price1h,
				SupportsCacheBreakdown:                    enableBreakdown,
				LongContextInputThreshold:                 litellmPricing.LongContextInputTokenThreshold,
				LongContextInputMultiplier:                litellmPricing.LongContextInputCostMultiplier,
				LongContextOutputMultiplier:               litellmPricing.LongContextOutputCostMultiplier,
			}), nil
		}
	}

	// 2. Fall back to hardcoded pricing.
	fallback := s.getFallbackPricing(model)
	if fallback != nil {
		log.Printf("[Billing] Using fallback pricing for model: %s", model)
		return s.applyModelSpecificPricingPolicy(model, fallback), nil
	}

	return nil, fmt.Errorf("pricing not found for model: %s", model)
}

func (s *BillingService) ReplaceModelPriceOverrides(overrides map[string]*ModelPricingOverride) {
	normalized := make(map[string]*ModelPricingOverride, len(overrides))
	for model, override := range overrides {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" || override == nil || pricingEmpty(&override.ModelCatalogPricing) {
			continue
		}
		normalized[key] = cloneModelPricingOverride(override)
	}
	s.overrideMu.Lock()
	s.priceOverrides = normalized
	s.overrideMu.Unlock()
}

func (s *BillingService) ReplaceModelOfficialPriceOverrides(overrides map[string]*ModelPricingOverride) {
	normalized := make(map[string]*ModelPricingOverride, len(overrides))
	for model, override := range overrides {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" || override == nil || pricingEmpty(&override.ModelCatalogPricing) {
			continue
		}
		normalized[key] = cloneModelPricingOverride(override)
	}
	s.overrideMu.Lock()
	s.officialPriceOverrides = normalized
	s.overrideMu.Unlock()
}

func (s *BillingService) getModelOfficialPriceOverride(model string) *ModelPricingOverride {
	key := CanonicalizeModelNameForPricing(model)
	if key == "" {
		return nil
	}
	s.overrideMu.RLock()
	override := s.officialPriceOverrides[key]
	s.overrideMu.RUnlock()
	return cloneModelPricingOverride(override)
}

func (s *BillingService) getModelPriceOverride(model string) *ModelPricingOverride {
	key := CanonicalizeModelNameForPricing(model)
	if key == "" {
		return nil
	}
	s.overrideMu.RLock()
	override := s.priceOverrides[key]
	s.overrideMu.RUnlock()
	return cloneModelPricingOverride(override)
}

func applyModelPricingOverride(pricing *ModelPricing, override *ModelPricingOverride) *ModelPricing {
	if pricing == nil || override == nil {
		return pricing
	}
	cloned := *pricing
	if override.InputCostPerToken != nil {
		cloned.InputPricePerToken = *override.InputCostPerToken
	}
	if override.InputCostPerTokenPriority != nil {
		cloned.InputPricePerTokenPriority = *override.InputCostPerTokenPriority
	}
	if override.InputTokenThreshold != nil {
		cloned.InputTokenThreshold = *override.InputTokenThreshold
	}
	if override.InputCostPerTokenAboveThreshold != nil {
		cloned.InputPricePerTokenAboveThreshold = *override.InputCostPerTokenAboveThreshold
	}
	if override.InputCostPerTokenPriorityAboveThreshold != nil {
		cloned.InputPricePerTokenPriorityAboveThreshold = *override.InputCostPerTokenPriorityAboveThreshold
	}
	if override.OutputCostPerToken != nil {
		cloned.OutputPricePerToken = *override.OutputCostPerToken
	}
	if override.OutputCostPerTokenPriority != nil {
		cloned.OutputPricePerTokenPriority = *override.OutputCostPerTokenPriority
	}
	if override.OutputTokenThreshold != nil {
		cloned.OutputTokenThreshold = *override.OutputTokenThreshold
	}
	if override.OutputCostPerTokenAboveThreshold != nil {
		cloned.OutputPricePerTokenAboveThreshold = *override.OutputCostPerTokenAboveThreshold
	}
	if override.OutputCostPerTokenPriorityAboveThreshold != nil {
		cloned.OutputPricePerTokenPriorityAboveThreshold = *override.OutputCostPerTokenPriorityAboveThreshold
	}
	if override.CacheCreationInputTokenCost != nil {
		cloned.CacheCreationPricePerToken = *override.CacheCreationInputTokenCost
	}
	if override.CacheCreationInputTokenCostAbove1hr != nil {
		cloned.CacheCreation1hPrice = *override.CacheCreationInputTokenCostAbove1hr
	}
	if override.CacheReadInputTokenCost != nil {
		cloned.CacheReadPricePerToken = *override.CacheReadInputTokenCost
	}
	if override.CacheReadInputTokenCostPriority != nil {
		cloned.CacheReadPricePerTokenPriority = *override.CacheReadInputTokenCostPriority
	}
	if override.OutputCostPerImage != nil {
		cloned.OutputPricePerImage = *override.OutputCostPerImage
	}
	if override.OutputCostPerImagePriority != nil {
		cloned.OutputPricePerImagePriority = *override.OutputCostPerImagePriority
	}
	if override.OutputCostPerVideoRequest != nil {
		cloned.OutputPricePerVideoRequest = *override.OutputCostPerVideoRequest
	}
	return &cloned
}

func (s *BillingService) getPricingForBilling(model string) (*ModelPricing, error) {
	pricing, err := s.GetModelPricing(model)
	if err != nil {
		return nil, err
	}
	pricing = applyModelPricingOverride(pricing, s.getModelOfficialPriceOverride(model))
	pricing = applyModelPricingOverride(pricing, s.getModelPriceOverride(model))
	return pricing, nil
}

// CalculateCost computes billing using the model pricing table.
func (s *BillingService) CalculateCost(model string, tokens UsageTokens, rateMultiplier float64) (*CostBreakdown, error) {
	return s.CalculateCostWithServiceTier(model, tokens, rateMultiplier, "")
}

func (s *BillingService) CalculateCostWithServiceTier(model string, tokens UsageTokens, rateMultiplier float64, serviceTier string) (*CostBreakdown, error) {
	pricing, err := s.getPricingForBilling(model)
	if err != nil {
		return nil, err
	}
	return s.calculateCostWithPricing(pricing, tokens, tokens, rateMultiplier, serviceTier), nil
}

func (s *BillingService) calculateCostWithPricing(
	pricing *ModelPricing,
	billedTokens UsageTokens,
	thresholdTokens UsageTokens,
	rateMultiplier float64,
	serviceTier string,
) *CostBreakdown {
	breakdown := &CostBreakdown{}
	inputPricePerToken := pricing.InputPricePerToken
	outputPricePerToken := pricing.OutputPricePerToken
	cacheReadPricePerToken := pricing.CacheReadPricePerToken
	tierMultiplier := 1.0
	usingPriorityPricing := usePriorityServiceTierPricing(serviceTier, pricing)
	if usingPriorityPricing {
		if pricing.InputPricePerTokenPriority > 0 {
			inputPricePerToken = pricing.InputPricePerTokenPriority
		}
		if pricing.OutputPricePerTokenPriority > 0 {
			outputPricePerToken = pricing.OutputPricePerTokenPriority
		}
		if pricing.CacheReadPricePerTokenPriority > 0 {
			cacheReadPricePerToken = pricing.CacheReadPricePerTokenPriority
		}
	} else {
		tierMultiplier = serviceTierCostMultiplier(serviceTier)
	}

	if usingPriorityPricing {
		inputPricePerToken = resolveTieredTokenPrice(thresholdTokens.InputTokens, inputPricePerToken, pricing.InputTokenThreshold, pricing.InputPricePerTokenPriorityAboveThreshold)
		outputPricePerToken = resolveTieredTokenPrice(thresholdTokens.OutputTokens, outputPricePerToken, pricing.OutputTokenThreshold, pricing.OutputPricePerTokenPriorityAboveThreshold)
	} else {
		inputPricePerToken = resolveTieredTokenPrice(thresholdTokens.InputTokens, inputPricePerToken, pricing.InputTokenThreshold, pricing.InputPricePerTokenAboveThreshold)
		outputPricePerToken = resolveTieredTokenPrice(thresholdTokens.OutputTokens, outputPricePerToken, pricing.OutputTokenThreshold, pricing.OutputPricePerTokenAboveThreshold)
	}

	if s.shouldApplySessionLongContextPricing(thresholdTokens, pricing) {
		inputPricePerToken *= pricing.LongContextInputMultiplier
		outputPricePerToken *= pricing.LongContextOutputMultiplier
	}

	breakdown.InputCost = float64(billedTokens.InputTokens) * inputPricePerToken
	breakdown.OutputCost = float64(billedTokens.OutputTokens) * outputPricePerToken

	if pricing.SupportsCacheBreakdown && (pricing.CacheCreation5mPrice > 0 || pricing.CacheCreation1hPrice > 0) {
		if billedTokens.CacheCreation5mTokens == 0 && billedTokens.CacheCreation1hTokens == 0 && billedTokens.CacheCreationTokens > 0 {
			breakdown.CacheCreationCost = float64(billedTokens.CacheCreationTokens) * pricing.CacheCreation5mPrice
		} else {
			breakdown.CacheCreationCost = float64(billedTokens.CacheCreation5mTokens)*pricing.CacheCreation5mPrice +
				float64(billedTokens.CacheCreation1hTokens)*pricing.CacheCreation1hPrice
		}
	} else {
		breakdown.CacheCreationCost = float64(billedTokens.CacheCreationTokens) * pricing.CacheCreationPricePerToken
	}

	breakdown.CacheReadCost = float64(billedTokens.CacheReadTokens) * cacheReadPricePerToken

	if tierMultiplier != 1.0 {
		breakdown.InputCost *= tierMultiplier
		breakdown.OutputCost *= tierMultiplier
		breakdown.CacheCreationCost *= tierMultiplier
		breakdown.CacheReadCost *= tierMultiplier
	}

	breakdown.TotalCost = breakdown.InputCost + breakdown.OutputCost +
		breakdown.CacheCreationCost + breakdown.CacheReadCost
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	breakdown.ActualCost = breakdown.TotalCost * rateMultiplier
	return breakdown
}

func (s *BillingService) applyModelSpecificPricingPolicy(model string, pricing *ModelPricing) *ModelPricing {
	if pricing == nil {
		return nil
	}
	if !isOpenAIGPT54Model(model) {
		return pricing
	}
	if pricing.LongContextInputThreshold > 0 && pricing.LongContextInputMultiplier > 0 && pricing.LongContextOutputMultiplier > 0 {
		return pricing
	}
	cloned := *pricing
	if cloned.LongContextInputThreshold <= 0 {
		cloned.LongContextInputThreshold = openAIGPT54LongContextInputThreshold
	}
	if cloned.LongContextInputMultiplier <= 0 {
		cloned.LongContextInputMultiplier = openAIGPT54LongContextInputMultiplier
	}
	if cloned.LongContextOutputMultiplier <= 0 {
		cloned.LongContextOutputMultiplier = openAIGPT54LongContextOutputMultiplier
	}
	return &cloned
}

func (s *BillingService) shouldApplySessionLongContextPricing(tokens UsageTokens, pricing *ModelPricing) bool {
	if pricing == nil || pricing.LongContextInputThreshold <= 0 {
		return false
	}
	if pricing.LongContextInputMultiplier <= 1 && pricing.LongContextOutputMultiplier <= 1 {
		return false
	}
	totalInputTokens := tokens.InputTokens + tokens.CacheReadTokens
	return totalInputTokens > pricing.LongContextInputThreshold
}

func isOpenAIGPT54Model(model string) bool {
	normalized := normalizeCodexModel(strings.TrimSpace(strings.ToLower(model)))
	base := modelDateVersionSuffixPattern.ReplaceAllString(normalized, "")
	switch base {
	case "gpt-5.4", "gpt-5.4-pro":
		return true
	default:
		return false
	}
}

// CalculateCostWithConfig calculates cost using the configured default multiplier.
func (s *BillingService) CalculateCostWithConfig(model string, tokens UsageTokens) (*CostBreakdown, error) {
	multiplier := s.cfg.Default.RateMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}
	return s.CalculateCost(model, tokens, multiplier)
}

// CalculateCostWithLongContext calculates cost with a long-context overflow multiplier.
// threshold: threshold value, for example 200000.
// extraMultiplier: multiplier applied to the overflow portion, for example 2.0.
//
// Example:
// cache-read 210k + input 10k = 220k, threshold 200k, multiplier 2.0.
// Split into in-range (200k, 0) and overflow (10k, 10k).
// The in-range portion uses normal pricing and the overflow portion uses the extra multiplier.
func (s *BillingService) CalculateCostWithLongContext(model string, tokens UsageTokens, rateMultiplier float64, threshold int, extraMultiplier float64) (*CostBreakdown, error) {
	// Invalid long-context configuration falls back to normal pricing.
	if threshold <= 0 || extraMultiplier <= 1 {
		return s.CalculateCost(model, tokens, rateMultiplier)
	}

	pricing, err := s.getPricingForBilling(model)
	if err != nil {
		return nil, err
	}

	// Determine whether input + cache-read tokens cross the threshold.
	total := tokens.CacheReadTokens + tokens.InputTokens
	if total <= threshold {
		return s.calculateCostWithPricing(pricing, tokens, tokens, rateMultiplier, ""), nil
	}

	// Split tokens into in-range and overflow ranges.
	var inRangeCacheTokens, inRangeInputTokens int
	var outRangeCacheTokens, outRangeInputTokens int

	if tokens.CacheReadTokens >= threshold {
		// Cache-read tokens alone already exceed the threshold.
		inRangeCacheTokens = threshold
		inRangeInputTokens = 0
		outRangeCacheTokens = tokens.CacheReadTokens - threshold
		outRangeInputTokens = tokens.InputTokens
	} else {
		// Cache-read tokens stay in range, input tokens cross the threshold.
		inRangeCacheTokens = tokens.CacheReadTokens
		inRangeInputTokens = threshold - tokens.CacheReadTokens
		outRangeCacheTokens = 0
		outRangeInputTokens = tokens.InputTokens - inRangeInputTokens
	}

	// Price the in-range portion normally.
	inRangeTokens := UsageTokens{
		InputTokens:           inRangeInputTokens,
		OutputTokens:          tokens.OutputTokens, // output tokens stay in the base tier
		CacheCreationTokens:   tokens.CacheCreationTokens,
		CacheReadTokens:       inRangeCacheTokens,
		CacheCreation5mTokens: tokens.CacheCreation5mTokens,
		CacheCreation1hTokens: tokens.CacheCreation1hTokens,
	}
	inRangeCost := s.calculateCostWithPricing(pricing, inRangeTokens, tokens, rateMultiplier, "")

	// Price the overflow portion with the extra multiplier.
	outRangeTokens := UsageTokens{
		InputTokens:     outRangeInputTokens,
		CacheReadTokens: outRangeCacheTokens,
	}
	outRangeCost := s.calculateCostWithPricing(pricing, outRangeTokens, tokens, rateMultiplier*extraMultiplier, "")

	// Merge the two partial cost breakdowns.
	return &CostBreakdown{
		InputCost:         inRangeCost.InputCost + outRangeCost.InputCost,
		OutputCost:        inRangeCost.OutputCost,
		CacheCreationCost: inRangeCost.CacheCreationCost,
		CacheReadCost:     inRangeCost.CacheReadCost + outRangeCost.CacheReadCost,
		TotalCost:         inRangeCost.TotalCost + outRangeCost.TotalCost,
		ActualCost:        inRangeCost.ActualCost + outRangeCost.ActualCost,
	}, nil
}

// ListSupportedModels returns models covered by fallback pricing.
func (s *BillingService) ListSupportedModels() []string {
	models := make([]string, 0)
	// Return model families covered by fallback pricing.
	for model := range s.fallbackPrices {
		models = append(models, model)
	}
	return models
}

// IsModelSupported checks whether a model is covered by fallback pricing rules.
func (s *BillingService) IsModelSupported(model string) bool {
	// All Claude-family models have fallback pricing coverage.
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "claude") ||
		strings.Contains(modelLower, "opus") ||
		strings.Contains(modelLower, "sonnet") ||
		strings.Contains(modelLower, "haiku")
}

// GetEstimatedCost returns an estimated cost for frontend display.
func (s *BillingService) GetEstimatedCost(model string, estimatedInputTokens, estimatedOutputTokens int) (float64, error) {
	tokens := UsageTokens{
		InputTokens:  estimatedInputTokens,
		OutputTokens: estimatedOutputTokens,
	}

	breakdown, err := s.CalculateCostWithConfig(model, tokens)
	if err != nil {
		return 0, err
	}

	return breakdown.ActualCost, nil
}

// GetPricingServiceStatus returns pricing service status.
func (s *BillingService) GetPricingServiceStatus() map[string]any {
	if s.pricingService != nil {
		return s.pricingService.GetStatus()
	}
	return map[string]any{
		"model_count":  len(s.fallbackPrices),
		"last_updated": "using fallback",
		"local_hash":   "N/A",
	}
}

// ForceUpdatePricing forces a pricing data refresh.
func (s *BillingService) ForceUpdatePricing() error {
	if s.pricingService != nil {
		return s.pricingService.ForceUpdate()
	}
	return fmt.Errorf("pricing service not initialized")
}

// ImagePriceConfig defines image billing overrides.
type ImagePriceConfig struct {
	Price1K *float64 // 1K size price, nil means using the default price
	Price2K *float64 // 2K size price, nil means using the default price
	Price4K *float64 // 4K size price, nil means using the default price
}

// CalculateImageCost calculates image-generation cost.
// model: requested model name, used to resolve default pricing.
// imageSize: image size such as "1K", "2K", or "4K".
// imageCount: generated image count.
// groupConfig: optional group override pricing, nil means using defaults.
// rateMultiplier: billing rate multiplier.
func (s *BillingService) CalculateImageCost(model string, imageSize string, imageCount int, groupConfig *ImagePriceConfig, rateMultiplier float64) *CostBreakdown {
	return s.CalculateImageCostWithServiceTier(model, imageSize, imageCount, groupConfig, rateMultiplier, "")
}

func (s *BillingService) CalculateImageCostWithServiceTier(model string, imageSize string, imageCount int, groupConfig *ImagePriceConfig, rateMultiplier float64, serviceTier string) *CostBreakdown {
	if imageCount <= 0 {
		return &CostBreakdown{}
	}

	unitPrice := s.getImageUnitPrice(model, imageSize, groupConfig, serviceTier)
	totalCost := unitPrice * float64(imageCount)
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	actualCost := totalCost * rateMultiplier

	return &CostBreakdown{
		TotalCost:  totalCost,
		ActualCost: actualCost,
	}
}

// CalculateVideoRequestCost calculates one-shot video request billing using model pricing.
func (s *BillingService) CalculateVideoRequestCost(model string, rateMultiplier float64) *CostBreakdown {
	unitPrice := 0.0
	if pricing, err := s.getPricingForBilling(model); err == nil && pricing != nil && pricing.OutputPricePerVideoRequest > 0 {
		unitPrice = pricing.OutputPricePerVideoRequest
	}
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	return &CostBreakdown{
		TotalCost:  unitPrice,
		ActualCost: unitPrice * rateMultiplier,
	}
}

// getImageUnitPrice returns the unit price for image generation.
func (s *BillingService) getImageUnitPrice(model string, imageSize string, groupConfig *ImagePriceConfig, serviceTier string) float64 {
	if groupConfig != nil {
		switch imageSize {
		case "1K":
			if groupConfig.Price1K != nil {
				return *groupConfig.Price1K
			}
		case "2K":
			if groupConfig.Price2K != nil {
				return *groupConfig.Price2K
			}
		case "4K":
			if groupConfig.Price4K != nil {
				return *groupConfig.Price4K
			}
		}
	}
	return s.getDefaultImagePrice(model, imageSize, serviceTier)
}

// getDefaultImagePrice returns the default image price for the requested size and service tier.
func (s *BillingService) getDefaultImagePrice(model string, imageSize string, serviceTier string) float64 {
	basePrice := 0.0

	if pricing, err := s.getPricingForBilling(model); err == nil && pricing != nil {
		basePrice = pricing.OutputPricePerImage
		switch normalizeBillingServiceTier(serviceTier) {
		case BillingServiceTierPriority:
			if pricing.OutputPricePerImagePriority > 0 {
				basePrice = pricing.OutputPricePerImagePriority
			}
		case BillingServiceTierFlex:
			if basePrice > 0 {
				basePrice *= serviceTierCostMultiplier(BillingServiceTierFlex)
			}
		}
	}
	if basePrice <= 0 {
		basePrice = 0.134
	}
	if imageSize == "2K" {
		return basePrice * 1.5
	}
	if imageSize == "4K" {
		return basePrice * 2
	}

	return basePrice
}
