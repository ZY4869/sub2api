package service

type BillingRuleMatchers struct {
	Models         []string `json:"models,omitempty"`
	ModelFamilies  []string `json:"model_families,omitempty"`
	InputModality  string   `json:"input_modality,omitempty"`
	OutputModality string   `json:"output_modality,omitempty"`
	CachePhase     string   `json:"cache_phase,omitempty"`
	GroundingKind  string   `json:"grounding_kind,omitempty"`
	ContextWindow  string   `json:"context_window,omitempty"`
}

type BillingRule struct {
	ID            string              `json:"id"`
	Provider      string              `json:"provider"`
	Layer         string              `json:"layer"`
	Surface       string              `json:"surface"`
	OperationType string              `json:"operation_type"`
	ServiceTier   string              `json:"service_tier"`
	BatchMode     string              `json:"batch_mode"`
	Matchers      BillingRuleMatchers `json:"matchers"`
	Unit          string              `json:"unit"`
	Price         float64             `json:"price"`
	Priority      int                 `json:"priority"`
	Enabled       bool                `json:"enabled"`
}

type GeminiBillingMatrixCell struct {
	Price      *float64 `json:"price,omitempty"`
	RuleID     string   `json:"rule_id,omitempty"`
	DerivedVia string   `json:"derived_via,omitempty"`
}

type GeminiBillingMatrixRow struct {
	Surface     string                             `json:"surface"`
	ServiceTier string                             `json:"service_tier"`
	Slots       map[string]GeminiBillingMatrixCell `json:"slots"`
}

type GeminiBillingMatrix struct {
	Surfaces     []string                 `json:"surfaces"`
	ServiceTiers []string                 `json:"service_tiers"`
	ChargeSlots  []string                 `json:"charge_slots"`
	Rows         []GeminiBillingMatrixRow `json:"rows"`
}

type ModelBillingSheet struct {
	ID                              string               `json:"id"`
	Provider                        string               `json:"provider"`
	Model                           string               `json:"model"`
	ModelFamily                     string               `json:"model_family,omitempty"`
	DisplayName                     string               `json:"display_name,omitempty"`
	OfficialPricing                 *ModelCatalogPricing `json:"official_pricing,omitempty"`
	SalePricing                     *ModelCatalogPricing `json:"sale_pricing,omitempty"`
	OfficialMatrix                  *GeminiBillingMatrix `json:"official_matrix,omitempty"`
	SaleMatrix                      *GeminiBillingMatrix `json:"sale_matrix,omitempty"`
	SupportsServiceTier             bool                 `json:"supports_service_tier"`
	LongContextInputTokenThreshold  int                  `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier  float64              `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier float64              `json:"long_context_output_cost_multiplier,omitempty"`
}

type BillingCenterPayload struct {
	Sheets []ModelBillingSheet `json:"sheets"`
	Rules  []BillingRule       `json:"rules"`
}

type UpsertModelBillingSheetInput struct {
	Model   string               `json:"model"`
	Layer   string               `json:"layer"`
	Pricing *ModelCatalogPricing `json:"pricing,omitempty"`
	Matrix  *GeminiBillingMatrix `json:"matrix,omitempty"`
}

type BillingSimulationCharges struct {
	TextInputTokens           float64 `json:"text_input_tokens,omitempty"`
	TextOutputTokens          float64 `json:"text_output_tokens,omitempty"`
	AudioInputTokens          float64 `json:"audio_input_tokens,omitempty"`
	AudioOutputTokens         float64 `json:"audio_output_tokens,omitempty"`
	CacheCreateTokens         float64 `json:"cache_create_tokens,omitempty"`
	CacheReadTokens           float64 `json:"cache_read_tokens,omitempty"`
	CacheStorageTokenHours    float64 `json:"cache_storage_token_hours,omitempty"`
	ImageOutputs              float64 `json:"image_outputs,omitempty"`
	VideoRequests             float64 `json:"video_requests,omitempty"`
	FileSearchEmbeddingTokens float64 `json:"file_search_embedding_tokens,omitempty"`
	FileSearchRetrievalTokens float64 `json:"file_search_retrieval_tokens,omitempty"`
	GroundingSearchQueries    float64 `json:"grounding_search_queries,omitempty"`
	GroundingMapsQueries      float64 `json:"grounding_maps_queries,omitempty"`
}

type BillingSimulationInput struct {
	Provider       string                   `json:"provider"`
	Layer          string                   `json:"layer"`
	Model          string                   `json:"model"`
	Surface        string                   `json:"surface"`
	OperationType  string                   `json:"operation_type"`
	ServiceTier    string                   `json:"service_tier"`
	BatchMode      string                   `json:"batch_mode"`
	InputModality  string                   `json:"input_modality"`
	OutputModality string                   `json:"output_modality"`
	CachePhase     string                   `json:"cache_phase"`
	GroundingKind  string                   `json:"grounding_kind"`
	Charges        BillingSimulationCharges `json:"charges,omitempty"`

	InputTokens         float64 `json:"input_tokens,omitempty"`
	OutputTokens        float64 `json:"output_tokens,omitempty"`
	CacheCreationTokens float64 `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens     float64 `json:"cache_read_tokens,omitempty"`
	ImageCount          float64 `json:"image_count,omitempty"`
	VideoRequests       float64 `json:"video_requests,omitempty"`
	MediaUnits          float64 `json:"media_units,omitempty"`
}

type BillingSimulationMatchedRule struct {
	ID            string              `json:"id"`
	Provider      string              `json:"provider"`
	Layer         string              `json:"layer"`
	Surface       string              `json:"surface"`
	OperationType string              `json:"operation_type"`
	ServiceTier   string              `json:"service_tier"`
	BatchMode     string              `json:"batch_mode"`
	Unit          string              `json:"unit"`
	Price         float64             `json:"price"`
	Priority      int                 `json:"priority"`
	Matchers      BillingRuleMatchers `json:"matchers"`
}

type BillingSimulationLine struct {
	ChargeSlot string  `json:"charge_slot"`
	Unit       string  `json:"unit"`
	Units      float64 `json:"units"`
	Price      float64 `json:"price"`
	Cost       float64 `json:"cost"`
	ActualCost float64 `json:"actual_cost"`
	RuleID     string  `json:"rule_id,omitempty"`
	RuleLabel  string  `json:"rule_label,omitempty"`
}

type BillingSimulationUnmatchedDemand struct {
	ChargeSlot        string   `json:"charge_slot"`
	Unit              string   `json:"unit"`
	Units             float64  `json:"units"`
	Reason            string   `json:"reason"`
	MissingDimensions []string `json:"missing_dimensions,omitempty"`
}

type BillingSimulationFallback struct {
	Policy      string                  `json:"policy,omitempty"`
	Applied     bool                    `json:"applied"`
	Reason      string                  `json:"reason,omitempty"`
	DerivedFrom string                  `json:"derived_from,omitempty"`
	CostLines   []BillingSimulationLine `json:"cost_lines,omitempty"`
}

type BillingSimulationResult struct {
	Classification   *GeminiRequestClassification       `json:"classification,omitempty"`
	MatchedRules     []BillingSimulationMatchedRule     `json:"matched_rules,omitempty"`
	MatchedRuleIDs   []string                           `json:"matched_rule_ids,omitempty"`
	Lines            []BillingSimulationLine            `json:"lines"`
	UnmatchedDemands []BillingSimulationUnmatchedDemand `json:"unmatched_demands,omitempty"`
	Fallback         *BillingSimulationFallback         `json:"fallback,omitempty"`
	TotalCost        float64                            `json:"total_cost"`
	ActualCost       float64                            `json:"actual_cost"`
}

type GeminiRequestClassification struct {
	Surface        string `json:"surface"`
	OperationType  string `json:"operation_type"`
	ServiceTier    string `json:"service_tier,omitempty"`
	BatchMode      string `json:"batch_mode,omitempty"`
	InputModality  string `json:"input_modality,omitempty"`
	OutputModality string `json:"output_modality,omitempty"`
	CachePhase     string `json:"cache_phase,omitempty"`
	GroundingKind  string `json:"grounding_kind,omitempty"`
	ContextWindow  string `json:"context_window,omitempty"`
	ChargeSource   string `json:"charge_source,omitempty"`
	MediaType      string `json:"media_type,omitempty"`
	MediaUnits     int    `json:"media_units,omitempty"`
}

type GeminiBillingCalculationInput struct {
	Model                string
	InboundEndpoint      string
	RequestBody          []byte
	Tokens               UsageTokens
	ImageCount           int
	VideoRequests        int
	MediaType            string
	RateMultiplier       float64
	RequestedServiceTier string
	Charges              BillingSimulationCharges
}

type GeminiBillingCalculationResult struct {
	Cost             *CostBreakdown                     `json:"cost,omitempty"`
	Classification   *GeminiRequestClassification       `json:"classification,omitempty"`
	MatchedRules     []BillingSimulationMatchedRule     `json:"matched_rules,omitempty"`
	MatchedRuleIDs   []string                           `json:"matched_rule_ids,omitempty"`
	Lines            []BillingSimulationLine            `json:"lines,omitempty"`
	UnmatchedDemands []BillingSimulationUnmatchedDemand `json:"unmatched_demands,omitempty"`
	Fallback         *BillingSimulationFallback         `json:"fallback,omitempty"`
	TotalCost        float64                            `json:"total_cost"`
	ActualCost       float64                            `json:"actual_cost"`
}

const (
	BillingRuleProviderGemini = "gemini"

	BillingLayerOfficial = "official"
	BillingLayerSale     = "sale"

	BillingSurfaceAny            = "any"
	BillingSurfaceGeminiNative   = "native"
	BillingSurfaceOpenAICompat   = "openai_compat"
	BillingSurfaceGeminiLive     = "live"
	BillingSurfaceInteractions   = "interactions"
	BillingSurfaceVertexExisting = "vertex_existing"

	BillingBatchModeAny      = "any"
	BillingBatchModeRealtime = "realtime"
	BillingBatchModeBatch    = "batch"

	BillingServiceTierStandard = "standard"
	BillingServiceTierFlex     = "flex"
	BillingServiceTierPriority = "priority"

	BillingContextWindowStandard = "standard"
	BillingContextWindowLong     = "long"

	BillingUnitInputToken             = "input_token"
	BillingUnitOutputToken            = "output_token"
	BillingUnitCacheCreateToken       = "cache_create_token"
	BillingUnitCacheReadToken         = "cache_read_token"
	BillingUnitCacheStorageTokenHour  = "cache_storage_token_hour"
	BillingUnitImage                  = "image"
	BillingUnitVideoRequest           = "video_request"
	BillingUnitMediaUnit              = "media_unit"
	BillingUnitFileSearchEmbedding    = "file_search_embedding_token"
	BillingUnitFileSearchRetrieval    = "file_search_retrieval_token"
	BillingUnitGroundingSearchRequest = "grounding_search_request"
	BillingUnitGroundingMapsRequest   = "grounding_maps_request"

	BillingChargeSlotTextInput                = "text_input"
	BillingChargeSlotTextInputLongContext     = "text_input_long_context"
	BillingChargeSlotTextOutput               = "text_output"
	BillingChargeSlotTextOutputLongContext    = "text_output_long_context"
	BillingChargeSlotAudioInput               = "audio_input"
	BillingChargeSlotAudioOutput              = "audio_output"
	BillingChargeSlotCacheCreate              = "cache_create"
	BillingChargeSlotCacheRead                = "cache_read"
	BillingChargeSlotCacheStorageTokenHour    = "cache_storage_token_hour"
	BillingChargeSlotImageOutput              = "image_output"
	BillingChargeSlotVideoRequest             = "video_request"
	BillingChargeSlotFileSearchEmbeddingToken = "file_search_embedding_token"
	BillingChargeSlotFileSearchRetrievalToken = "file_search_retrieval_token"
	BillingChargeSlotGroundingSearchRequest   = "grounding_search_request"
	BillingChargeSlotGroundingMapsRequest     = "grounding_maps_request"
)
