package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountManualModelsToExtraValue_PreservesProviderMetadata(t *testing.T) {
	values := AccountManualModelsToExtraValue([]AccountManualModel{
		{
			ModelID:      "custom-model",
			RequestAlias: "alias-model",
			Provider:     "Grok",
		},
	}, false)

	require.Equal(t, []map[string]any{
		{
			"model_id":      "custom-model",
			"request_alias": "alias-model",
			"provider":      "grok",
		},
	}, values)
}

func TestAccountManualModelsFromExtra_CompatibilityWithoutProvider(t *testing.T) {
	models := AccountManualModelsFromExtra(map[string]any{
		"manual_models": []any{
			map[string]any{
				"model_id":        "custom-model",
				"request_alias":   "alias-model",
				"source_protocol": "openai",
			},
		},
	}, true)

	require.Equal(t, []AccountManualModel{
		{
			ModelID:        "custom-model",
			RequestAlias:   "alias-model",
			SourceProtocol: "openai",
		},
	}, models)
}
