package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestResponsesWSPromptAuditCloseStatusContract(t *testing.T) {
	tests := []struct {
		name   string
		scan   securityaudit.PromptScanner
		want   coderws.StatusCode
		accept bool
	}{
		{
			name:   "block",
			scan:   &wsPromptAuditScanner{result: wsPromptAuditBlockResult()},
			want:   coderws.StatusCode(4403),
			accept: false,
		},
		{
			name:   "unavailable",
			scan:   &wsPromptAuditScanner{err: &securityaudit.GuardError{Code: securityaudit.ErrorCodeUnavailable, Retryable: true}},
			want:   coderws.StatusTryAgainLater,
			accept: false,
		},
		{
			name:   "invalid response",
			scan:   &wsPromptAuditScanner{},
			want:   coderws.StatusTryAgainLater,
			accept: false,
		},
		{
			name:   "allow",
			scan:   &wsPromptAuditScanner{result: wsPromptAuditPassResult()},
			want:   coderws.StatusNormalClosure,
			accept: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &OpenAIGatewayHandler{}
			h.SetPromptAuditService(newWSPromptAuditService(t, tt.scan))
			c := wsPromptAuditContext()
			apiKey := wsPromptAuditAPIKey()

			status, ok := h.auditResponsesWSFirstMessage(c, apiKey, []byte(`{"model":"gpt-5","input":"hello"}`), "gpt-5")

			require.Equal(t, tt.accept, ok)
			require.Equal(t, tt.want, status)
		})
	}
}

func TestResponsesWSPromptAuditScansSubsequentTurns(t *testing.T) {
	scanner := &wsPromptAuditSequenceScanner{
		results: []*securityaudit.NormalizedResult{
			wsPromptAuditPassResult(),
			wsPromptAuditBlockResult(),
		},
	}
	h := &OpenAIGatewayHandler{}
	h.SetPromptAuditService(newWSPromptAuditService(t, scanner))
	c := wsPromptAuditContext()
	apiKey := wsPromptAuditAPIKey()

	status, ok := h.auditResponsesWSTurn(c, apiKey, []byte(`{"model":"gpt-5","input":"safe first turn"}`), "gpt-5", "first_turn")
	require.True(t, ok)
	require.Equal(t, coderws.StatusNormalClosure, status)

	status, ok = h.auditResponsesWSTurn(c, apiKey, []byte(`{"model":"gpt-5","input":"blocked follow up"}`), "gpt-5", "subsequent_turn")
	require.False(t, ok)
	require.Equal(t, coderws.StatusCode(4403), status)
	require.Equal(t, 2, scanner.calls)
	require.Contains(t, scanner.chunks[0], "safe first turn")
	require.Contains(t, scanner.chunks[1], "blocked follow up")
}

func TestResponsesWSPromptAuditSubsequentTurnFailureStatus(t *testing.T) {
	tests := []struct {
		name string
		scan securityaudit.PromptScanner
	}{
		{
			name: "unavailable",
			scan: &wsPromptAuditSequenceScanner{
				results: []*securityaudit.NormalizedResult{wsPromptAuditPassResult()},
				errs:    []error{nil, &securityaudit.GuardError{Code: securityaudit.ErrorCodeUnavailable, Retryable: true}},
			},
		},
		{
			name: "invalid response",
			scan: &wsPromptAuditSequenceScanner{
				results: []*securityaudit.NormalizedResult{wsPromptAuditPassResult(), nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &OpenAIGatewayHandler{}
			h.SetPromptAuditService(newWSPromptAuditService(t, tt.scan))
			c := wsPromptAuditContext()
			apiKey := wsPromptAuditAPIKey()

			status, ok := h.auditResponsesWSTurn(c, apiKey, []byte(`{"model":"gpt-5","input":"safe first turn"}`), "gpt-5", "first_turn")
			require.True(t, ok)
			require.Equal(t, coderws.StatusNormalClosure, status)

			status, ok = h.auditResponsesWSTurn(c, apiKey, []byte(`{"model":"gpt-5","input":"guard unavailable"}`), "gpt-5", "subsequent_turn")
			require.False(t, ok)
			require.Equal(t, coderws.StatusTryAgainLater, status)
		})
	}
}
