package xai

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAuthorizationURLAddsCLIParametersAndNonce(t *testing.T) {
	authURL, err := BuildAuthorizationURL("", "", "", "", "state-1", "challenge-1", "nonce-1")
	require.NoError(t, err)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	require.Equal(t, "auth.x.ai", parsed.Host)

	query := parsed.Query()
	require.Equal(t, "code", query.Get("response_type"))
	require.Equal(t, DefaultClientID, query.Get("client_id"))
	require.Equal(t, DefaultScope, query.Get("scope"))
	require.Equal(t, DefaultRedirectURI, query.Get("redirect_uri"))
	require.Equal(t, "state-1", query.Get("state"))
	require.Equal(t, "nonce-1", query.Get("nonce"))
	require.Equal(t, "challenge-1", query.Get("code_challenge"))
	require.Equal(t, "S256", query.Get("code_challenge_method"))
	require.Equal(t, "generic", query.Get("plan"))
	require.Equal(t, "sub2api", query.Get("referrer"))
}

func TestValidateOAuthEndpointURLRejectsUntrustedHost(t *testing.T) {
	_, err := ValidateOAuthEndpointURL("https://evil.example/oauth2/token")
	require.Error(t, err)

	tokenURL, err := ValidateOAuthEndpointURL(DefaultTokenURL)
	require.NoError(t, err)
	require.Equal(t, DefaultTokenURL, tokenURL)
}

func TestValidateTrustedBaseURLNormalizesKnownBases(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "api root", raw: "https://api.x.ai", want: "https://api.x.ai/v1"},
		{name: "api v1", raw: "https://api.x.ai/v1/", want: "https://api.x.ai/v1"},
		{name: "cli v1", raw: DefaultCLIBaseURL, want: DefaultCLIBaseURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateTrustedBaseURL(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	_, err := ValidateTrustedBaseURL("https://api.x.ai/custom")
	require.Error(t, err)
}
