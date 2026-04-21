package service

const (
	PublicModelCatalogSourcePublished    = "published"
	PublicModelCatalogSourceLiveFallback = "live_fallback"
)

type PublicModelCatalogSnapshot struct {
	ETag          string                   `json:"etag"`
	UpdatedAt     string                   `json:"updated_at"`
	PageSize      int                      `json:"page_size,omitempty"`
	CatalogSource string                   `json:"catalog_source,omitempty"`
	Items         []PublicModelCatalogItem `json:"items"`
}

type PublicModelCatalogItem struct {
	Model             string                              `json:"model"`
	DisplayName       string                              `json:"display_name,omitempty"`
	Provider          string                              `json:"provider,omitempty"`
	ProviderIconKey   string                              `json:"provider_icon_key,omitempty"`
	RequestProtocols  []string                            `json:"request_protocols,omitempty"`
	SourceIDs         []string                            `json:"source_ids,omitempty"`
	Mode              string                              `json:"mode,omitempty"`
	Currency          string                              `json:"currency"`
	PriceDisplay      PublicModelCatalogPriceDisplay      `json:"price_display"`
	MultiplierSummary PublicModelCatalogMultiplierSummary `json:"multiplier_summary"`
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

type PublicModelCatalogDetail struct {
	Item              PublicModelCatalogItem `json:"item"`
	CatalogSource     string                 `json:"catalog_source,omitempty"`
	ExampleSource     string                 `json:"example_source,omitempty"`
	ExampleProtocol   string                 `json:"example_protocol,omitempty"`
	ExamplePageID     string                 `json:"example_page_id,omitempty"`
	ExampleMarkdown   string                 `json:"example_markdown,omitempty"`
	ExampleOverrideID string                 `json:"example_override_id,omitempty"`
}

type PublicModelCatalogDraft struct {
	SelectedModels []string `json:"selected_models,omitempty"`
	PageSize       int      `json:"page_size,omitempty"`
	UpdatedAt      string   `json:"updated_at,omitempty"`
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
	Draft          PublicModelCatalogDraft             `json:"draft"`
	AvailableItems []PublicModelCatalogItem            `json:"available_items"`
	Published      *PublicModelCatalogPublishedSummary `json:"published,omitempty"`
}
