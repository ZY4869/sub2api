package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccount_IsAnthropicAPIKeyPassthroughEnabled(t *testing.T) {
	t.Run("Anthropic API Key 开启", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"anthropic_passthrough": true,
			},
		}
		require.True(t, account.IsAnthropicAPIKeyPassthroughEnabled())
	})

	t.Run("Anthropic API Key 关闭", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"anthropic_passthrough": false,
			},
		}
		require.False(t, account.IsAnthropicAPIKeyPassthroughEnabled())
	})

	t.Run("字段类型非法默认关闭", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"anthropic_passthrough": "true",
			},
		}
		require.False(t, account.IsAnthropicAPIKeyPassthroughEnabled())
	})

	t.Run("非 Anthropic API Key 账号始终关闭", func(t *testing.T) {
		oauth := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeOAuth,
			Extra: map[string]any{
				"anthropic_passthrough": true,
			},
		}
		require.False(t, oauth.IsAnthropicAPIKeyPassthroughEnabled())

		openai := &Account{
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"anthropic_passthrough": true,
			},
		}
		require.False(t, openai.IsAnthropicAPIKeyPassthroughEnabled())
	})
}

func TestAccount_AnthropicAPIKeyAuthScheme(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		want    string
	}{
		{
			name: "默认 x-api-key",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
			},
			want: AnthropicAPIKeyAuthSchemeXAPIKey,
		},
		{
			name: "显式 Bearer",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Extra: map[string]any{
					anthropicAPIKeyAuthSchemeExtraKey: AnthropicAPIKeyAuthSchemeAuthorizationBearer,
				},
			},
			want: AnthropicAPIKeyAuthSchemeAuthorizationBearer,
		},
		{
			name: "非法值回退",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Extra: map[string]any{
					anthropicAPIKeyAuthSchemeExtraKey: "authorization",
				},
			},
			want: AnthropicAPIKeyAuthSchemeXAPIKey,
		},
		{
			name: "非 Anthropic API Key 忽略",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Extra: map[string]any{
					anthropicAPIKeyAuthSchemeExtraKey: AnthropicAPIKeyAuthSchemeAuthorizationBearer,
				},
			},
			want: AnthropicAPIKeyAuthSchemeXAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.AnthropicAPIKeyAuthScheme())
		})
	}
}

func TestApplyAnthropicAPIKeyAuthHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", nil)
	require.NoError(t, err)
	req.Header.Set("authorization", "Bearer inbound")
	req.Header.Set("x-api-key", "inbound")

	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			anthropicAPIKeyAuthSchemeExtraKey: AnthropicAPIKeyAuthSchemeAuthorizationBearer,
		},
	}
	ApplyAnthropicAPIKeyAuthHeader(req, account, "upstream-key")

	require.Equal(t, "Bearer upstream-key", req.Header.Get("authorization"))
	require.Empty(t, req.Header.Get("x-api-key"))
}
