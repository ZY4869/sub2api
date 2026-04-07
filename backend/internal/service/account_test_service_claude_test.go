//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountTestService_TestClaudeAccountConnection_UsesClaudeTokenProviderForKiroOAuth(t *testing.T) {
	ctx, recorder := newGatewayTestContext()
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newKiroEventStreamHTTPResponse(http.StatusOK, kiroTestStreamEvent{
				EventType: "assistantResponseEvent",
				Payload:   `{"assistantResponseEvent":{"content":"hello from kiro"}}`,
			}),
		},
	}
	tokenCache := newClaudeTokenCacheStub()
	account := &Account{
		ID:          321,
		Platform:    PlatformKiro,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "stale-token",
			"profile_arn":  "arn:aws:codewhisperer:us-east-1:123456789012:profile/test",
		},
	}
	tokenCache.tokens[KiroTokenCacheKey(account)] = "refreshed-token"
	provider := NewClaudeTokenProvider(nil, tokenCache, nil)

	svc := &AccountTestService{
		httpUpstream:        upstream,
		claudeTokenProvider: provider,
	}

	err := svc.testClaudeAccountConnection(ctx, account, "claude-sonnet-4.5", "", "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://q.us-east-1.amazonaws.com/generateAssistantResponse", upstream.requests[0].URL.String())
	require.Equal(t, "Bearer refreshed-token", upstream.requests[0].Header.Get("Authorization"))
	require.Equal(t, kiroAgentMode, upstream.requests[0].Header.Get("x-amzn-kiro-agent-mode"))
	require.Empty(t, upstream.requests[0].Header.Get("anthropic-beta"))
	require.Contains(t, recorder.Body.String(), `"type":"content"`)
	require.Contains(t, recorder.Body.String(), "Kiro runtime region")
	require.Contains(t, recorder.Body.String(), "Kiro runtime endpoint")
	require.Contains(t, recorder.Body.String(), "hello from kiro")
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
}

func TestAccountTestService_TestClaudeAccountConnection_KiroUnauthorizedDoesNotLeakAnthropicMessage(t *testing.T) {
	ctx, recorder := newGatewayTestContext()
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusUnauthorized, `{"message":"token expired"}`),
		},
	}
	account := &Account{
		ID:          322,
		Platform:    PlatformKiro,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "live-token",
			"profile_arn":  "arn:aws:codewhisperer:us-west-2:123456789012:profile/test",
		},
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
	}

	err := svc.testClaudeAccountConnection(ctx, account, "claude-sonnet-4.5", "", "")
	require.Error(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://q.us-west-2.amazonaws.com/generateAssistantResponse", upstream.requests[0].URL.String())
	require.Contains(t, recorder.Body.String(), "Kiro runtime region")
	require.Contains(t, recorder.Body.String(), "Kiro runtime endpoint")
	require.Contains(t, recorder.Body.String(), "API returned 401")
	require.Contains(t, recorder.Body.String(), "token expired")
	require.NotContains(t, recorder.Body.String(), "Invalid bearer token")
}
