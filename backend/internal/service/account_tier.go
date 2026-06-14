package service

import "strings"

const (
	AccountExtraKeyAccountTier = "account_tier"

	OpenAIAccountTierPro20x = "pro_20x"
	OpenAIAccountTierPro5x  = "pro_5x"
	OpenAIAccountTierPlus   = "plus"
	OpenAIAccountTierTeam   = "team"
	OpenAIAccountTierFree   = "free"

	ClaudeAccountTierMax20x = "max_20x"
	ClaudeAccountTierMax5x  = "max_5x"
	ClaudeAccountTierPro    = "pro"
)

func NormalizeOpenAIAccountTier(value string) string {
	switch normalizeAccountTierToken(value) {
	case "pro20x":
		return OpenAIAccountTierPro20x
	case "pro5x":
		return OpenAIAccountTierPro5x
	case "plus":
		return OpenAIAccountTierPlus
	case "team":
		return OpenAIAccountTierTeam
	case "free":
		return OpenAIAccountTierFree
	default:
		return ""
	}
}

func NormalizeClaudeAccountTier(value string) string {
	switch normalizeAccountTierToken(value) {
	case "max20x":
		return ClaudeAccountTierMax20x
	case "max5x":
		return ClaudeAccountTierMax5x
	case "pro":
		return ClaudeAccountTierPro
	default:
		return ""
	}
}

func NormalizeAccountTier(platform string, value string) string {
	switch {
	case IsOpenAIFamily(platform):
		return NormalizeOpenAIAccountTier(value)
	case IsAnthropicFamily(platform):
		return NormalizeClaudeAccountTier(value)
	default:
		return ""
	}
}

func DefaultAccountTierConcurrency(platform string, tier string) int {
	switch NormalizeAccountTier(platform, tier) {
	case OpenAIAccountTierPro20x, ClaudeAccountTierMax20x:
		return 10
	case OpenAIAccountTierPro5x, ClaudeAccountTierMax5x:
		return 5
	case OpenAIAccountTierPlus, OpenAIAccountTierTeam, ClaudeAccountTierPro:
		return 2
	case OpenAIAccountTierFree:
		return 1
	default:
		return 0
	}
}

func ApplyAccountTierDefaults(platform string, accountType string, extra map[string]any, concurrency int) (map[string]any, int) {
	nextExtra := NormalizeAccountTierExtra(platform, accountType, extra)
	if concurrency > 0 {
		return nextExtra, concurrency
	}
	tier := AccountTierFromExtra(platform, nextExtra)
	if capacity := DefaultAccountTierConcurrency(platform, tier); capacity > 0 {
		concurrency = capacity
	}
	return nextExtra, concurrency
}

func NormalizeAccountTierExtra(platform string, accountType string, extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return nil
	}
	nextExtra := cloneStringAnyMap(extra)
	tier := NormalizeAccountTier(platform, stringAny(nextExtra[AccountExtraKeyAccountTier]))
	if tier == "" {
		delete(nextExtra, AccountExtraKeyAccountTier)
		return emptyMapToNil(nextExtra)
	}
	nextExtra[AccountExtraKeyAccountTier] = tier
	if IsOpenAIFamily(platform) && strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeOAuth && tier == OpenAIAccountTierFree {
		if NormalizeOpenAIImageProtocolMode(stringAny(nextExtra[openAIImageProtocolModeExtraKey])) == "" {
			nextExtra[openAIImageProtocolModeExtraKey] = OpenAIImageProtocolModeNative
		}
		if _, exists := nextExtra[openAIImageCompatAllowedExtraKey]; !exists {
			nextExtra[openAIImageCompatAllowedExtraKey] = false
		}
	}
	return emptyMapToNil(nextExtra)
}

func AccountTierFromExtra(platform string, extra map[string]any) string {
	if len(extra) == 0 {
		return ""
	}
	return NormalizeAccountTier(platform, stringAny(extra[AccountExtraKeyAccountTier]))
}

func normalizeAccountTierToken(value string) string {
	token := strings.ToLower(strings.TrimSpace(value))
	token = strings.ReplaceAll(token, "-", "")
	token = strings.ReplaceAll(token, "_", "")
	token = strings.ReplaceAll(token, " ", "")
	return token
}
