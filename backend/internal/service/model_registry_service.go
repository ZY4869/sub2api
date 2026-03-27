package service

type ModelRegistryListFilter struct {
	Search            string
	Provider          string
	Platform          string
	Exposure          string
	Status            string
	Availability      string
	SortMode          string
	IncludeHidden     bool
	IncludeTombstoned bool
	Page              int
	PageSize          int
}

type ModelRegistryProviderSummary struct {
	Provider       string `json:"provider"`
	TotalCount     int    `json:"total_count"`
	AvailableCount int    `json:"available_count"`
}

type UpsertModelRegistryEntryInput struct {
	ID                   string            `json:"id"`
	DisplayName          string            `json:"display_name"`
	Provider             string            `json:"provider"`
	Platforms            []string          `json:"platforms"`
	ProtocolIDs          []string          `json:"protocol_ids"`
	Aliases              []string          `json:"aliases"`
	PricingLookupIDs     []string          `json:"pricing_lookup_ids"`
	PreferredProtocolIDs map[string]string `json:"preferred_protocol_ids"`
	Modalities           []string          `json:"modalities"`
	Capabilities         []string          `json:"capabilities"`
	UIPriority           int               `json:"ui_priority"`
	ExposedIn            []string          `json:"exposed_in"`
	Status               string            `json:"status"`
	DeprecatedAt         string            `json:"deprecated_at"`
	ReplacedBy           string            `json:"replaced_by"`
	DeprecationNotice    string            `json:"deprecation_notice"`
}

type UpdateModelRegistryVisibilityInput struct {
	Model  string `json:"model"`
	Hidden bool   `json:"hidden"`
}

type BatchSyncModelRegistryExposuresInput struct {
	Models    []string `json:"models"`
	Exposures []string `json:"exposures"`
	Mode      string   `json:"mode"`
}

type MoveModelRegistryProviderInput struct {
	Models         []string `json:"models"`
	TargetProvider string   `json:"target_provider"`
}

type UpdateModelRegistryAvailabilityInput struct {
	Models []string `json:"models"`
}

type BatchHardDeleteModelRegistryInput struct {
	Models []string `json:"models"`
}

type ModelRegistryExposureSyncFailure struct {
	Model string `json:"model"`
	Error string `json:"error"`
}

type BatchSyncModelRegistryExposuresResult struct {
	Exposures     []string                           `json:"exposures"`
	Mode          string                             `json:"mode"`
	UpdatedCount  int                                `json:"updated_count"`
	SkippedCount  int                                `json:"skipped_count"`
	FailedCount   int                                `json:"failed_count"`
	UpdatedModels []string                           `json:"updated_models"`
	SkippedModels []string                           `json:"skipped_models,omitempty"`
	FailedModels  []ModelRegistryExposureSyncFailure `json:"failed_models,omitempty"`
}

type ModelRegistryProviderMoveFailure struct {
	Model string `json:"model"`
	Error string `json:"error"`
}

type MoveModelRegistryProviderResult struct {
	UpdatedCount  int                              `json:"updated_count"`
	SkippedCount  int                              `json:"skipped_count"`
	FailedCount   int                              `json:"failed_count"`
	UpdatedModels []string                         `json:"updated_models"`
	SkippedModels []string                         `json:"skipped_models,omitempty"`
	FailedModels  []ModelRegistryProviderMoveFailure `json:"failed_models,omitempty"`
}

type UpsertDiscoveredEntryInput struct {
	ModelID        string
	SourcePlatform string
}

type UpsertDiscoveredEntryResult struct {
	RegistryModelID string
	CanonicalModel  string
	Changed         bool
	Existing        bool
	Blocked         bool
}

type ModelRegistryService struct {
	settingRepo SettingRepository
	accountRepo AccountRepository
}

var modelRegistryCapabilityOrder = []string{
	"text",
	"vision",
	"image_generation",
	"web_search",
	"audio_understanding",
	"video_understanding",
	"audio_generation",
	"video_generation",
}

var modelRegistryCapabilityAliases = map[string]string{
	"reasoning": "text",
	"image":     "image_generation",
	"video":     "video_generation",
	"audio":     "audio_understanding",
	"web":       "web_search",
}

func NewModelRegistryService(settingRepo SettingRepository) *ModelRegistryService {
	return &ModelRegistryService{settingRepo: settingRepo}
}

func (s *ModelRegistryService) SetAccountRepository(accountRepo AccountRepository) {
	s.accountRepo = accountRepo
}
