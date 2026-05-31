package service

const (
	PublicModelCatalogSourcePublished    = "published"
	PublicModelCatalogSourceLiveFallback = "live_fallback"

	PublicModelStatusOK          = "ok"
	PublicModelStatusError       = "error"
	PublicModelStatusMaintenance = "maintenance"
	PublicModelStatusWarning     = "warning"
	PublicModelStatusInfo        = "info"

	PublicModelLifecycleStable     = "stable"
	PublicModelLifecycleBeta       = "beta"
	PublicModelLifecycleDeprecated = "deprecated"
)

const (
	PublicModelSupportSupported   = "supported"
	PublicModelSupportPartial     = "partial"
	PublicModelSupportUnsupported = "unsupported"
	PublicModelSupportUnknown     = "unknown"

	PublicModelCapabilitySourceVerifiedProbe     = "verified_probe"
	PublicModelCapabilitySourceRuntimeObserved   = "runtime_observed"
	PublicModelCapabilitySourceAccountProbe      = "account_probe"
	PublicModelCapabilitySourceOfficialRegistry  = "official_registry"
	PublicModelCapabilitySourceManualConfig      = "manual_config"
	PublicModelCapabilitySourcePricingCatalog    = "pricing_catalog"
	PublicModelCapabilitySourceInferred          = "inferred"
	PublicModelCapabilitySourcePublishedSnapshot = "published_snapshot"

	PublicModelLifecycleSourceOfficialRegistry  = "official_registry"
	PublicModelLifecycleSourceManualConfig      = "manual_config"
	PublicModelLifecycleSourcePublishedSnapshot = "published_snapshot"
	PublicModelLifecycleSourceInferred          = "inferred"

	PublicModelLifecycleConfidenceVerified = "verified"
	PublicModelLifecycleConfidenceDeclared = "declared"
	PublicModelLifecycleConfidenceInferred = "inferred"

	PublicModelContextLimitKindInput = "input"

	PublicModelCatalogEntrySourceRealAccount    = "real_account"
	PublicModelCatalogEntrySourceLiveProjection = "live_projection"
	PublicModelCatalogEntrySourceDemo           = "demo"
	PublicModelCatalogEntrySourceLegacySnapshot = "legacy_snapshot"

	PublicModelCatalogExampleValidationDryRunContract = "dry_run_contract"

	PublicModelCatalogModeReal = "real"
	PublicModelCatalogModeDemo = "demo"
)

type PublicModelCatalogReadOptions struct {
	CatalogMode string
}

type PublicModelCatalogSnapshot struct {
	ETag              string                   `json:"etag"`
	UpdatedAt         string                   `json:"updated_at"`
	RefreshedAt       string                   `json:"refreshed_at,omitempty"`
	PublishedAt       string                   `json:"published_at,omitempty"`
	LastRevalidatedAt string                   `json:"last_revalidated_at,omitempty"`
	StaleReason       string                   `json:"stale_reason,omitempty"`
	PageSize          int                      `json:"page_size,omitempty"`
	CatalogSource     string                   `json:"catalog_source,omitempty"`
	Items             []PublicModelCatalogItem `json:"items"`
}

type PublicModelCatalogItem struct {
	EntryID              string                              `json:"entry_id,omitempty"`
	PublicModelID        string                              `json:"public_model_id,omitempty"`
	Model                string                              `json:"model"`
	BaseModel            string                              `json:"base_model,omitempty"`
	SourceModelID        string                              `json:"source_model_id,omitempty"`
	SourceProtocol       string                              `json:"source_protocol,omitempty"`
	SourceAlias          string                              `json:"source_alias,omitempty"`
	SourceAccountID      int64                               `json:"source_account_id,omitempty"`
	SourceAccountName    string                              `json:"source_account_name,omitempty"`
	DisplayName          string                              `json:"display_name,omitempty"`
	Provider             string                              `json:"provider,omitempty"`
	ProviderIconKey      string                              `json:"provider_icon_key,omitempty"`
	Status               string                              `json:"status,omitempty"`
	AvailabilityState    string                              `json:"availability_state,omitempty"`
	StaleState           string                              `json:"stale_state,omitempty"`
	PublicationStatus    string                              `json:"publication_status,omitempty"`
	HealthStatus         string                              `json:"health_status,omitempty"`
	VerificationSource   string                              `json:"verification_source,omitempty"`
	KeyAvailability      string                              `json:"key_availability,omitempty"`
	UnavailableReason    string                              `json:"unavailable_reason,omitempty"`
	LifecycleStatus      string                              `json:"lifecycle_status,omitempty"`
	Lifecycle            PublicModelLifecycle                `json:"lifecycle,omitempty"`
	ContextWindowTokens  int64                               `json:"context_window_tokens,omitempty"`
	ContextWindow        PublicModelContextWindow            `json:"context_window,omitempty"`
	Modalities           []string                            `json:"modalities,omitempty"`
	Capabilities         []string                            `json:"capabilities,omitempty"`
	CapabilityMatrix     []PublicModelCapabilityMatrixEntry  `json:"capability_matrix,omitempty"`
	RequestProtocols     []string                            `json:"request_protocols,omitempty"`
	ProtocolEndpoints    []PublicModelProtocolEndpoint       `json:"protocol_endpoints,omitempty"`
	SourceIDs            []string                            `json:"source_ids,omitempty"`
	IsDemo               bool                                `json:"is_demo,omitempty"`
	CatalogEntrySource   string                              `json:"catalog_entry_source,omitempty"`
	Mode                 string                              `json:"mode,omitempty"`
	Currency             string                              `json:"currency"`
	PriceDisplay         PublicModelCatalogPriceDisplay      `json:"price_display"`
	OfficialPriceDisplay PublicModelCatalogPriceDisplay      `json:"official_price_display,omitempty"`
	SalePriceDisplay     PublicModelCatalogPriceDisplay      `json:"sale_price_display,omitempty"`
	MultiplierSummary    PublicModelCatalogMultiplierSummary `json:"multiplier_summary"`
	RuntimePriceSpec     PublicModelCatalogRuntimePriceSpec  `json:"runtime_price_spec,omitempty"`
}

type PublicModelContextWindow struct {
	Tokens        int64    `json:"tokens,omitempty"`
	Source        string   `json:"source,omitempty"`
	Verified      bool     `json:"verified"`
	LastCheckedAt string   `json:"last_checked_at,omitempty"`
	LimitKind     string   `json:"limit_kind,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

type PublicModelCapabilityMatrixEntry struct {
	Capability    string   `json:"capability"`
	Protocol      string   `json:"protocol,omitempty"`
	Endpoint      string   `json:"endpoint,omitempty"`
	Support       string   `json:"support"`
	Mode          string   `json:"mode,omitempty"`
	Source        string   `json:"source,omitempty"`
	Verified      bool     `json:"verified"`
	LastCheckedAt string   `json:"last_checked_at,omitempty"`
	Limitations   []string `json:"limitations,omitempty"`
}

type PublicModelProtocolEndpoint struct {
	Key           string   `json:"key"`
	Protocol      string   `json:"protocol"`
	Endpoint      string   `json:"endpoint"`
	Method        string   `json:"method,omitempty"`
	Support       string   `json:"support"`
	Source        string   `json:"source,omitempty"`
	Verified      bool     `json:"verified"`
	LastCheckedAt string   `json:"last_checked_at,omitempty"`
	Limitations   []string `json:"limitations,omitempty"`
}

type PublicModelLifecycle struct {
	Status     string `json:"status,omitempty"`
	Source     string `json:"source,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

type PublicModelCatalogPriceDisplay struct {
	Primary   []PublicModelCatalogPriceEntry `json:"primary"`
	Secondary []PublicModelCatalogPriceEntry `json:"secondary,omitempty"`
}

type PublicModelCatalogPriceEntry struct {
	ID                string  `json:"id"`
	Unit              string  `json:"unit,omitempty"`
	UnitKind          string  `json:"unit_kind,omitempty"`
	DisplayUnit       string  `json:"display_unit,omitempty"`
	Value             float64 `json:"value"`
	Configured        bool    `json:"configured"`
	SupportedUnpriced bool    `json:"supported_unpriced,omitempty"`
}

type PublicModelCatalogMultiplierSummary struct {
	Enabled bool     `json:"enabled"`
	Kind    string   `json:"kind"`
	Mode    string   `json:"mode,omitempty"`
	Value   *float64 `json:"value,omitempty"`
}

type PublicModelCatalogRuntimePriceSpec struct {
	Currency                        string  `json:"currency,omitempty"`
	InputSupported                  bool    `json:"input_supported,omitempty"`
	OutputChargeSlot                string  `json:"output_charge_slot,omitempty"`
	SupportsPromptCaching           bool    `json:"supports_prompt_caching,omitempty"`
	LongContextInputTokenThreshold  int     `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier  float64 `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier float64 `json:"long_context_output_cost_multiplier,omitempty"`
}

type PublicModelCatalogDetail struct {
	Item              PublicModelCatalogItem `json:"item"`
	CatalogSource     string                 `json:"catalog_source,omitempty"`
	ExampleSource     string                 `json:"example_source,omitempty"`
	ExampleProtocol   string                 `json:"example_protocol,omitempty"`
	ExamplePageID     string                 `json:"example_page_id,omitempty"`
	ExampleMarkdown   string                 `json:"example_markdown,omitempty"`
	ExampleOverrideID string                 `json:"example_override_id,omitempty"`
	ExampleValidation string                 `json:"example_validation,omitempty"`
}

type PublicModelCatalogEntryDraft struct {
	EntryID          string                         `json:"entry_id"`
	PublicModelID    string                         `json:"public_model_id"`
	SourceAccountID  int64                          `json:"source_account_id,omitempty"`
	SourceAlias      string                         `json:"source_alias,omitempty"`
	SourceModelID    string                         `json:"source_model_id,omitempty"`
	BaseModel        string                         `json:"base_model,omitempty"`
	SourceProtocol   string                         `json:"source_protocol,omitempty"`
	SalePriceDisplay PublicModelCatalogPriceDisplay `json:"sale_price_display,omitempty"`
}

type PublicModelCatalogDraft struct {
	SelectedModels  []string                       `json:"selected_models,omitempty"`
	SelectedEntries []PublicModelCatalogEntryDraft `json:"selected_entries,omitempty"`
	PageSize        int                            `json:"page_size,omitempty"`
	UpdatedAt       string                         `json:"updated_at,omitempty"`
}

type PublicModelCatalogPublishedSnapshot struct {
	Snapshot PublicModelCatalogSnapshot          `json:"snapshot"`
	Details  map[string]PublicModelCatalogDetail `json:"details,omitempty"`
}

type PublicModelCatalogPublishedSummary struct {
	ETag              string `json:"etag"`
	UpdatedAt         string `json:"updated_at"`
	PublishedAt       string `json:"published_at,omitempty"`
	LastRevalidatedAt string `json:"last_revalidated_at,omitempty"`
	StaleReason       string `json:"stale_reason,omitempty"`
	PageSize          int    `json:"page_size"`
	ModelCount        int    `json:"model_count"`
}

type PublicModelCatalogDraftPayload struct {
	Draft              PublicModelCatalogDraft             `json:"draft"`
	AvailableItems     []PublicModelCatalogItem            `json:"available_items"`
	AvailableEntries   []PublicModelCatalogItem            `json:"available_entries"`
	AvailableUpdatedAt string                              `json:"available_updated_at,omitempty"`
	AvailableSource    string                              `json:"available_source,omitempty"`
	Published          *PublicModelCatalogPublishedSummary `json:"published,omitempty"`
	Revalidation       PublicModelCatalogRevalidationState `json:"revalidation"`
}

type PublicModelCatalogRevalidationState struct {
	AutoEnabled bool `json:"auto_enabled"`
}

type PublicModelCatalogRevalidationInput struct {
	AutoEnabled *bool `json:"auto_enabled,omitempty"`
}

type PublicModelCatalogRevalidationResult struct {
	Published  PublicModelCatalogPublishedSummary `json:"published"`
	CheckedAt  string                             `json:"checked_at"`
	ModelCount int                                `json:"model_count"`
	StaleCount int                                `json:"stale_count"`
	Reasons    map[string]int                     `json:"reasons,omitempty"`
}

const (
	PublicModelPublicationStatusPublished = "published"

	PublicModelVerificationSourcePublishedSnapshot = "published_snapshot"
	PublicModelVerificationSourceLiveFallback      = "live_fallback"

	PublicModelKeyAvailabilityAvailable   = "available"
	PublicModelKeyAvailabilityUnavailable = "unavailable"

	PublicModelUnavailableReasonNotSelectedByKey           = "not_selected_by_key"
	PublicModelUnavailableReasonGroupUnavailable           = "group_unavailable"
	PublicModelUnavailableReasonImageOnlyKeyRestricted     = "image_only_key_restricted"
	PublicModelUnavailableReasonPublishedSourceUnavailable = "published_source_unavailable"
)

const (
	PublicModelHealthStatusHealthy = "healthy"
	PublicModelHealthStatusWarning = "warning"
	PublicModelHealthStatusError   = "error"
	PublicModelHealthStatusPending = "pending"
)

const (
	PublicModelHealthSourceTraffic = "traffic"
	PublicModelHealthSourceProbe   = "probe"
	PublicModelHealthSourceNone    = "none"

	PublicModelHealthReasonTrafficRecent   = "traffic_recent"
	PublicModelHealthReasonProbeRecent     = "probe_recent"
	PublicModelHealthReasonMonitorDisabled = "monitor_disabled"
	PublicModelHealthReasonNoHistory       = "no_history"
	PublicModelHealthReasonStaleHistory    = "stale_history"
	PublicModelHealthReasonChecking        = "checking"
)

type PublicModelCatalogStatusSnapshot struct {
	UpdatedAt string                         `json:"updated_at"`
	Items     []PublicModelCatalogStatusItem `json:"items"`
}

type PublicModelCatalogStatusItem struct {
	PublicModelID    string                              `json:"public_model_id"`
	Model            string                              `json:"model"`
	Aliases          []string                            `json:"aliases"`
	Status           string                              `json:"-"`
	HealthStatus     string                              `json:"health_status"`
	HealthSource     string                              `json:"health_source"`
	StatusReason     string                              `json:"status_reason"`
	SuccessRateToday *float64                            `json:"success_rate_today,omitempty"`
	SuccessRate7d    *float64                            `json:"success_rate_7d,omitempty"`
	LatencyMs        *int64                              `json:"latency_ms,omitempty"`
	LastCheckedAt    string                              `json:"last_checked_at,omitempty"`
	Daily            []PublicModelCatalogDailyStatus     `json:"daily"`
	Trend            []PublicModelCatalogTrendPoint      `json:"trend"`
	RateLimit        *PublicModelCatalogRateLimitSummary `json:"rate_limit,omitempty"`
}

type PublicModelCatalogDailyStatus struct {
	Date        string   `json:"date"`
	Status      string   `json:"status"`
	SuccessRate *float64 `json:"success_rate,omitempty"`
	LatencyMs   *int64   `json:"latency_ms,omitempty"`
}

type PublicModelCatalogTrendPoint struct {
	Timestamp   string   `json:"timestamp"`
	SuccessRate *float64 `json:"success_rate,omitempty"`
	LatencyMs   *int64   `json:"latency_ms,omitempty"`
}

type PublicModelCatalogRateLimitSummary struct {
	RPM *int64 `json:"rpm,omitempty"`
	TPM *int64 `json:"tpm,omitempty"`
	RPD *int64 `json:"rpd,omitempty"`
}

type PublicModelCatalogCapacityDiagnosticsSnapshot struct {
	UpdatedAt string                                       `json:"updated_at"`
	Items     []PublicModelCatalogCapacityDiagnosticItem   `json:"items"`
	Summary   PublicModelCatalogCapacityDiagnosticsSummary `json:"summary"`
}

type PublicModelCatalogCapacityDiagnosticsSummary struct {
	ModelCount           int            `json:"model_count"`
	AvailableCount       int            `json:"available_count"`
	LimitedCount         int            `json:"limited_count"`
	UnschedulableCount   int            `json:"unschedulable_count"`
	SourceCounts         map[string]int `json:"source_counts,omitempty"`
	RestrictionCounts    map[string]int `json:"restriction_counts,omitempty"`
	EffectiveLimitCounts map[string]int `json:"effective_limit_counts,omitempty"`
}

type PublicModelCatalogCapacityDiagnosticItem struct {
	PublicModelID      string                                       `json:"public_model_id"`
	Model              string                                       `json:"model"`
	EntryID            string                                       `json:"entry_id,omitempty"`
	Provider           string                                       `json:"provider,omitempty"`
	SourceProtocol     string                                       `json:"source_protocol,omitempty"`
	SourceAccountID    int64                                        `json:"source_account_id,omitempty"`
	BindingGroupID     int64                                        `json:"binding_group_id,omitempty"`
	Scope              string                                       `json:"scope,omitempty"`
	Availability       string                                       `json:"availability"`
	EffectiveRateLimit *PublicModelCatalogRateLimitSummary          `json:"effective_rate_limit,omitempty"`
	Restrictions       []PublicModelCatalogCapacityRestriction      `json:"restrictions,omitempty"`
	Sources            []PublicModelCatalogCapacityDiagnosticSource `json:"sources,omitempty"`
}

type PublicModelCatalogCapacityRestriction struct {
	Kind    string   `json:"kind"`
	Scope   string   `json:"scope,omitempty"`
	Message string   `json:"message,omitempty"`
	Until   string   `json:"until,omitempty"`
	Limit   *float64 `json:"limit,omitempty"`
	Used    *float64 `json:"used,omitempty"`
}

type PublicModelCatalogCapacityDiagnosticSource struct {
	Source string `json:"source"`
	Scope  string `json:"scope,omitempty"`
	Detail string `json:"detail,omitempty"`
}
