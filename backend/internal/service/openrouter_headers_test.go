package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyOpenRouterAttributionHeaders(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenRouter,
		Credentials: map[string]any{
			"http_referer":     " https://example.com ",
			"openrouter_title": " Sub2API ",
		},
	}

	headers := map[string]string{}
	applyOpenRouterAttributionHeaders(account, headers)

	require.Equal(t, "https://example.com", headers["HTTP-Referer"])
	require.Equal(t, "Sub2API", headers["X-OpenRouter-Title"])
}

func TestApplyOpenRouterAttributionRequestHeadersIgnoresOtherPlatforms(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Credentials: map[string]any{
			"http_referer":     "https://example.com",
			"openrouter_title": "Sub2API",
		},
	}

	headers := http.Header{}
	applyOpenRouterAttributionRequestHeaders(account, headers)

	require.Empty(t, headers.Get("HTTP-Referer"))
	require.Empty(t, headers.Get("X-OpenRouter-Title"))
}
