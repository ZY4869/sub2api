package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicModelCatalogRuntimeErrorIsCapabilityFailure(t *testing.T) {
	require.True(t, publicModelCatalogRuntimeErrorIsCapabilityFailure(http.StatusBadRequest, "model not supported on this endpoint"))
	require.True(t, publicModelCatalogRuntimeErrorIsCapabilityFailure(http.StatusUnprocessableEntity, "context limit exceeded"))

	require.False(t, publicModelCatalogRuntimeErrorIsCapabilityFailure(http.StatusTooManyRequests, "model not supported after quota"))
	require.False(t, publicModelCatalogRuntimeErrorIsCapabilityFailure(http.StatusBadRequest, "rate limit exceeded"))
	require.False(t, publicModelCatalogRuntimeErrorIsCapabilityFailure(http.StatusForbidden, "does not support this action"))
}

func TestApplyPublicModelCatalogRuntimeFailure_AddsScopedUnsupportedEntries(t *testing.T) {
	item := PublicModelCatalogItem{
		Model:            "gpt-5.4",
		PublicModelID:    "gpt-5.4",
		RequestProtocols: []string{PlatformOpenAI},
		Capabilities:     []string{"text"},
	}

	item = applyPublicModelCatalogRuntimeFailure(item, PlatformOpenAI, "openai.responses", "text", PublicModelSupportUnsupported, "2026-05-30T00:00:00Z")

	require.Len(t, item.ProtocolEndpoints, 1)
	require.Equal(t, "openai.responses", item.ProtocolEndpoints[0].Key)
	require.Equal(t, PublicModelSupportUnsupported, item.ProtocolEndpoints[0].Support)
	require.Equal(t, PublicModelCapabilitySourceRuntimeObserved, item.ProtocolEndpoints[0].Source)
	require.Empty(t, item.RequestProtocols)

	require.Len(t, item.CapabilityMatrix, 1)
	require.Equal(t, "text", item.CapabilityMatrix[0].Capability)
	require.Equal(t, PublicModelSupportUnsupported, item.CapabilityMatrix[0].Support)
	require.Empty(t, item.Capabilities)
}

func TestApplyPublicModelCatalogRuntimeFailure_PreservesVerifiedSupportedMetadata(t *testing.T) {
	item := PublicModelCatalogItem{
		Model:            "gpt-5.4",
		PublicModelID:    "gpt-5.4",
		RequestProtocols: []string{PlatformOpenAI},
		Capabilities:     []string{"text"},
		ProtocolEndpoints: []PublicModelProtocolEndpoint{{
			Key:      "openai.responses",
			Protocol: PlatformOpenAI,
			Support:  PublicModelSupportSupported,
			Source:   PublicModelCapabilitySourceVerifiedProbe,
			Verified: true,
		}},
		CapabilityMatrix: []PublicModelCapabilityMatrixEntry{{
			Capability: "text",
			Protocol:   PlatformOpenAI,
			Endpoint:   "openai.responses",
			Support:    PublicModelSupportPartial,
			Source:     PublicModelCapabilitySourceAccountProbe,
			Verified:   true,
		}},
	}

	item = applyPublicModelCatalogRuntimeFailure(item, PlatformOpenAI, "openai.responses", "text", PublicModelSupportUnsupported, "2026-05-30T00:00:00Z")

	require.Len(t, item.ProtocolEndpoints, 1)
	require.Equal(t, PublicModelSupportSupported, item.ProtocolEndpoints[0].Support)
	require.Equal(t, PublicModelCapabilitySourceVerifiedProbe, item.ProtocolEndpoints[0].Source)
	require.Equal(t, "2026-05-30T00:00:00Z", item.ProtocolEndpoints[0].LastCheckedAt)
	require.Contains(t, item.ProtocolEndpoints[0].Limitations, "runtime_failure_observed:unsupported")

	require.Len(t, item.CapabilityMatrix, 1)
	require.Equal(t, PublicModelSupportPartial, item.CapabilityMatrix[0].Support)
	require.Equal(t, PublicModelCapabilitySourceAccountProbe, item.CapabilityMatrix[0].Source)
	require.Equal(t, "2026-05-30T00:00:00Z", item.CapabilityMatrix[0].LastCheckedAt)
	require.Contains(t, item.CapabilityMatrix[0].Limitations, "runtime_failure_observed:unsupported")
	require.Contains(t, item.RequestProtocols, PlatformOpenAI)
	require.Contains(t, item.Capabilities, "text")
}
