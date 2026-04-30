package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewayService_MaybeInjectAnthropicCacheTTL1h_EnabledInjectsTTL(t *testing.T) {
	repo := &staticSettingRepoStub{
		values: map[string]string{
			SettingKeyEnableAnthropicCacheTTL1hInjection: "true",
		},
	}
	settingSvc := NewSettingService(repo, nil)
	svc := &GatewayService{settingService: settingSvc}

	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeOAuth}
	body := []byte(`{"alpha":1,"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral"}},{"type":"text","text":"s2"}],"messages":[{"role":"user","content":[{"type":"text","text":"m","cache_control":{"type":"ephemeral","ttl":"5m"}}]}],"omega":2}`)

	out := svc.maybeInjectAnthropicCacheTTL1h(context.Background(), account, body)
	require.Equal(t, "1h", gjson.GetBytes(out, "system.0.cache_control.ttl").String())
	require.Equal(t, "1h", gjson.GetBytes(out, "messages.0.content.0.cache_control.ttl").String())
}

func TestGatewayService_MaybeInjectAnthropicCacheTTL1h_DisabledNoChange(t *testing.T) {
	repo := &staticSettingRepoStub{
		values: map[string]string{
			SettingKeyEnableAnthropicCacheTTL1hInjection: "false",
		},
	}
	settingSvc := NewSettingService(repo, nil)
	svc := &GatewayService{settingService: settingSvc}

	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeOAuth}
	body := []byte(`{"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":[{"type":"text","text":"m","cache_control":{"type":"ephemeral"}}]}]}`)

	out := svc.maybeInjectAnthropicCacheTTL1h(context.Background(), account, body)
	require.False(t, gjson.GetBytes(out, "system.0.cache_control.ttl").Exists())
	require.False(t, gjson.GetBytes(out, "messages.0.content.0.cache_control.ttl").Exists())
}

func TestGatewayService_MaybeInjectAnthropicCacheTTL1h_IgnoresNonOAuthAccounts(t *testing.T) {
	repo := &staticSettingRepoStub{
		values: map[string]string{
			SettingKeyEnableAnthropicCacheTTL1hInjection: "true",
		},
	}
	settingSvc := NewSettingService(repo, nil)
	svc := &GatewayService{settingService: settingSvc}

	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	body := []byte(`{"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral"}}]}`)

	out := svc.maybeInjectAnthropicCacheTTL1h(context.Background(), account, body)
	require.Equal(t, string(body), string(out))
}
