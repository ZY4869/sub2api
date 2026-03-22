//go:build unit

package service

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/stretchr/testify/require"
)

func TestAccountTestService_TestClaudeAccountConnection_UsesClaudeTokenProviderForKiroOAuth(t *testing.T) {
	ctx, _ := newSoraTestContext()
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`data: {"type":"message_stop"}

`)),
			},
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
		},
	}
	tokenCache.tokens[KiroTokenCacheKey(account)] = "refreshed-token"
	provider := NewClaudeTokenProvider(nil, tokenCache, nil)

	svc := &AccountTestService{
		httpUpstream:        upstream,
		claudeTokenProvider: provider,
	}

	err := svc.testClaudeAccountConnection(ctx, account, "claude-sonnet-4.5")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "Bearer refreshed-token", upstream.requests[0].Header.Get("Authorization"))
	require.Equal(t, claude.DefaultBetaHeader, upstream.requests[0].Header.Get("anthropic-beta"))
}
