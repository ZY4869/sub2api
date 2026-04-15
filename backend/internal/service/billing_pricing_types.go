package service

type BillingPriceItemMode string

const (
	BillingPriceItemModeBase         BillingPriceItemMode = "base"
	BillingPriceItemModeTiered       BillingPriceItemMode = "tiered"
	BillingPriceItemModeBatch        BillingPriceItemMode = "batch"
	BillingPriceItemModeServiceTier  BillingPriceItemMode = "service_tier"
	BillingPriceItemModeProviderRule BillingPriceItemMode = "provider_special"
	billingPricingRuleIDPrefix                            = "pricing_item"
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

type BillingPricingListItem struct {
	Model          string                     `json:"model"`
	DisplayName    string                     `json:"display_name,omitempty"`
	Provider       string                     `json:"provider,omitempty"`
	Mode           string                     `json:"mode,omitempty"`
	PriceItemCount int                        `json:"price_item_count"`
	OfficialCount  int                        `json:"official_count"`
	SaleCount      int                        `json:"sale_count"`
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
	SupportsPromptCaching           bool                       `json:"supports_prompt_caching"`
	SupportsServiceTier             bool                       `json:"supports_service_tier"`
	LongContextInputTokenThreshold  int                        `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier  float64                    `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier float64                    `json:"long_context_output_cost_multiplier,omitempty"`
	Capabilities                    BillingPricingCapabilities `json:"capabilities"`
	OfficialItems                   []BillingPriceItem         `json:"official_items"`
	SaleItems                       []BillingPriceItem         `json:"sale_items"`
}

type BillingPricingListFilter struct {
	Search   string
	Provider string
	Mode     string
	Page     int
	PageSize int
}

type BillingPricingDetailsRequest struct {
	Models []string `json:"models"`
}

type UpsertBillingPricingLayerInput struct {
	Model string             `json:"model"`
	Layer string             `json:"layer"`
	Items []BillingPriceItem `json:"items"`
}

type BillingCopyOfficialToSaleInput struct {
	Models []string `json:"models"`
}

type BillingBulkApplyRequest struct {
	Models        []string `json:"models"`
	ItemIDs       []string `json:"item_ids,omitempty"`
	DiscountRatio float64  `json:"discount_ratio"`
}
