package admin

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPrepareAccountModelScopePrefersManualMappingRows(t *testing.T) {
	handler := &AccountHandler{
		modelRegistryService: service.NewModelRegistryService(newTestSettingRepo()),
	}

	credentials := map[string]any{
		"model_mapping": map[string]any{
			"legacy": "legacy-target",
		},
	}
	extra := map[string]any{
		"model_scope_v2": map[string]any{
			"manual_mapping_rows": []map[string]any{
				{
					"from": "中文模型",
					"to":   "gpt-4.1",
				},
				{
					"from": "另一个模型",
					"to":   "gpt-4.1-mini",
				},
			},
			"manual_mappings": map[string]any{
				"gpt-4.1": "should-not-win",
			},
		},
	}

	nextCredentials, nextExtra, err := handler.prepareAccountModelScope(
		context.Background(),
		service.PlatformOpenAI,
		service.AccountTypeAPIKey,
		credentials,
		extra,
	)
	require.NoError(t, err)

	modelMapping, ok := nextCredentials["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-4.1", modelMapping["中文模型"])
	require.Equal(t, "gpt-4.1-mini", modelMapping["另一个模型"])
	_, exists := modelMapping["gpt-4.1"]
	require.False(t, exists)

	require.NotNil(t, nextExtra)
	require.Contains(t, nextExtra, "model_scope_v2")
}

func TestPrepareAccountModelScopeClearsLegacyModelMappingWhenScopeRemoved(t *testing.T) {
	handler := &AccountHandler{
		modelRegistryService: service.NewModelRegistryService(newTestSettingRepo()),
	}

	credentials := map[string]any{
		"api_key": "sk-test",
		"model_mapping": map[string]any{
			"friendly-gpt": "gpt-5.4",
		},
	}
	extra := map[string]any{
		"gateway_protocol": "openai",
	}

	nextCredentials, nextExtra, err := handler.prepareAccountModelScope(
		context.Background(),
		service.PlatformProtocolGateway,
		service.AccountTypeAPIKey,
		credentials,
		extra,
	)
	require.NoError(t, err)
	require.NotNil(t, nextCredentials)
	_, exists := nextCredentials["model_mapping"]
	require.False(t, exists)
	require.Equal(t, "sk-test", nextCredentials["api_key"])
	require.Equal(t, extra, nextExtra)
}

func TestPrepareAccountModelScopeTreatsEmptyStructuredScopeAsUnrestricted(t *testing.T) {
	handler := &AccountHandler{
		modelRegistryService: service.NewModelRegistryService(newTestSettingRepo()),
	}

	credentials := map[string]any{
		"api_key": "sk-test",
		"model_mapping": map[string]any{
			"pp-ocrv5-server": "pp-ocrv5-server",
		},
	}
	extra := map[string]any{
		"model_scope_v2": map[string]any{
			"policy_mode": "whitelist",
			"entries":     []any{},
		},
		"document_ai_mode": "direct",
	}

	nextCredentials, nextExtra, err := handler.prepareAccountModelScope(
		context.Background(),
		service.PlatformBaiduDocumentAI,
		service.AccountTypeAPIKey,
		credentials,
		extra,
	)
	require.NoError(t, err)
	require.NotNil(t, nextCredentials)
	require.Equal(t, "sk-test", nextCredentials["api_key"])
	_, exists := nextCredentials["model_mapping"]
	require.False(t, exists)
	require.Equal(t, map[string]any{
		"document_ai_mode": "direct",
	}, nextExtra)
}
