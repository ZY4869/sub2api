package service

import (
	"context"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
	"github.com/stretchr/testify/require"
)

type grokOAuthClientStub struct {
	lastCode        string
	lastVerifier    string
	lastRedirectURI string
	lastClientID    string
	lastScope       string
}

func (s *grokOAuthClientStub) ExchangeCode(ctx context.Context, tokenURL string, code string, codeVerifier string, redirectURI string, clientID string, proxyURL string) (*grokoauth.TokenResponse, error) {
	s.lastCode = code
	s.lastVerifier = codeVerifier
	s.lastRedirectURI = redirectURI
	s.lastClientID = clientID
	return &grokoauth.TokenResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        "openid profile email",
	}, nil
}

func (s *grokOAuthClientStub) RefreshToken(ctx context.Context, tokenURL string, refreshToken string, clientID string, scope string, proxyURL string) (*grokoauth.TokenResponse, error) {
	s.lastClientID = clientID
	s.lastScope = scope
	return &grokoauth.TokenResponse{
		AccessToken:  "refreshed-access",
		RefreshToken: "rotated-refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        scope,
	}, nil
}

func (s *grokOAuthClientStub) FetchUserInfo(ctx context.Context, userInfoURL string, accessToken string, proxyURL string) (*grokoauth.UserInfo, error) {
	return &grokoauth.UserInfo{
		Sub:           "user-1",
		Email:         "grok@example.com",
		Name:          "Grok User",
		EmailVerified: true,
	}, nil
}

func TestGrokOAuthService_GenerateAuthURL_UsesPKCEAndConfiguredDefaults(t *testing.T) {
	cfg := &config.Config{}
	svc := NewGrokOAuthService(nil, &grokOAuthClientStub{}, cfg)

	result, err := svc.GenerateAuthURL(context.Background(), &GrokGenerateAuthURLInput{})
	require.NoError(t, err)
	require.NotEmpty(t, result.SessionID)
	require.NotEmpty(t, result.State)
	require.Equal(t, grokoauth.DefaultRedirectURI, result.RedirectURI)

	parsed, err := url.Parse(result.AuthURL)
	require.NoError(t, err)
	query := parsed.Query()
	require.Equal(t, "code", query.Get("response_type"))
	require.Equal(t, grokoauth.DefaultClientID, query.Get("client_id"))
	require.Equal(t, grokoauth.DefaultRedirectURI, query.Get("redirect_uri"))
	require.Equal(t, grokoauth.DefaultScope, query.Get("scope"))
	require.Equal(t, result.State, query.Get("state"))
	require.Equal(t, "S256", query.Get("code_challenge_method"))
	require.NotEmpty(t, query.Get("code_challenge"))
}

func TestGrokOAuthService_ExchangeCode_AcceptsCallbackURLAndBuildsCredentials(t *testing.T) {
	client := &grokOAuthClientStub{}
	svc := NewGrokOAuthService(nil, client, &config.Config{})
	authURL, err := svc.GenerateAuthURL(context.Background(), &GrokGenerateAuthURLInput{})
	require.NoError(t, err)

	tokenInfo, err := svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID: authURL.SessionID,
		Code:      "http://127.0.0.1:56121/callback?code=oauth-code&state=" + authURL.State,
	})
	require.NoError(t, err)
	require.Equal(t, "oauth-code", client.lastCode)
	require.NotEmpty(t, client.lastVerifier)
	require.Equal(t, authURL.RedirectURI, client.lastRedirectURI)
	require.Equal(t, grokoauth.DefaultClientID, client.lastClientID)
	require.Equal(t, "grok@example.com", tokenInfo.Email)
	require.Equal(t, "user-1", tokenInfo.Subject)

	credentials := svc.BuildAccountCredentials(tokenInfo)
	require.Equal(t, "access-token", credentials["access_token"])
	require.Equal(t, "refresh-token", credentials["refresh_token"])
	require.Equal(t, "https://api.x.ai/v1", credentials["base_url"])
	require.Equal(t, "grok@example.com", credentials["email"])
}

func TestGrokOAuthService_ExchangeCode_RejectsInvalidState(t *testing.T) {
	svc := NewGrokOAuthService(nil, &grokOAuthClientStub{}, &config.Config{})
	authURL, err := svc.GenerateAuthURL(context.Background(), &GrokGenerateAuthURLInput{})
	require.NoError(t, err)

	_, err = svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID: authURL.SessionID,
		Code:      "oauth-code",
		State:     "wrong-state",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "GROK_OAUTH_INVALID_STATE")
}

func TestGrokOAuthService_RefreshAccountToken_UsesStoredClientAndScope(t *testing.T) {
	client := &grokOAuthClientStub{}
	svc := NewGrokOAuthService(nil, client, &config.Config{})
	info, err := svc.RefreshAccountToken(context.Background(), &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "refresh-token",
			"client_id":     "client-1",
			"scope":         "openid api:access",
			"base_url":      "https://api.x.ai/v1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "client-1", client.lastClientID)
	require.Equal(t, "openid api:access", client.lastScope)
	require.Equal(t, "refreshed-access", info.AccessToken)
}
