package service

import (
	"context"
	"errors"
	"net/url"
	"testing"

	pkgkiro "github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
	"github.com/stretchr/testify/require"
)

type kiroOAuthClientStub struct{}

func (s *kiroOAuthClientStub) RegisterAuthCodeClient(ctx context.Context, redirectURI, issuerURL, region, proxyURL string) (*KiroClientRegistration, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthClientStub) ExchangeSocialCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL string) (*KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthClientStub) RefreshSocialToken(ctx context.Context, refreshToken, proxyURL string) (*KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthClientStub) ExchangeOIDCCode(ctx context.Context, clientID, clientSecret, code, codeVerifier, redirectURI, region, proxyURL string) (*KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthClientStub) RefreshOIDCToken(ctx context.Context, clientID, clientSecret, refreshToken, region, startURL, proxyURL string) (*KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthClientStub) FetchOIDCUserInfo(ctx context.Context, accessToken, region, proxyURL string) (*KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func TestKiroOAuthService_GenerateAuthURL_NormalizesRedirectURI(t *testing.T) {
	svc := NewKiroOAuthService(nil, &kiroOAuthClientStub{})

	result, err := svc.GenerateAuthURL(context.Background(), &KiroGenerateAuthURLInput{
		Method:      pkgkiro.OAuthMethodGitHub,
		RedirectURI: "http://localhost:19877/oauth/callback",
	})
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:19877/oauth/callback", result.RedirectURI)

	parsed, err := url.Parse(result.AuthURL)
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:19877/oauth/callback", parsed.Query().Get("redirect_uri"))
}

func TestKiroOAuthService_GenerateAuthURL_RejectsNonLoopbackRedirectURI(t *testing.T) {
	svc := NewKiroOAuthService(nil, &kiroOAuthClientStub{})

	_, err := svc.GenerateAuthURL(context.Background(), &KiroGenerateAuthURLInput{
		Method:      pkgkiro.OAuthMethodGitHub,
		RedirectURI: "https://sub2api.example.com/oauth/callback",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "KIRO_OAUTH_INVALID_REDIRECT_URI")
}

func TestKiroOAuthService_BuildAccountExtra_IncludesMembershipFields(t *testing.T) {
	svc := NewKiroOAuthService(nil, &kiroOAuthClientStub{})
	credits := 2000

	extra := svc.BuildAccountExtra(&KiroTokenInfo{
		Provider:      "aws",
		Email:         "user@example.com",
		Username:      "demo-user",
		DisplayName:   "Demo User",
		MemberLevel:   "kiro_pro_plus",
		MemberCredits: &credits,
	})

	require.Equal(t, "aws", extra["provider"])
	require.Equal(t, "kiro_browser_oauth", extra["source"])
	require.Equal(t, "user@example.com", extra["email"])
	require.Equal(t, "demo-user", extra["username"])
	require.Equal(t, "Demo User", extra["display_name"])
	require.Equal(t, "kiro_pro_plus", extra["kiro_member_level"])
	require.Equal(t, 2000, extra["kiro_member_credits"])
}

func TestKiroOAuthService_BuildAccountExtra_IgnoresInvalidMembershipFields(t *testing.T) {
	svc := NewKiroOAuthService(nil, &kiroOAuthClientStub{})
	invalidCredits := -10

	extra := svc.BuildAccountExtra(&KiroTokenInfo{
		Provider:      "",
		MemberLevel:   "invalid-tier",
		MemberCredits: &invalidCredits,
	})

	require.Equal(t, "kiro", extra["provider"])
	require.Equal(t, "kiro_browser_oauth", extra["source"])
	_, hasLevel := extra["kiro_member_level"]
	_, hasCredits := extra["kiro_member_credits"]
	require.False(t, hasLevel)
	require.False(t, hasCredits)
}
