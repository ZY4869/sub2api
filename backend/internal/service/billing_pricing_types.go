package service

import "time"

type BillingPriceItemMode string
type BillingPricingMultiplierMode string
type BillingPricingStatus string

const (
	BillingPriceItemModeBase         BillingPriceItemMode         = "base"
	BillingPriceItemModeTiered       BillingPriceItemMode         = "tiered"
	BillingPriceItemModeBatch        BillingPriceItemMode         = "batch"
	BillingPriceItemModeServiceTier  BillingPriceItemMode         = "service_tier"
	BillingPriceItemModeProviderRule BillingPriceItemMode         = "provider_special"
	BillingPricingMultiplierShared   BillingPricingMultiplierMode = "shared"
	BillingPricingMultiplierItem     BillingPricingMultiplierMode = "item"
	BillingPricingStatusOK           BillingPricingStatus         = "ok"
	BillingPricingStatusFallback     BillingPricingStatus         = "fallback"
	BillingPricingStatusConflict     BillingPricingStatus         = "conflict"
	BillingPricingStatusMissing      BillingPricingStatus         = "missing"
	billingPricingRuleIDPrefix                                    = "pricing_item"
)

type BillingPriceItem struct {
	ID                string               `json:"id"`
	ChargeSlot        string               `json:"charge_slot"`
	Unit              string               `json:"unit"`
	Layer             string               `json:"layer"`
	Mode              BillingPriceItemMode `json:"mode"`
	ServiceTier       string               `json:"service_tier,omitempty"`
	BatchMode         string               `json:"batch_mode,omitempty"`
	Surface           string               `json:"surface,omitempty"`
	OperationType     string               `json:"operation_type,omitempty"`
	InputModality     string               `json:"input_modality,omitempty"`
	OutputModality    string               `json:"output_modality,omitempty"`
	CachePhase        string               `json:"cache_phase,omitempty"`
	GroundingKind     string               `json:"grounding_kind,omitempty"`
	ContextWindow     string               `json:"context_window,omitempty"`
	ThresholdTokens   *int                 `json:"threshold_tokens,omitempty"`
	Price             float64              `json:"price"`
	PriceAboveThresh  *float64             `json:"price_above_threshold,omitempty"`
	FormulaSource     string               `json:"formula_source,omitempty"`
	FormulaMultiplier *float64             `json:"formula_multiplier,omitempty"`
	RuleID            string               `json:"rule_id,omitempty"`
	DerivedVia        string               `json:"derived_via,omitempty"`
	Enabled           bool                 `json:"enabled"`
}

type BillingPricingCapabilities struct {
	SupportsTieredPricing   bool `json:"supports_tiered_pricing"`
	SupportsBatchPricing    bool `json:"supports_batch_pricing"`
	SupportsServiceTier     bool `json:"supports_service_tier"`
	SupportsPromptCaching   bool `json:"supports_prompt_caching"`
	SupportsProviderSpecial bool `json:"supports_provider_special"`
}

type BillingPricingSimpleSpecial struct {
	BatchInputPrice     *float64 `json:"batch_input_price,omitempty"`
	BatchOutputPrice    *float64 `json:"batch_output_price,omitempty"`
	BatchCachePrice     *float64 `json:"batch_cache_price,omitempty"`
	GroundingSearch     *float64 `json:"grounding_search,omitempty"`
	GroundingMaps       *float64 `json:"grounding_maps,omitempty"`
	FileSearchEmbedding *float64 `json:"file_search_embedding,omitempty"`
	FileSearchRetrieval *float64 `json:"file_search_retrieval,omitempty"`
}

type BillingPricingLayerForm struct {
	InputPrice                *float64                     `json:"input_price,omitempty"`
	OutputPrice               *float64                     `json:"output_price,omitempty"`
	CachePrice                *float64                     `json:"cache_price,omitempty"`
	SpecialEnabled            bool                         `json:"special_enabled"`
	Special                   BillingPricingSimpleSpecial  `json:"special"`
	TieredEnabled             bool                         `json:"tiered_enabled"`
	TierThresholdTokens       *int                         `json:"tier_threshold_tokens,omitempty"`
	InputPriceAboveThreshold  *float64                     `json:"input_price_above_threshold,omitempty"`
	OutputPriceAboveThreshold *float64                     `json:"output_price_above_threshold,omitempty"`
	MultiplierEnabled         bool                         `json:"multiplier_enabled"`
	MultiplierMode            BillingPricingMultiplierMode `json:"multiplier_mode,omitempty"`
	SharedMultiplier          *float64                     `json:"shared_multiplier,omitempty"`
	ItemMultipliers           map[string]float64           `json:"item_multipliers,omitempty"`
}

type BillingPricingCurrencyPreference struct {
	Currency        string    `json:"currency"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedByUserID int64     `json:"updated_by_user_id"`
	UpdatedByEmail  string    `json:"updated_by_email,omitempty"`
}

type BillingPricingListItem struct {
	Model          string                     `json:"model"`
	DisplayName    string                     `json:"display_name,omitempty"`
	Provider       string                     `json:"provider,omitempty"`
	Mode           string                     `json:"mode,omitempty"`
	PriceItemCount int                        `json:"price_item_count"`
	OfficialCount  int                        `json:"official_count"`
	SaleCount      int                        `json:"sale_count"`
	PricingStatus  BillingPricingStatus       `json:"pricing_status"`
	PricingWarnings []string                  `json:"pricing_warnings,omitempty"`
	Capabilities   BillingPricingCapabilities `json:"capabilities"`
}

type BillingPricingProviderGroup struct {
	Provider      string `json:"provider"`
	Label         string `json:"label"`
	TotalCount    int    `json:"total_count"`
	OfficialCount int    `json:"official_count"`
	SaleCount     int    `json:"sale_count"`
}

type BillingPricingSheetDetail struct {
	Model                           string                     `json:"model"`
	DisplayName                     string                     `json:"display_name,omitempty"`
	Provider                        string                     `json:"provider,omitempty"`
	Mode                            string                     `json:"mode,omitempty"`
	InputSupported                  bool                       `json:"input_supported"`
	OutputChargeSlot                string                     `json:"output_charge_slot,omitempty"`
	SupportsPromptCaching           bool                       `json:"supports_prompt_caching"`
	SupportsServiceTier             bool                       `json:"supports_service_tier"`
	LongContextInputTokenThreshold  int                        `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier  float64                    `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier float64                    `json:"long_context_output_cost_multiplier,omitempty"`
	Currency                        string                     `json:"currency"`
	PricingStatus                   BillingPricingStatus       `json:"pricing_status"`
	PricingWarnings                 []string                   `json:"pricing_warnings,omitempty"`
	Capabilities                    BillingPricingCapabilities `json:"capabilities"`
	OfficialForm                    BillingPricingLayerForm    `json:"official_form"`
	SaleForm                        BillingPricingLayerForm    `json:"sale_form"`
	OfficialItems                   []BillingPriceItem         `json:"-"`
	SaleItems                       []BillingPriceItem         `json:"-"`
}

type BillingPricingListFilter struct {
	Search    string
	Provider  string
	Mode      string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

type BillingPricingDetailsRequest struct {
	Models []string `json:"models"`
}

type UpsertBillingPricingLayerInput struct {
	Model    string                   `json:"model"`
	Layer    string                   `json:"layer"`
	Currency string                   `json:"currency,omitempty"`
	Form     *BillingPricingLayerForm `json:"form,omitempty"`
	Items    []BillingPriceItem       `json:"items,omitempty"`
}

type BillingCopyOfficialToSaleInput struct {
	Models []string `json:"models"`
}

type BillingBulkApplyRequest struct {
	Models        []string `json:"models"`
	ItemIDs       []string `json:"item_ids,omitempty"`
	DiscountRatio float64  `json:"discount_ratio"`
}
