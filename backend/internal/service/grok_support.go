package service

import "strings"

const (
	GrokTierBasic = "basic"
	GrokTierSuper = "super"
	GrokTierHeavy = "heavy"

	GrokDetectedKindSSO        = AccountTypeSSO
	GrokDetectedKindAPIKey     = AccountTypeAPIKey
	GrokDetectedKindLegacyPool = "legacy_pool"
)

type GrokCapabilities struct {
	AllowHeavyModel         bool   `json:"allow_heavy_model"`
	VideoMaxResolution      string `json:"video_max_resolution"`
	VideoMaxDurationSeconds int    `json:"video_max_duration_seconds"`
}

var grokSharedModelIDs = []string{
	"grok-3-beta",
	"grok-3-mini-beta",
	"grok-3-fast-beta",
	"grok-2",
	"grok-2-vision",
	"grok-2-image",
	"grok-beta",
	"grok-vision-beta",
}

var grokHeavyOnlyModelIDs = []string{
	"grok-4",
	"grok-4-0709",
}

func NormalizeGrokTierValue(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case GrokTierBasic:
		return GrokTierBasic
	case GrokTierSuper:
		return GrokTierSuper
	case GrokTierHeavy:
		return GrokTierHeavy
	default:
		return ""
	}
}

func ResolveGrokTier(extra map[string]any) string {
	if len(extra) > 0 {
		if rawTier, _ := extra["grok_tier"].(string); NormalizeGrokTierValue(rawTier) != "" {
			return NormalizeGrokTierValue(rawTier)
		}
	}
	return GrokTierBasic
}

func DefaultGrokPriorityForTier(tier string) int {
	switch NormalizeGrokTierValue(tier) {
	case GrokTierHeavy:
		return 30
	case GrokTierSuper:
		return 40
	default:
		return 50
	}
}

func DefaultGrokCapabilitiesForTier(tier string) GrokCapabilities {
	switch NormalizeGrokTierValue(tier) {
	case GrokTierHeavy:
		return GrokCapabilities{
			AllowHeavyModel:         true,
			VideoMaxResolution:      "720p",
			VideoMaxDurationSeconds: 15,
		}
	case GrokTierSuper:
		return GrokCapabilities{
			AllowHeavyModel:         false,
			VideoMaxResolution:      "720p",
			VideoMaxDurationSeconds: 15,
		}
	default:
		return GrokCapabilities{
			AllowHeavyModel:         false,
			VideoMaxResolution:      "480p",
			VideoMaxDurationSeconds: 6,
		}
	}
}

func (c GrokCapabilities) ToMap() map[string]any {
	return map[string]any{
		"allow_heavy_model":          c.AllowHeavyModel,
		"video_max_resolution":       c.VideoMaxResolution,
		"video_max_duration_seconds": c.VideoMaxDurationSeconds,
	}
}

func ResolveGrokCapabilities(extra map[string]any) GrokCapabilities {
	result := DefaultGrokCapabilitiesForTier(ResolveGrokTier(extra))
	if len(extra) == 0 {
		return result
	}
	rawCapabilities, _ := extra["grok_capabilities"].(map[string]any)
	if len(rawCapabilities) == 0 {
		return result
	}
	if value, ok := rawCapabilities["allow_heavy_model"]; ok {
		result.AllowHeavyModel = parseGrokBool(value, result.AllowHeavyModel)
	}
	if value, ok := rawCapabilities["video_max_resolution"].(string); ok && strings.TrimSpace(value) != "" {
		result.VideoMaxResolution = strings.TrimSpace(value)
	}
	if value, ok := rawCapabilities["video_max_duration_seconds"]; ok {
		if parsed := ParseExtraInt(value); parsed > 0 {
			result.VideoMaxDurationSeconds = parsed
		}
	}
	return result
}

func DefaultGrokModelIDsForTier(tier string) []string {
	models := make([]string, 0, len(grokSharedModelIDs)+len(grokHeavyOnlyModelIDs))
	models = append(models, grokSharedModelIDs...)
	if NormalizeGrokTierValue(tier) == GrokTierHeavy {
		models = append(models, grokHeavyOnlyModelIDs...)
	}
	return models
}

func DefaultGrokModelMappingForTier(tier string) map[string]any {
	models := DefaultGrokModelIDsForTier(tier)
	mapping := make(map[string]any, len(models))
	for _, modelID := range models {
		mapping[modelID] = modelID
	}
	return mapping
}

func IsGrokHeavyModel(modelID string) bool {
	switch strings.TrimSpace(strings.ToLower(modelID)) {
	case "grok-4", "grok-4-0709":
		return true
	default:
		return false
	}
}

func NormalizeGrokCredentialValue(kind string, value string) string {
	normalized := strings.TrimSpace(value)
	if strings.TrimSpace(strings.ToLower(kind)) == GrokDetectedKindSSO {
		lower := strings.ToLower(normalized)
		if strings.HasPrefix(lower, "bearer ") {
			normalized = strings.TrimSpace(normalized[7:])
		}
	}
	return normalized
}

func InferGrokCredentialKind(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(normalized, "xai-"), strings.HasPrefix(normalized, "sk-"):
		return GrokDetectedKindAPIKey
	default:
		return GrokDetectedKindSSO
	}
}

func MaskGrokCredentialValue(kind string, value string) string {
	normalized := NormalizeGrokCredentialValue(kind, value)
	if normalized == "" {
		return ""
	}
	if len(normalized) <= 8 {
		return strings.Repeat("*", len(normalized))
	}
	prefixLen := 4
	if strings.TrimSpace(strings.ToLower(kind)) == GrokDetectedKindAPIKey {
		prefixLen = 6
	}
	if prefixLen > len(normalized)-4 {
		prefixLen = len(normalized) / 2
	}
	return normalized[:prefixLen] + "..." + normalized[len(normalized)-4:]
}

func LegacyGrokPoolTier(poolName string) string {
	normalized := strings.TrimSpace(strings.ToLower(poolName))
	switch {
	case strings.Contains(normalized, GrokTierHeavy):
		return GrokTierHeavy
	case strings.Contains(normalized, GrokTierSuper):
		return GrokTierSuper
	default:
		return GrokTierBasic
	}
}

func parseGrokBool(value any, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		switch strings.TrimSpace(strings.ToLower(v)) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	return fallback
}
