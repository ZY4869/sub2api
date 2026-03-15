package modelregistry

type ModelEntry struct {
	ID                   string            `json:"id"`
	DisplayName          string            `json:"display_name"`
	Provider             string            `json:"provider"`
	Platforms            []string          `json:"platforms"`
	ProtocolIDs          []string          `json:"protocol_ids"`
	Aliases              []string          `json:"aliases"`
	PricingLookupIDs     []string          `json:"pricing_lookup_ids"`
	PreferredProtocolIDs map[string]string `json:"preferred_protocol_ids,omitempty"`
	Modalities           []string          `json:"modalities"`
	Capabilities         []string          `json:"capabilities"`
	UIPriority           int               `json:"ui_priority"`
	ExposedIn            []string          `json:"exposed_in"`
	Status               string            `json:"status,omitempty"`
	DeprecatedAt         string            `json:"deprecated_at,omitempty"`
	ReplacedBy           string            `json:"replaced_by,omitempty"`
	DeprecationNotice    string            `json:"deprecation_notice,omitempty"`
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
	Available  bool   `json:"available"`
}

type Resolution struct {
	Input            string      `json:"input"`
	NormalizedInput  string      `json:"normalized_input"`
	CanonicalID      string      `json:"canonical_id"`
	EffectiveID      string      `json:"effective_id"`
	PricingID        string      `json:"pricing_id,omitempty"`
	MatchedBy        string      `json:"matched_by,omitempty"`
	MatchedValue     string      `json:"matched_value,omitempty"`
	Route            string      `json:"route,omitempty"`
	RouteProtocolID  string      `json:"route_protocol_id,omitempty"`
	Deprecated       bool        `json:"deprecated"`
	Entry            ModelEntry  `json:"entry"`
	ReplacementEntry *ModelEntry `json:"replacement_entry,omitempty"`
}
