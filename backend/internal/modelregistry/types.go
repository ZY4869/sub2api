package modelregistry

type ModelEntry struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"display_name"`
	Provider         string   `json:"provider"`
	Platforms        []string `json:"platforms"`
	ProtocolIDs      []string `json:"protocol_ids"`
	Aliases          []string `json:"aliases"`
	PricingLookupIDs []string `json:"pricing_lookup_ids"`
	Modalities       []string `json:"modalities"`
	Capabilities     []string `json:"capabilities"`
	UIPriority       int      `json:"ui_priority"`
	ExposedIn        []string `json:"exposed_in"`
}

type PresetMapping struct {
	Platform string `json:"platform"`
	Label    string `json:"label"`
	From     string `json:"from"`
	To       string `json:"to"`
	Color    string `json:"color"`
	Order    int    `json:"order,omitempty"`
}

type PublicSnapshot struct {
	ETag      string          `json:"etag"`
	UpdatedAt string          `json:"updated_at"`
	Models    []ModelEntry    `json:"models"`
	Presets   []PresetMapping `json:"presets"`
}

type AdminModelDetail struct {
	ModelEntry
	Source     string `json:"source"`
	Hidden     bool   `json:"hidden"`
	Tombstoned bool   `json:"tombstoned"`
}
