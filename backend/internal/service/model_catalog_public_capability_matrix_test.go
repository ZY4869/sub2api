package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDedupePublicModelProtocolEndpoints_PrefersRuntimeObservedUnsupported(t *testing.T) {
	endpoints := dedupePublicModelProtocolEndpoints([]PublicModelProtocolEndpoint{
		{
			Key:      "openai.responses",
			Protocol: PlatformOpenAI,
			Support:  PublicModelSupportSupported,
			Source:   PublicModelCapabilitySourceOfficialRegistry,
		},
		{
			Key:           "openai.responses",
			Protocol:      PlatformOpenAI,
			Support:       PublicModelSupportUnsupported,
			Source:        PublicModelCapabilitySourceRuntimeObserved,
			Verified:      true,
			LastCheckedAt: "2026-05-30T00:00:00Z",
		},
	})

	require.Len(t, endpoints, 1)
	require.Equal(t, PublicModelSupportUnsupported, endpoints[0].Support)
	require.Equal(t, PublicModelCapabilitySourceRuntimeObserved, endpoints[0].Source)
	require.True(t, endpoints[0].Verified)
}

func TestDedupePublicModelCapabilityMatrix_ExcludesUnsupportedFromLegacySummary(t *testing.T) {
	matrix := dedupePublicModelCapabilityMatrix([]PublicModelCapabilityMatrixEntry{
		{
			Capability: "text",
			Protocol:   PlatformOpenAI,
			Endpoint:   "openai.responses",
			Support:    PublicModelSupportSupported,
			Source:     PublicModelCapabilitySourceManualConfig,
		},
		{
			Capability:    "text",
			Protocol:      PlatformOpenAI,
			Endpoint:      "openai.responses",
			Support:       PublicModelSupportUnsupported,
			Source:        PublicModelCapabilitySourceRuntimeObserved,
			Verified:      true,
			LastCheckedAt: "2026-05-30T00:00:00Z",
		},
	})

	require.Len(t, matrix, 1)
	require.Equal(t, PublicModelSupportUnsupported, matrix[0].Support)
	require.Empty(t, publicModelCapabilitiesFromMatrix(matrix, nil))
}

func TestNormalizePublicModelProtocolEndpoints_AddsCapabilityMatrixCandidatesAsUnverified(t *testing.T) {
	endpoints := normalizePublicModelProtocolEndpoints(nil, []string{PlatformGemini}, publicModelCatalogMetadataSource{
		CapabilitySource: PublicModelCapabilitySourceManualConfig,
		Verified:         true,
	})

	var countTokens *PublicModelProtocolEndpoint
	for index := range endpoints {
		if endpoints[index].Key == "gemini.countTokens" {
			countTokens = &endpoints[index]
			break
		}
	}

	require.NotNil(t, countTokens)
	require.Equal(t, PublicModelCapabilitySourceManualConfig, countTokens.Source)
	require.False(t, countTokens.Verified)
	require.NotEqual(t, PublicModelSupportUnsupported, countTokens.Support)
}

func TestPublicModelLifecycleResolutionMarksNameInference(t *testing.T) {
	explicit := publicModelLifecycleFromResolution(
		resolvePublicModelLifecycleStatus(PublicModelLifecycleDeprecated, "gpt-next"),
		PublicModelLifecycleSourceOfficialRegistry,
	)
	require.Equal(t, PublicModelLifecycleDeprecated, explicit.Status)
	require.Equal(t, PublicModelLifecycleSourceOfficialRegistry, explicit.Source)
	require.NotEqual(t, PublicModelLifecycleConfidenceInferred, explicit.Confidence)

	inferred := publicModelLifecycleFromResolution(
		resolvePublicModelLifecycleStatus("", "gpt-next-preview"),
		PublicModelLifecycleSourceManualConfig,
	)
	require.Equal(t, PublicModelLifecycleBeta, inferred.Status)
	require.Equal(t, PublicModelLifecycleSourceInferred, inferred.Source)
	require.Equal(t, PublicModelLifecycleConfidenceInferred, inferred.Confidence)
}

func TestPublicModelCatalogItemLogFieldsIncludeCapabilityAuditKeys(t *testing.T) {
	fields := publicModelCatalogItemLogFields(PublicModelCatalogItem{
		EntryID:         "entry-1",
		PublicModelID:   "public-model",
		SourceAccountID: 42,
		SourceProtocol:  PlatformOpenAI,
		SourceModelID:   "source-model",
		ProtocolEndpoints: []PublicModelProtocolEndpoint{{
			Key:      "openai.responses",
			Protocol: PlatformOpenAI,
		}},
		CapabilityMatrix: []PublicModelCapabilityMatrixEntry{{
			Capability: "text",
			Protocol:   PlatformOpenAI,
			Endpoint:   "openai.responses",
		}},
		AvailabilityState: AccountModelAvailabilityVerified,
	})

	encoded := map[string]struct{}{}
	for _, field := range fields {
		encoded[field.Key] = struct{}{}
	}
	for _, key := range []string{
		"account_id",
		"public_model_id",
		"source_model_id",
		"protocol",
		"endpoint",
		"capability",
		"result",
	} {
		require.Contains(t, encoded, key)
	}
	require.NotContains(t, encoded, "token")
	require.NotContains(t, encoded, "api_key")
	require.IsType(t, zap.Field{}, fields[0])
}
