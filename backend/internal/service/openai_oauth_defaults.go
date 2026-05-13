package service

var (
	openAIOAuthFreeDefaultModelIDs = []string{
		"gpt-5.2",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.5",
	}
	openAIOAuthPaidDefaultModelIDs = []string{
		"gpt-image-2",
		"gpt-5.2",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.5",
	}
)

const openAIOAuthProSparkModelID = "gpt-5.3-codex-spark"

func ResolveOpenAIOAuthDefaultAllowedModels(planType string, proMultiplier int) []string {
	normalizedPlanType := normalizeOpenAIPlanType(planType)
	if normalizedPlanType == "free" {
		return append([]string(nil), openAIOAuthFreeDefaultModelIDs...)
	}

	next := append([]string(nil), openAIOAuthPaidDefaultModelIDs...)
	if shouldEnableOpenAIOAuthProSpark(normalizedPlanType, proMultiplier) {
		next = append(next, openAIOAuthProSparkModelID)
	}
	return next
}

func BuildOpenAIOAuthDefaultModelScopeExtra(baseExtra map[string]any, planType string, proMultiplier int) map[string]any {
	allowedModels := ResolveOpenAIOAuthDefaultAllowedModels(planType, proMultiplier)
	if len(allowedModels) == 0 {
		return MergeStringAnyMap(nil, baseExtra)
	}

	out := MergeStringAnyMap(nil, baseExtra)
	if out == nil {
		out = make(map[string]any, 1)
	}
	out["model_scope_v2"] = buildOpenAIWhitelistScopeMap(allowedModels)
	return out
}

func shouldEnableOpenAIOAuthProSpark(planType string, proMultiplier int) bool {
	if normalizeOpenAIPlanType(planType) != "pro" {
		return false
	}
	if proMultiplier > 0 {
		return true
	}
	return true
}
