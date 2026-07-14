package repository

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarizeGrokTokenError_InvalidGrantKeepsSanitizedClassification(t *testing.T) {
	message := summarizeGrokTokenError(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"authorization code expired"}`)

	require.Contains(t, message, "invalid, expired, already used")
	require.Contains(t, message, "authorization code expired")
	require.NotContains(t, message, "access_token")
}

func TestSummarizeGrokTokenError_RedirectAndPKCEMismatchHints(t *testing.T) {
	redirectMessage := summarizeGrokTokenError(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"redirect_uri mismatch"}`)
	require.Contains(t, redirectMessage, "redirect URI mismatch")

	pkceMessage := summarizeGrokTokenError(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"PKCE code_verifier mismatch"}`)
	require.Contains(t, pkceMessage, "PKCE verifier mismatch")
}
