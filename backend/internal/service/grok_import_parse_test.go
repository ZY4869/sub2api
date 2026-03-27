package service

import "testing"

import "github.com/stretchr/testify/require"

func TestParseGrokImportPayloadLegacyPools(t *testing.T) {
	result, err := ParseGrokImportPayload(`{
		"ssoBasic": [{"token": "Bearer basic-token-12345678"}],
		"ssoHeavy": [{"token": "heavy-token-12345678"}]
	}`)
	require.NoError(t, err)
	require.Equal(t, GrokDetectedKindLegacyPool, result.DetectedKind)
	require.Len(t, result.Candidates, 2)

	candidatesByPool := make(map[string]GrokImportCandidate, len(result.Candidates))
	for _, candidate := range result.Candidates {
		candidatesByPool[candidate.SourcePool] = candidate
	}

	basic := candidatesByPool["ssoBasic"]
	require.Equal(t, GrokDetectedKindSSO, basic.Type)
	require.Equal(t, GrokTierBasic, basic.Tier)
	require.Equal(t, 50, basic.Priority)
	require.Equal(t, 1, basic.Concurrency)
	require.Equal(t, "basic-token-12345678", basic.Credentials["sso_token"])

	heavy := candidatesByPool["ssoHeavy"]
	require.Equal(t, GrokDetectedKindSSO, heavy.Type)
	require.Equal(t, GrokTierHeavy, heavy.Tier)
	require.Equal(t, 30, heavy.Priority)
	require.Equal(t, 1, heavy.Concurrency)
	require.Equal(t, "heavy-token-12345678", heavy.Credentials["sso_token"])

	heavyCapabilities, ok := heavy.Extra["grok_capabilities"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, heavyCapabilities["allow_heavy_model"])
	require.Equal(t, "ssoHeavy", heavy.Extra["grok_import_source_pool"])
}

func TestParseGrokImportPayloadPlainTextMixedKinds(t *testing.T) {
	result, err := ParseGrokImportPayload("xai-demo-key-12345678\nBearer sso-demo-token-12345678\n")
	require.NoError(t, err)
	require.Equal(t, GrokDetectedKindSSO, result.DetectedKind)
	require.Len(t, result.Candidates, 2)
	require.Empty(t, result.Errors)

	apiKeyCandidate := result.Candidates[0]
	require.Equal(t, GrokDetectedKindAPIKey, apiKeyCandidate.Type)
	require.Equal(t, "xai-demo-key-12345678", apiKeyCandidate.Credentials["api_key"])
	require.Equal(t, "https://api.x.ai", apiKeyCandidate.Credentials["base_url"])

	ssoCandidate := result.Candidates[1]
	require.Equal(t, GrokDetectedKindSSO, ssoCandidate.Type)
	require.Equal(t, "sso-demo-token-12345678", ssoCandidate.Credentials["sso_token"])
	require.NotEmpty(t, ssoCandidate.Credentials["model_mapping"])
}
