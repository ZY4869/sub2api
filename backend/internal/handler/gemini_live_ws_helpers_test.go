package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGeminiLiveAuthTokenProxyRequested(t *testing.T) {
	require.True(t, geminiLiveAuthTokenProxyRequested("/v1alpha/authTokens"))
	require.True(t, geminiLiveAuthTokenProxyRequested("/v1beta/live/auth-token"))
	require.True(t, geminiLiveAuthTokenProxyRequested("/v1beta/live/auth-tokens"))
	require.True(t, geminiLiveAuthTokenProxyRequested("/v1beta/live/authtokens"))
	require.False(t, geminiLiveAuthTokenProxyRequested("/v1beta/live"))
}

func TestDetectGeminiLiveSetupMetadata(t *testing.T) {
	payload := []byte(`{
		"setup": {
			"model": "models/gemini-live-2.5-flash",
			"sessionResumption": {
				"handle": "resume-handle-123"
			}
		}
	}`)

	require.Equal(t, "gemini-live-2.5-flash", detectGeminiLiveRequestedModel(payload))
	require.Equal(t, service.DeriveSessionHashFromSeed("gemini-live:resume-handle-123"), detectGeminiLiveSessionHash(payload))
}

func TestGeminiLiveUsageStateObservesServerFrame(t *testing.T) {
	state := &geminiLiveUsageState{}

	handle := state.observeServerFrame([]byte(`{
		"usageMetadata": {
			"promptTokenCount": 128,
			"responseTokenCount": 64,
			"cachedContentTokenCount": 32,
			"responseTokensDetails": [{"modality":"audio"}]
		},
		"responseId": "resp_live_123",
		"setupComplete": {
			"model": "models/gemini-live-2.5-flash"
		},
		"sessionResumptionUpdate": {
			"newHandle": "resume-next"
		}
	}`))

	usage, mediaType, requestID, upstreamModel := state.snapshot()
	require.Equal(t, "resume-next", handle)
	require.Equal(t, 128, usage.InputTokens)
	require.Equal(t, 64, usage.OutputTokens)
	require.Equal(t, 32, usage.CacheReadInputTokens)
	require.Equal(t, "audio", mediaType)
	require.Equal(t, "resp_live_123", requestID)
	require.Equal(t, "gemini-live-2.5-flash", upstreamModel)
}
