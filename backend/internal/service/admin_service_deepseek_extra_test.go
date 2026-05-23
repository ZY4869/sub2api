package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAccountExtraForStorageDeepSeekCanonicalizesConcurrencyLimits(t *testing.T) {
	extra := normalizeAccountExtraForStorage(PlatformDeepSeek, AccountTypeAPIKey, nil, map[string]any{
		DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{
			"Deepseek/deepseek V4 Flash:free": 2500,
			"DEEPSEEK V4 PRO":                 "500",
			"deepseek-v4-lite":                10,
			"deepseek-v4-flash-zero":          0,
		},
	})

	require.Equal(t, map[string]any{
		DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{
			"deepseek-v4-flash": 2500,
			"deepseek-v4-pro":   500,
		},
	}, extra)
}

func TestNormalizeAccountExtraForStorageNonDeepSeekDropsConcurrencyLimits(t *testing.T) {
	extra := normalizeAccountExtraForStorage(PlatformOpenAI, AccountTypeAPIKey, nil, map[string]any{
		DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{"deepseek-v4-pro": 500},
		"gateway_protocol":                     "openai",
	})

	require.Equal(t, map[string]any{
		"gateway_protocol":    "openai",
		"image_protocol_mode": "native",
	}, extra)
}
