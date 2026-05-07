package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeBaiduDocumentAICredentialsForStorage_MirrorsSingleTokenAcrossAsyncAndDirect(t *testing.T) {
	normalized := normalizeBaiduDocumentAICredentialsForStorage(map[string]any{
		"async_bearer_token": " shared-token ",
		"async_base_url":     "https://aistudio.baidu.com/async/",
	})

	require.Equal(t, "shared-token", normalized["async_bearer_token"])
	require.Equal(t, "shared-token", normalized["direct_token"])
	require.Equal(t, "https://aistudio.baidu.com/async", normalized["async_base_url"])
}

func TestNormalizeBaiduDocumentAICredentialsForStorage_BackfillsAsyncFromDirectOnlyToken(t *testing.T) {
	normalized := normalizeBaiduDocumentAICredentialsForStorage(map[string]any{
		"direct_token": "direct-only-token",
	})

	require.Equal(t, "direct-only-token", normalized["async_bearer_token"])
	require.Equal(t, "direct-only-token", normalized["direct_token"])
	require.Equal(t, DefaultBaiduDocumentAIAsyncBaseURL(), normalized["async_base_url"])
}
