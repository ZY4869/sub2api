//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeepSeekEffectiveAccountConcurrencyUsesCanonicalModelLimit(t *testing.T) {
	account := &Account{
		ID:          1,
		Platform:    PlatformDeepSeek,
		Concurrency: 1000,
		Extra: map[string]any{
			DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{
				"Deepseek/deepseek V4 Pro:free": 500,
			},
		},
	}

	require.Equal(t, 500, DeepSeekEffectiveAccountConcurrency(account, "deepseek-v4-pro"))
	require.Equal(t, 1000, DeepSeekEffectiveAccountConcurrency(account, "deepseek-v4-flash"))
}

func TestNormalizeDeepSeekModelIDVariants(t *testing.T) {
	require.Equal(t, "deepseek-v4-flash", normalizeDeepSeekModelID("Deepseek/deepseek V4 Flash:free"))
	require.Equal(t, "deepseek-v4-flash", normalizeDeepSeekModelID("deepseek_v4_flash_free"))
	require.Equal(t, "deepseek-v4-pro", normalizeDeepSeekModelID("DEEPSEEK V4 PRO"))
	require.Equal(t, "deepseek-v4-lite", normalizeDeepSeekModelID("deepseek-v4-lite"))
}

func TestDeepSeekEffectiveAccountConcurrencyFallsBackAndIgnoresInvalidExtra(t *testing.T) {
	account := &Account{
		ID:          2,
		Platform:    PlatformDeepSeek,
		Concurrency: 42,
		Extra: map[string]any{
			DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{
				"deepseek-v4-pro":  0,
				"deepseek-v4-lite": 10,
			},
		},
	}

	require.Equal(t, 42, DeepSeekEffectiveAccountConcurrency(account, "deepseek-v4-pro"))
	require.Nil(t, account.DeepSeekModelConcurrencyLimits())
}

func TestDeepSeekEffectiveAccountConcurrencyNonDeepSeekUnaffected(t *testing.T) {
	account := &Account{
		ID:          3,
		Platform:    PlatformOpenAI,
		Concurrency: 7,
		Extra: map[string]any{
			DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{
				"deepseek-v4-pro": 1,
			},
		},
	}

	require.Equal(t, 7, DeepSeekEffectiveAccountConcurrency(account, "deepseek-v4-pro"))
}
