package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestResponsesImageToolEndpointPlatformSupport(t *testing.T) {
	testCases := []struct {
		name          string
		platform      string
		wantSupported bool
		wantProvider  string
	}{
		{name: "openai", platform: service.PlatformOpenAI, wantSupported: true, wantProvider: service.PlatformOpenAI},
		{name: "copilot", platform: service.PlatformCopilot, wantSupported: true, wantProvider: service.PlatformOpenAI},
		{name: "grok", platform: service.PlatformGrok, wantSupported: false, wantProvider: service.PlatformGrok},
		{name: "gemini", platform: service.PlatformGemini, wantSupported: false, wantProvider: service.PlatformGemini},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantSupported, supportsResponsesImageToolPlatform(tc.platform))
			require.Equal(t, tc.wantProvider, responsesImageToolResolvedProvider(tc.platform))
		})
	}
}

func TestResponsesImageToolEndpointUnsupportedMessage(t *testing.T) {
	message := responsesImageToolUnsupportedPlatformMessage()

	require.Contains(t, message, "/grok/v1/images/*")
	require.Contains(t, message, "/v1beta/openai/images/generations")
	require.Contains(t, message, "/v1/responses")
}

func TestResolveResponsesImageToolOpenAITargetModel(t *testing.T) {
	account := &service.Account{
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-image": "gpt-image-2",
			},
		},
	}

	target, ok := resolveResponsesImageToolOpenAITargetModel(account, "friendly-image")
	require.True(t, ok)
	require.Equal(t, "gpt-image-2", target)

	target, ok = resolveResponsesImageToolOpenAITargetModel(account, "gemini-2.5-flash-image")
	require.False(t, ok)
	require.Equal(t, "gemini-2.5-flash-image", target)

	target, ok = resolveResponsesImageToolOpenAITargetModel(account, "")
	require.True(t, ok)
	require.Empty(t, target)
}

func TestResponsesImageToolUnsupportedModelMessage(t *testing.T) {
	message := responsesImageToolUnsupportedModelMessage("gemini-2.5-flash-image")

	require.Contains(t, message, "gemini-2.5-flash-image")
	require.Contains(t, message, "provider-specific image endpoint")
}
