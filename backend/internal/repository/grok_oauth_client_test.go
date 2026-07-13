package repository

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarizeGrokTokenError_BadRequestUsesReauthHint(t *testing.T) {
	message := summarizeGrokTokenError(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"secret upstream detail"}`)

	require.Contains(t, message, "authorization code expired or already used")
	require.NotContains(t, message, "secret upstream detail")
	require.NotContains(t, message, "invalid_grant")
}
