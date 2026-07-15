package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrokJoinVersionedEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		endpoint string
		want     string
	}{
		{
			name:     "official root base",
			baseURL:  "https://api.x.ai",
			endpoint: "/v1/responses",
			want:     "https://api.x.ai/v1/responses",
		},
		{
			name:     "official v1 base",
			baseURL:  "https://api.x.ai/v1",
			endpoint: "/v1/responses",
			want:     "https://api.x.ai/v1/responses",
		},
		{
			name:     "custom v1 base",
			baseURL:  "https://grok.example.test/v1/",
			endpoint: "/v1/chat/completions",
			want:     "https://grok.example.test/v1/chat/completions",
		},
		{
			name:     "relay nested v1 base",
			baseURL:  "https://relay.example.test/xai/v1",
			endpoint: "/v1/models",
			want:     "https://relay.example.test/xai/v1/models",
		},
		{
			name:     "cli base",
			baseURL:  defaultGrokCLIBaseURL,
			endpoint: "/v1/responses",
			want:     "https://cli-chat-proxy.grok.com/v1/responses",
		},
		{
			name:     "non versioned custom base keeps existing path",
			baseURL:  "https://relay.example.test/xai",
			endpoint: "/v1/responses",
			want:     "https://relay.example.test/xai/v1/responses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grokJoinVersionedEndpoint(tt.baseURL, tt.endpoint)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
