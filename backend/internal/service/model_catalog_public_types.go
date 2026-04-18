package service

type PublicModelCatalogSnapshot struct {
	ETag      string                   `json:"etag"`
	UpdatedAt string                   `json:"updated_at"`
	Items     []PublicModelCatalogItem `json:"items"`
}

type PublicModelCatalogItem struct {
	Model             string                              `json:"model"`
	DisplayName       string                              `json:"display_name,omitempty"`
	Provider          string                              `json:"provider,omitempty"`
	ProviderIconKey   string                              `json:"provider_icon_key,omitempty"`
	RequestProtocols  []string                            `json:"request_protocols,omitempty"`
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
