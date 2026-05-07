package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountGetBaiduDocumentAITokens_FallbackBetweenAsyncAndDirect(t *testing.T) {
	asyncOnly := &Account{
		Credentials: map[string]any{
			"async_bearer_token": "async-token",
		},
	}
	require.Equal(t, "async-token", asyncOnly.GetBaiduDocumentAIAsyncBearerToken())
	require.Equal(t, "async-token", asyncOnly.GetBaiduDocumentAIDirectToken())
	require.True(t, asyncOnly.IsBaiduDocumentAIAsyncMode())

	directOnly := &Account{
		Credentials: map[string]any{
			"direct_token": "direct-token",
		},
	}
	require.Equal(t, "direct-token", directOnly.GetBaiduDocumentAIAsyncBearerToken())
	require.Equal(t, "direct-token", directOnly.GetBaiduDocumentAIDirectToken())
	require.False(t, directOnly.IsBaiduDocumentAIAsyncMode())
}

func TestAccountGetBaiduDocumentAIModeHonorsExplicitHint(t *testing.T) {
	direct := &Account{
		Extra: map[string]any{
			"document_ai_mode": "direct",
		},
		Credentials: map[string]any{
			"async_bearer_token": "async-token",
			"direct_token":       "direct-token",
		},
	}
	require.Equal(t, "direct", direct.GetBaiduDocumentAIMode())
	require.False(t, direct.IsBaiduDocumentAIAsyncMode())

	async := &Account{
		Extra: map[string]any{
			"document_ai_mode": "async",
		},
		Credentials: map[string]any{
			"direct_token": "direct-token",
		},
	}
	require.Equal(t, "async", async.GetBaiduDocumentAIMode())
	require.True(t, async.IsBaiduDocumentAIAsyncMode())
}
