package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const OpenAIOAuthDefaultTestModelID = "gpt-5.4"

func EnsureOpenAIOAuthTestTargetExtra(extra map[string]any) map[string]any {
	provider := NormalizeModelProvider(stringValueFromAny(extra[gatewayExtraTestProviderKey]))
	modelID := strings.TrimSpace(stringValueFromAny(extra[gatewayExtraTestModelIDKey]))
	if provider != "" && modelID != "" {
		return MergeStringAnyMap(nil, extra)
	}

	out := MergeStringAnyMap(nil, extra)
	if out == nil {
		out = make(map[string]any, 2)
	}
	if provider == "" {
		out[gatewayExtraTestProviderKey] = PlatformOpenAI
	}
	if modelID == "" {
		out[gatewayExtraTestModelIDKey] = OpenAIOAuthDefaultTestModelID
	}
	return out
}

func defaultOpenAIOAuthTestModelID(ctx context.Context, account *Account, registry *ModelRegistryService) string {
	if account == nil || !account.IsOpenAIOAuth() {
		return openai.DefaultTestModel
	}

	knownModels := normalizeStringSliceAny(account.Extra["openai_known_models"], NormalizeModelCatalogModelID)
	if len(knownModels) == 0 {
		return OpenAIOAuthDefaultTestModelID
	}

	knownSet := make(map[string]struct{}, len(knownModels))
	for _, modelID := range knownModels {
		knownSet[NormalizeModelCatalogModelID(modelID)] = struct{}{}
	}
	if _, ok := knownSet[NormalizeModelCatalogModelID(OpenAIOAuthDefaultTestModelID)]; ok {
		return OpenAIOAuthDefaultTestModelID
	}

	for _, candidate := range BuildAvailableTestModels(ctx, account, registry) {
		if NormalizeModelProvider(candidate.Provider) != PlatformOpenAI {
			continue
		}
		canonicalID := NormalizeModelCatalogModelID(candidate.CanonicalID)
		modelID := NormalizeModelCatalogModelID(candidate.ID)
		if canonicalID != "" {
			if _, ok := knownSet[canonicalID]; ok {
				return canonicalID
			}
		}
		if modelID != "" {
			if _, ok := knownSet[modelID]; ok {
				return modelID
			}
		}
	}

	return knownModels[0]
}

func resolveOpenAITestModelID(ctx context.Context, account *Account, requestedModelID string, registry *ModelRegistryService) string {
	requestedModelID = strings.TrimSpace(requestedModelID)
	if requestedModelID == "" {
		return defaultOpenAIOAuthTestModelID(ctx, account, registry)
	}
	if account == nil || !account.IsOpenAIOAuth() || !isChatGPTOpenAIOAuthAccount(account) {
		return requestedModelID
	}
	if isChatGPTOpenAITestModelAllowed(ctx, account, requestedModelID, registry) {
		return requestedModelID
	}

	fallbackModelID := defaultOpenAIOAuthTestModelID(ctx, account, registry)
	if fallbackModelID == "" {
		return requestedModelID
	}
	if !strings.EqualFold(strings.TrimSpace(fallbackModelID), requestedModelID) {
		slog.Warn(
			"openai_oauth_test_model_fallback",
			"account_id", account.ID,
			"requested_model_id", requestedModelID,
			"fallback_model_id", fallbackModelID,
		)
	}
	return fallbackModelID
}

func isChatGPTOpenAITestModelAllowed(ctx context.Context, account *Account, requestedModelID string, registry *ModelRegistryService) bool {
	requestedModelID = strings.TrimSpace(requestedModelID)
	if requestedModelID == "" {
		return false
	}
	if !isChatGPTOpenAIOAuthAccount(account) {
		return true
	}

	lookupIDs := openAIOAuthTestModelLookupIDs(ctx, requestedModelID, registry)
	if len(lookupIDs) == 0 {
		return false
	}
	for _, modelID := range lookupIDs {
		if isChatGPTOpenAIUnsupportedTestModelID(modelID) {
			return false
		}
	}

	knownModels := normalizeStringSliceAny(account.Extra["openai_known_models"], NormalizeModelCatalogModelID)
	if len(knownModels) == 0 {
		return true
	}

	knownSet := make(map[string]struct{}, len(knownModels))
	for _, modelID := range knownModels {
		knownSet[modelID] = struct{}{}
	}
	for _, modelID := range lookupIDs {
		if _, ok := knownSet[modelID]; ok {
			return true
		}
	}
	return false
}

func openAIOAuthTestModelLookupIDs(ctx context.Context, modelID string, registry *ModelRegistryService) []string {
	appendUnique := func(result []string, seen map[string]struct{}, value string) []string {
		normalized := NormalizeModelCatalogModelID(value)
		if normalized == "" {
			return result
		}
		if _, ok := seen[normalized]; ok {
			return result
		}
		seen[normalized] = struct{}{}
		return append(result, normalized)
	}

	seen := map[string]struct{}{}
	result := appendUnique(nil, seen, modelID)
	if registry == nil {
		return result
	}
	resolution, err := registry.ExplainResolution(ctx, modelID)
	if err != nil || resolution == nil {
		return result
	}
	result = appendUnique(result, seen, resolution.Entry.ID)
	result = appendUnique(result, seen, resolution.CanonicalID)
	result = appendUnique(result, seen, resolution.EffectiveID)
	if resolution.ReplacementEntry != nil {
		result = appendUnique(result, seen, resolution.ReplacementEntry.ID)
	}
	return result
}

func isChatGPTOpenAIUnsupportedTestModelID(modelID string) bool {
	switch NormalizeModelCatalogModelID(modelID) {
	case "gpt-5.1-codex-mini":
		return true
	}

	switch strings.TrimSpace(strings.ToLower(modelID)) {
	case "gpt-5.1-codex-mini", "codex-mini-latest", "gpt-5-codex-mini":
		return true
	default:
		return false
	}
}
