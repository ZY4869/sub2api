package service

import (
	"context"
	"sync"
	"time"

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
	Currency                                  string
	USDToCNYRate                              float64
	FXRateDate                                string
	FXLockedAt                                *time.Time
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
	Currency                string
	USDToCNYRate            float64
	FXRateDate              string
	FXLockedAt              *time.Time
	InputCost               float64
	OutputCost              float64
	CacheCreationCost       float64
	CacheReadCost           float64
	TotalCost               float64
	ActualCost              float64 // Final billed amount in Currency after multipliers are applied.
	TotalCostUSDEquivalent  float64
	ActualCostUSDEquivalent float64
	CostByCurrency          map[string]float64
	ActualCostByCurrency    map[string]float64
}

// BillingService provides billing and pricing operations.
type BillingService struct {
	cfg                    *config.Config
	pricingService         *PricingService
	modelRegistryService   *ModelRegistryService
	billingCenterService   *BillingCenterService
	fallbackPrices         map[string]*ModelPricing // fallback pricing table
	fallbackPricingLogs    sync.Map
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

// CalculateCostWithConfig calculates cost using the configured default multiplier.
func (s *BillingService) CalculateCostWithConfig(model string, tokens UsageTokens) (*CostBreakdown, error) {
	multiplier := s.cfg.Default.RateMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}
	return s.CalculateCost(model, tokens, multiplier)
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

// ImagePriceConfig defines image billing overrides.
type ImagePriceConfig struct {
	Price1K *float64 // 1K size price, nil means using the default price
	Price2K *float64 // 2K size price, nil means using the default price
	Price4K *float64 // 4K size price, nil means using the default price
}
