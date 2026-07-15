package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountRequestHeaderOverrides_FiltersBlockedHeaders(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			AccountRequestHeadersCredentialKey: map[string]any{
				"X-Trace-Id":     "trace-1",
				"authorization":  "Bearer attacker",
				"Cookie":         "secret=1",
				"Host":           "evil.example",
				"x-grok-conv-id": "conv-from-account",
				"Bad\nName":      "ignored",
				"X-Multi":        []any{"a", "b"},
			},
		},
	}

	headers := AccountRequestHeaderOverrides(account)

	require.Equal(t, "trace-1", headers.Get("X-Trace-Id"))
	require.ElementsMatch(t, []string{"a", "b"}, headers.Values("X-Multi"))
	require.Empty(t, headers.Get("Authorization"))
	require.Empty(t, headers.Get("Cookie"))
	require.Empty(t, headers.Get("Host"))
	require.Empty(t, headers.Get("x-grok-conv-id"))
}

func TestApplyAccountRequestHeaderOverrides_PreservesAuthHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.test/v1/responses", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer real-token")
	req.Header.Set("X-Trace-Id", "old")

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			AccountRequestHeadersCredentialKey: map[string]any{
				"Authorization": "Bearer attacker",
				"X-Trace-Id":    "new",
			},
		},
	}

	ApplyAccountRequestHeaderOverrides(req, account)

	require.Equal(t, "Bearer real-token", req.Header.Get("Authorization"))
	require.Equal(t, "new", req.Header.Get("X-Trace-Id"))
}

func TestAccountRequestHeaderOverrides_IgnoresNonAPIKeyAccounts(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			AccountRequestHeadersCredentialKey: map[string]any{"X-Trace-Id": "trace-1"},
		},
	}

	require.Empty(t, AccountRequestHeaderOverrides(account))
}
