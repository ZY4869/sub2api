//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecideGeminiSuccessUsagePersistence(t *testing.T) {
	testCases := []struct {
		name          string
		inbound       string
		body          []byte
		wantPersist   bool
		wantReason    string
		wantOperation string
	}{
		{
			name:          "generate content persists",
			inbound:       "/v1beta/models/gemini-2.5-pro:generateContent",
			body:          []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
			wantPersist:   true,
			wantReason:    "generate_content",
			wantOperation: "generate_content",
		},
		{
			name:          "embeddings persist",
			inbound:       "/v1beta/models/text-embedding-004:embedContent",
			body:          []byte(`{"content":{"parts":[{"text":"hi"}]}}`),
			wantPersist:   true,
			wantReason:    "embeddings",
			wantOperation: "embeddings",
		},
		{
			name:          "live session persists",
			inbound:       "/v1beta/live",
			body:          []byte(`{"model":"models/gemini-live-2.5-flash"}`),
			wantPersist:   true,
			wantReason:    "live_session",
			wantOperation: "live_session",
		},
		{
			name:          "models list skipped",
			inbound:       "/v1beta/models",
			wantPersist:   false,
			wantReason:    "control_plane_models",
			wantOperation: "models",
		},
		{
			name:          "auth tokens skipped",
			inbound:       "/v1beta/live/auth-token",
			wantPersist:   false,
			wantReason:    "control_plane_auth_tokens",
			wantOperation: "auth_tokens",
		},
		{
			name:          "operation polling skipped",
			inbound:       "/v1beta/models/gemini-2.5-pro/operations/123",
			wantPersist:   false,
			wantReason:    "control_plane_operation_status",
			wantOperation: "operation_status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decision := DecideGeminiSuccessUsagePersistence(tc.inbound, tc.body)
			require.Equal(t, tc.wantPersist, decision.Persist)
			require.Equal(t, tc.wantReason, decision.Reason)
			require.Equal(t, tc.wantOperation, decision.OperationType)
		})
	}
}
