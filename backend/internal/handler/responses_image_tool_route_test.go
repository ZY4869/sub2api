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
