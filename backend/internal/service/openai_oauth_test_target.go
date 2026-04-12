package service

import (
	"context"
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
