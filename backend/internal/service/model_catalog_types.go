package service

import "time"

const (
	ModelCatalogPricingSourceNone     = "none"
	ModelCatalogPricingSourceDynamic  = "dynamic"
	ModelCatalogPricingSourceFallback = "fallback"
	ModelCatalogPricingSourceOverride = "override"
	ModelCatalogAccessSourceLogin     = "login"
	ModelCatalogAccessSourceKey       = "key"
)

type ModelCatalogPricing struct {
	InputCostPerToken                        *float64 `json:"input_cost_per_token,omitempty"`
	InputCostPerTokenPriority                *float64 `json:"input_cost_per_token_priority,omitempty"`
	InputTokenThreshold                      *int     `json:"input_token_threshold,omitempty"`
	InputCostPerTokenAboveThreshold          *float64 `json:"input_cost_per_token_above_threshold,omitempty"`
	InputCostPerTokenPriorityAboveThreshold  *float64 `json:"input_cost_per_token_priority_above_threshold,omitempty"`
	OutputCostPerToken                       *float64 `json:"output_cost_per_token,omitempty"`
	OutputCostPerTokenPriority               *float64 `json:"output_cost_per_token_priority,omitempty"`
	OutputTokenThreshold                     *int     `json:"output_token_threshold,omitempty"`
	OutputCostPerTokenAboveThreshold         *float64 `json:"output_cost_per_token_above_threshold,omitempty"`
	OutputCostPerTokenPriorityAboveThreshold *float64 `json:"output_cost_per_token_priority_above_threshold,omitempty"`
	CacheCreationInputTokenCost              *float64 `json:"cache_creation_input_token_cost,omitempty"`
	CacheCreationInputTokenCostAbove1hr      *float64 `json:"cache_creation_input_token_cost_above_1hr,omitempty"`
	CacheReadInputTokenCost                  *float64 `json:"cache_read_input_token_cost,omitempty"`
	CacheReadInputTokenCostPriority          *float64 `json:"cache_read_input_token_cost_priority,omitempty"`
	OutputCostPerImage                       *float64 `json:"output_cost_per_image,omitempty"`
	OutputCostPerImagePriority               *float64 `json:"output_cost_per_image_priority,omitempty"`
	OutputCostPerVideoRequest                *float64 `json:"output_cost_per_video_request,omitempty"`
}

type ModelPricingOverride struct {
	ModelCatalogPricing
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedByUserID int64     `json:"updated_by_user_id"`
	UpdatedByEmail  string    `json:"updated_by_email,omitempty"`
}

type ModelCatalogExchangeRate struct {
	Base      string    `json:"base"`
	Quote     string    `json:"quote"`
	Rate      float64   `json:"rate"`
	Date      string    `json:"date"`
	UpdatedAt time.Time `json:"updated_at"`
	Cached    bool      `json:"cached"`
}

type ModelCatalogItem struct {
	Model                           string               `json:"model"`
	DisplayName                     string               `json:"display_name,omitempty"`
	IconKey                         string               `json:"icon_key,omitempty"`
	Provider                        string               `json:"provider,omitempty"`
	Mode                            string               `json:"mode,omitempty"`
	DefaultAvailable                bool                 `json:"default_available"`
	DefaultPlatforms                []string             `json:"default_platforms,omitempty"`
	AccessSources                   []string             `json:"access_sources,omitempty"`
	PricingSource                   string               `json:"pricing_source"`
	BasePricingSource               string               `json:"base_pricing_source"`
	HasOverride                     bool                 `json:"has_override"`
	OfficialPricing                 *ModelCatalogPricing `json:"official_pricing,omitempty"`
	SalePricing                     *ModelCatalogPricing `json:"sale_pricing,omitempty"`
	EffectivePricing                *ModelCatalogPricing `json:"effective_pricing,omitempty"`
	SupportsPromptCaching           bool                 `json:"supports_prompt_caching"`
	SupportsServiceTier             bool                 `json:"supports_service_tier"`
	LongContextInputTokenThreshold  int                  `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier  float64              `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier float64              `json:"long_context_output_cost_multiplier,omitempty"`
}

type ModelCatalogDetail struct {
	ModelCatalogItem
	UpstreamPricing         *ModelCatalogPricing         `json:"upstream_pricing,omitempty"`
	OfficialOverridePricing *ModelPricingOverride        `json:"official_override_pricing,omitempty"`
	SaleOverridePricing     *ModelPricingOverride        `json:"sale_override_pricing,omitempty"`
	BasePricing             *ModelCatalogPricing         `json:"base_pricing,omitempty"`
	OverridePricing         *ModelPricingOverride        `json:"override_pricing,omitempty"`
	RouteReferences         []ModelCatalogRouteReference `json:"route_references"`
	RouteReferenceCount     int                          `json:"route_reference_count"`
}

type ModelCatalogRouteReference struct {
	GroupID                int64    `json:"group_id"`
	GroupName              string   `json:"group_name"`
	Platform               string   `json:"platform"`
	ReferenceTypes         []string `json:"reference_types"`
	MatchedRoutingPatterns []string `json:"matched_routing_patterns,omitempty"`
}

type ModelCatalogListFilter struct {
	Search        string
	Provider      string
	Mode          string
	Availability  string
	PricingSource string
	Page          int
	PageSize      int
}

type ModelCatalogActor struct {
	UserID int64
	Email  string
}

type ModelCatalogEntryPayload struct {
	Model string `json:"model"`
}

type UpsertModelPricingOverrideInput struct {
	Model string `json:"model"`
	ModelCatalogPricing
}
