package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOpenAITargetURLForDeepSeek(t *testing.T) {
	require.Equal(t, "https://api.deepseek.com/chat/completions", buildOpenAIChatCompletionsURLForPlatform("", PlatformDeepSeek))
	require.Equal(t, "https://api.deepseek.com/models", buildOpenAIModelsURLForPlatform("", PlatformDeepSeek))
	require.Equal(t, "https://relay.example.com/chat/completions", buildOpenAIChatCompletionsURLForPlatform("https://relay.example.com", PlatformDeepSeek))
	require.Equal(t, "https://relay.example.com/models", buildOpenAIModelsURLForPlatform("https://relay.example.com", PlatformDeepSeek))
	require.Equal(t, "https://relay.example.com/v1/chat/completions", buildOpenAIChatCompletionsURLForPlatform("https://relay.example.com/v1", PlatformDeepSeek))
	require.Equal(t, "https://relay.example.com/v1/models", buildOpenAIModelsURLForPlatform("https://relay.example.com/v1", PlatformDeepSeek))
}

func TestResolveOpenAICompatibleBaseURLForDeepSeekStripsAnthropicSuffix(t *testing.T) {
	account := &Account{
		Platform: PlatformDeepSeek,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://relay.example.com/anthropic/",
		},
	}

	chatURL, err := resolveOpenAIChatCompletionsTargetURL(account, nil)
	require.NoError(t, err)
	require.Equal(t, "https://relay.example.com/chat/completions", chatURL)

	modelsURL := buildOpenAIModelsURLForPlatform(resolveOpenAICompatibleBaseURL(account), account.Platform)
	require.Equal(t, "https://relay.example.com/models", modelsURL)
}
