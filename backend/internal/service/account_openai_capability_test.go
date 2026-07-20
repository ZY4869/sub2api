package service

import "testing"

import "github.com/stretchr/testify/require"

func TestAccountOpenAIEndpointCapabilities_ParseCompatibilityFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  any
		want []OpenAIEndpointCapability
	}{
		{name: "string aliases", raw: "chat, embedding", want: []OpenAIEndpointCapability{OpenAIEndpointCapabilityChatCompletions, OpenAIEndpointCapabilityEmbeddings}},
		{name: "json array string", raw: `["embeddings","chat.completions"]`, want: []OpenAIEndpointCapability{OpenAIEndpointCapabilityEmbeddings, OpenAIEndpointCapabilityChatCompletions}},
		{name: "any slice dedupes and ignores unknown", raw: []any{"embeddings", "openai.embeddings", "unknown"}, want: []OpenAIEndpointCapability{OpenAIEndpointCapabilityEmbeddings}},
		{name: "responses alias", raw: "responses_api", want: []OpenAIEndpointCapability{OpenAIEndpointCapabilityResponses}},
		{name: "empty string", raw: " ", want: nil},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{Credentials: map[string]any{openAIEndpointCapabilitiesCredentialKey: tt.raw}}
			require.Equal(t, tt.want, account.GetOpenAIEndpointCapabilities())
		})
	}
}

func TestSupportsOpenAIEndpointCapability(t *testing.T) {
	t.Parallel()

	apiKey := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "sk-test"}}
	oauth := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "token"}}
	deepSeek := &Account{Platform: PlatformDeepSeek, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "deepseek"}}

	require.True(t, SupportsOpenAIEndpointCapability(apiKey, OpenAIEndpointCapabilityChatCompletions))
	require.True(t, SupportsOpenAIEndpointCapability(apiKey, OpenAIEndpointCapabilityEmbeddings))
	require.True(t, SupportsOpenAIEndpointCapability(apiKey, OpenAIEndpointCapabilityAlphaSearch))
	require.True(t, SupportsOpenAIEndpointCapability(apiKey, OpenAIEndpointCapabilityResponses))
	require.True(t, SupportsOpenAIEndpointCapability(oauth, OpenAIEndpointCapabilityChatCompletions))
	require.False(t, SupportsOpenAIEndpointCapability(oauth, OpenAIEndpointCapabilityEmbeddings))
	require.True(t, SupportsOpenAIEndpointCapability(oauth, OpenAIEndpointCapabilityAlphaSearch))
	require.True(t, SupportsOpenAIEndpointCapability(oauth, OpenAIEndpointCapabilityResponses))
	require.True(t, SupportsOpenAIEndpointCapability(deepSeek, OpenAIEndpointCapabilityChatCompletions))
	require.False(t, SupportsOpenAIEndpointCapability(deepSeek, OpenAIEndpointCapabilityEmbeddings))
	require.False(t, SupportsOpenAIEndpointCapability(deepSeek, OpenAIEndpointCapabilityAlphaSearch))
	require.True(t, SupportsOpenAIEndpointCapability(deepSeek, OpenAIEndpointCapabilityResponses))

	restricted := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", openAIEndpointCapabilitiesCredentialKey: []string{string(OpenAIEndpointCapabilityEmbeddings)}},
	}
	require.True(t, SupportsOpenAIEndpointCapability(restricted, OpenAIEndpointCapabilityEmbeddings))
	require.False(t, SupportsOpenAIEndpointCapability(restricted, OpenAIEndpointCapabilityChatCompletions))
	require.False(t, SupportsOpenAIEndpointCapability(restricted, OpenAIEndpointCapabilityResponses))

	chatRestricted := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", openAIEndpointCapabilitiesCredentialKey: []string{string(OpenAIEndpointCapabilityChatCompletions)}},
	}
	require.True(t, SupportsOpenAIEndpointCapability(chatRestricted, OpenAIEndpointCapabilityResponses))

	responsesUnsupported := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{openAIResponsesSupportedExtraKey: false},
	}
	require.False(t, SupportsOpenAIEndpointCapability(responsesUnsupported, OpenAIEndpointCapabilityResponses))
}

func TestGrokMediaGenerationEligibility(t *testing.T) {
	t.Parallel()

	forbiddenBilling := map[string]any{"status_code": float64(403)}

	tests := []struct {
		name       string
		account    *Account
		want       bool
		wantReason string
	}{
		{name: "nil", account: nil, want: false, wantReason: "not_grok"},
		{name: "non grok", account: &Account{Platform: PlatformOpenAI}, want: false, wantReason: "not_grok"},
		{name: "api key", account: &Account{Platform: PlatformGrok, Type: AccountTypeAPIKey}, want: true, wantReason: "non_oauth"},
		{name: "unobserved oauth", account: &Account{Platform: PlatformGrok, Type: AccountTypeOAuth}, want: true, wantReason: "billing_unobserved"},
		{name: "eligible billing", account: &Account{Platform: PlatformGrok, Type: AccountTypeOAuth, Extra: map[string]any{grokBillingExtraKey: map[string]any{"status_code": float64(200)}}}, want: true, wantReason: "eligible"},
		{name: "forbidden billing", account: &Account{Platform: PlatformGrok, Type: AccountTypeOAuth, Extra: map[string]any{grokBillingExtraKey: forbiddenBilling}}, want: false, wantReason: "billing_forbidden"},
		{name: "override disabled", account: &Account{Platform: PlatformGrok, Type: AccountTypeOAuth, Extra: map[string]any{GrokMediaEligibleExtraKey: false}}, want: false, wantReason: "override_disabled"},
		{name: "override enabled", account: &Account{Platform: PlatformGrok, Type: AccountTypeOAuth, Extra: map[string]any{GrokMediaEligibleExtraKey: true, grokBillingExtraKey: forbiddenBilling}}, want: true, wantReason: "override_enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := tt.account.GrokMediaGenerationEligibility()
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantReason, reason)
		})
	}
}

func TestSupportsOpenAIEndpointCapability_GrokMediaOnlyFiltersMedia(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{GrokMediaEligibleExtraKey: false},
	}

	require.True(t, SupportsOpenAIEndpointCapability(account, OpenAIEndpointCapabilityChatCompletions))
	require.False(t, SupportsOpenAIEndpointCapability(account, OpenAIEndpointCapabilityGrokMedia))
}
