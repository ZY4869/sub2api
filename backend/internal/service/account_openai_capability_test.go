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
	require.True(t, SupportsOpenAIEndpointCapability(oauth, OpenAIEndpointCapabilityChatCompletions))
	require.False(t, SupportsOpenAIEndpointCapability(oauth, OpenAIEndpointCapabilityEmbeddings))
	require.True(t, SupportsOpenAIEndpointCapability(deepSeek, OpenAIEndpointCapabilityChatCompletions))
	require.False(t, SupportsOpenAIEndpointCapability(deepSeek, OpenAIEndpointCapabilityEmbeddings))

	restricted := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", openAIEndpointCapabilitiesCredentialKey: []string{string(OpenAIEndpointCapabilityEmbeddings)}},
	}
	require.True(t, SupportsOpenAIEndpointCapability(restricted, OpenAIEndpointCapabilityEmbeddings))
	require.False(t, SupportsOpenAIEndpointCapability(restricted, OpenAIEndpointCapabilityChatCompletions))
}
