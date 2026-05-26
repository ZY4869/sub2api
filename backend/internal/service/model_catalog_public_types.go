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

type PublicModelCatalogSnapshot struct {
	ETag          string                   `json:"etag"`
	UpdatedAt     string                   `json:"updated_at"`
	PageSize      int                      `json:"page_size,omitempty"`
	CatalogSource string                   `json:"catalog_source,omitempty"`
	Items         []PublicModelCatalogItem `json:"items"`
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
	LifecycleStatus      string                              `json:"lifecycle_status,omitempty"`
	RequestProtocols     []string                            `json:"request_protocols,omitempty"`
	SourceIDs            []string                            `json:"source_ids,omitempty"`
	Mode                 string                              `json:"mode,omitempty"`
	Currency             string                              `json:"currency"`
	PriceDisplay         PublicModelCatalogPriceDisplay      `json:"price_display"`
	OfficialPriceDisplay PublicModelCatalogPriceDisplay      `json:"official_price_display,omitempty"`
	SalePriceDisplay     PublicModelCatalogPriceDisplay      `json:"sale_price_display,omitempty"`
	MultiplierSummary    PublicModelCatalogMultiplierSummary `json:"multiplier_summary"`
	RuntimePriceSpec     PublicModelCatalogRuntimePriceSpec  `json:"runtime_price_spec,omitempty"`
}

type PublicModelCatalogPriceDisplay struct {
	Primary   []PublicModelCatalogPriceEntry `json:"primary"`
	Secondary []PublicModelCatalogPriceEntry `json:"secondary,omitempty"`
}

type PublicModelCatalogPriceEntry struct {
	ID    string  `json:"id"`
	Unit  string  `json:"unit,omitempty"`
	Value float64 `json:"value"`
}

type PublicModelCatalogMultiplierSummary struct {
	Enabled bool     `json:"enabled"`
	Kind    string   `json:"kind"`
	Mode    string   `json:"mode,omitempty"`
	Value   *float64 `json:"value,omitempty"`
}

type PublicModelCatalogRuntimePriceSpec struct {
	Currency                        string  `json:"currency,omitempty"`
	OutputChargeSlot                string  `json:"output_charge_slot,omitempty"`
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
	ETag       string `json:"etag"`
	UpdatedAt  string `json:"updated_at"`
	PageSize   int    `json:"page_size"`
	ModelCount int    `json:"model_count"`
}

type PublicModelCatalogDraftPayload struct {
	Draft              PublicModelCatalogDraft             `json:"draft"`
	AvailableItems     []PublicModelCatalogItem            `json:"available_items"`
	AvailableEntries   []PublicModelCatalogItem            `json:"available_entries"`
	AvailableUpdatedAt string                              `json:"available_updated_at,omitempty"`
	AvailableSource    string                              `json:"available_source,omitempty"`
	Published          *PublicModelCatalogPublishedSummary `json:"published,omitempty"`
}
