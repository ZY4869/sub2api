//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateGrokAccountInput_APIKeyIgnoresLegacyTierFields(t *testing.T) {
	err := validateGrokAccountInput(
		PlatformGrok,
		AccountTypeAPIKey,
		map[string]any{"api_key": "xai-key"},
		map[string]any{
			"grok_tier":         "not-a-real-tier",
			"grok_capabilities": "legacy-string",
		},
	)

	require.NoError(t, err)
}

func TestNormalizeGrokExtraForStorageByType_APIKeyDropsTierFields(t *testing.T) {
	normalized := normalizeGrokExtraForStorageByType(AccountTypeAPIKey, map[string]any{
		"grok_tier":         GrokTierHeavy,
		"grok_capabilities": map[string]any{"vision": true},
		"manual_models":     []any{"grok-4"},
	})

	require.NotNil(t, normalized)
	require.NotContains(t, normalized, "grok_tier")
	require.NotContains(t, normalized, "grok_capabilities")
	require.Equal(t, []any{"grok-4"}, normalized["manual_models"])
}

func TestNormalizeGrokExtraForStorageByType_SSOKeepsTierAndCapabilities(t *testing.T) {
	normalized := normalizeGrokExtraForStorageByType(AccountTypeSSO, map[string]any{
		"grok_tier": GrokTierSuper,
	})

	require.Equal(t, GrokTierSuper, normalized["grok_tier"])
	capabilities, ok := normalized["grok_capabilities"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, capabilities)
}
