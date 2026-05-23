package service

import (
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateOpsRetryBodySafety_AllowsPlainJSON(t *testing.T) {
	err := validateOpsRetryBodySafety(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hello"}]}`, false)
	require.NoError(t, err)
}

func TestValidateOpsRetryBodySafety_RejectsUnsafeBodies(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		truncated bool
		reason    string
	}{
		{
			name:   "empty",
			body:   "  ",
			reason: "OPS_RETRY_NO_REQUEST_BODY",
		},
		{
			name:      "truncated flag",
			body:      `{"model":"claude"}`,
			truncated: true,
			reason:    "OPS_RETRY_BODY_TRUNCATED",
		},
		{
			name:   "invalid json",
			body:   `{invalid-json`,
			reason: "OPS_RETRY_BODY_INVALID",
		},
		{
			name:   "redacted value",
			body:   `{"messages":[{"role":"user","content":"[REDACTED]"}]}`,
			reason: "OPS_RETRY_BODY_REDACTED",
		},
		{
			name:   "nested sensitive key",
			body:   `{"metadata":{"client_secret":"should-not-replay"},"messages":[]}`,
			reason: "OPS_RETRY_BODY_SENSITIVE",
		},
		{
			name:   "stored truncation marker",
			body:   `{"request_body_truncated":true,"messages":[]}`,
			reason: "OPS_RETRY_BODY_TRUNCATED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOpsRetryBodySafety(tc.body, tc.truncated)
			require.Error(t, err)
			require.Equal(t, tc.reason, infraerrors.Reason(err))
		})
	}
}
