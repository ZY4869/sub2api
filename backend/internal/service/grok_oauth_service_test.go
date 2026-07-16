package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type grokOAuthClientStub struct {
	lastCode        string
	lastVerifier    string
	lastRedirectURI string
	lastClientID    string
	lastScope       string
	startProxyURL   string
	pollProxyURL    string
	devicePollErrs  []error
	devicePollCount int
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

func (s *grokOAuthClientStub) StartDeviceFlow(ctx context.Context, deviceURL string, clientID string, scope string, proxyURL string) (*grokoauth.DeviceAuthorizationResponse, error) {
	s.lastClientID = clientID
	s.lastScope = scope
	s.startProxyURL = proxyURL
	return &grokoauth.DeviceAuthorizationResponse{
		DeviceCode:              "device-secret",
		UserCode:                "ABCD-EFGH",
		VerificationURI:         "https://auth.x.ai/activate",
		VerificationURIComplete: "https://auth.x.ai/activate?user_code=ABCD-EFGH",
		ExpiresIn:               600,
		Interval:                1,
	}, nil
}

func (s *grokOAuthClientStub) PollDeviceToken(ctx context.Context, tokenURL string, deviceCode string, clientID string, proxyURL string) (*grokoauth.TokenResponse, error) {
	s.devicePollCount++
	s.lastClientID = clientID
	s.pollProxyURL = proxyURL
	if len(s.devicePollErrs) > 0 {
		err := s.devicePollErrs[0]
		s.devicePollErrs = s.devicePollErrs[1:]
		return nil, err
	}
	return &grokoauth.TokenResponse{
		AccessToken:  "device-access",
		RefreshToken: "device-refresh",
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

type grokProxyRepoStub struct {
	proxies map[int64]*Proxy
}

func (r *grokProxyRepoStub) Create(context.Context, *Proxy) error { return nil }
func (r *grokProxyRepoStub) GetByID(_ context.Context, id int64) (*Proxy, error) {
	if proxy := r.proxies[id]; proxy != nil {
		return proxy, nil
	}
	return nil, fmt.Errorf("proxy not found")
}
func (r *grokProxyRepoStub) ListByIDs(context.Context, []int64) ([]Proxy, error) { return nil, nil }
func (r *grokProxyRepoStub) Update(context.Context, *Proxy) error                { return nil }
func (r *grokProxyRepoStub) Delete(context.Context, int64) error                 { return nil }
func (r *grokProxyRepoStub) List(context.Context, pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *grokProxyRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *grokProxyRepoStub) ListWithFiltersAndAccountCount(context.Context, pagination.PaginationParams, string, string, string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *grokProxyRepoStub) ListActive(context.Context) ([]Proxy, error) { return nil, nil }
func (r *grokProxyRepoStub) ListActiveWithAccountCount(context.Context) ([]ProxyWithAccountCount, error) {
	return nil, nil
}
func (r *grokProxyRepoStub) ExistsByHostPortAuth(context.Context, string, int, string, string) (bool, error) {
	return false, nil
}
func (r *grokProxyRepoStub) CountAccountsByProxyID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (r *grokProxyRepoStub) ListAccountSummariesByProxyID(context.Context, int64) ([]ProxyAccountSummary, error) {
	return nil, nil
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
	require.Contains(t, query.Get("scope"), "grok-cli:access")
	require.Contains(t, query.Get("scope"), "api:access")
	require.Equal(t, result.State, query.Get("state"))
	require.NotEmpty(t, query.Get("nonce"))
	require.Equal(t, "generic", query.Get("plan"))
	require.Equal(t, "sub2api", query.Get("referrer"))
	require.Equal(t, "S256", query.Get("code_challenge_method"))
	require.NotEmpty(t, query.Get("code_challenge"))
}

func TestGrokOAuthService_GenerateAuthURL_RejectsUntrustedAuthorizeURL(t *testing.T) {
	svc := NewGrokOAuthService(nil, &grokOAuthClientStub{}, &config.Config{})
	svc.cfg.Grok.OAuth.AuthorizeURL = "https://evil.example/oauth2/authorize"

	_, err := svc.GenerateAuthURL(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "GROK_OAUTH_AUTH_URL_FAILED")
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
	require.Equal(t, defaultGrokCLIBaseURL, credentials["base_url"])
	require.Equal(t, "grok@example.com", credentials["email"])
}

func TestGrokOAuthService_RuntimeSanityReportsInvalidOverrides(t *testing.T) {
	cfg := &config.Config{}
	cfg.Grok.OAuth.TokenURL = "https://evil.example/oauth2/token"
	svc := NewGrokOAuthService(nil, &grokOAuthClientStub{}, cfg)

	report := svc.RuntimeSanity()

	require.True(t, report.BaseURL.Valid)
	require.Equal(t, defaultGrokCLIBaseURL, report.BaseURL.Value)
	require.False(t, report.OAuthTokenURL.Valid)
	require.Contains(t, report.OAuthTokenURL.Error, "host")
	require.Contains(t, report.PublicPaths, "/grok/v1/responses")
	require.Contains(t, report.PublicPaths, "/v1/responses")
}

func TestGrokOAuthService_BuildAccountExtra_DefaultsToGrokBuildTextScope(t *testing.T) {
	svc := NewGrokOAuthService(nil, &grokOAuthClientStub{}, &config.Config{})

	extra := svc.BuildAccountExtra(&GrokTokenInfo{
		Email:   "grok@example.com",
		Subject: "user-1",
		Name:    "Grok User",
	})

	require.Equal(t, "xai", extra["provider"])
	require.Equal(t, "grok_browser_oauth", extra["source"])
	require.Equal(t, "grok@example.com", extra["email"])

	scope, ok := ExtractAccountModelScopeV2(extra)
	require.True(t, ok)
	require.Equal(t, AccountModelPolicyModeWhitelist, scope.PolicyMode)
	require.Len(t, scope.Entries, len(GrokBuildTextModelIDs()))
	entriesByModel := make(map[string]AccountModelScopeEntry, len(scope.Entries))
	for _, entry := range scope.Entries {
		entriesByModel[entry.DisplayModelID] = entry
	}
	for _, modelID := range GrokBuildTextModelIDs() {
		entry, exists := entriesByModel[modelID]
		require.True(t, exists)
		require.Equal(t, modelID, entry.TargetModelID)
		require.Equal(t, PlatformGrok, entry.Provider)
		require.Equal(t, AccountModelVisibilityModeDefault, entry.VisibilityMode)
	}
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

func TestGrokOAuthService_ExchangeCode_RejectsDeviceShortCode(t *testing.T) {
	svc := NewGrokOAuthService(nil, &grokOAuthClientStub{}, &config.Config{})
	authURL, err := svc.GenerateAuthURL(context.Background(), &GrokGenerateAuthURLInput{})
	require.NoError(t, err)

	_, err = svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID: authURL.SessionID,
		Code:      "ABCD-EFGH",
		State:     authURL.State,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "GROK_OAUTH_DEVICE_CODE_NOT_EXCHANGEABLE")
}

func TestGrokOAuthService_DeviceFlow_PendingThenAuthorized(t *testing.T) {
	client := &grokOAuthClientStub{
		devicePollErrs: []error{&grokoauth.DeviceTokenError{Status: "authorization_pending"}},
	}
	svc := NewGrokOAuthService(nil, client, &config.Config{})

	started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{})
	require.NoError(t, err)
	require.Equal(t, "ABCD-EFGH", started.UserCode)
	require.Equal(t, "https://auth.x.ai/activate", started.VerificationURI)
	require.Equal(t, 1, started.Interval)

	pending, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.NoError(t, err)
	require.Equal(t, GrokDeviceStatusPending, pending.Status)
	require.Nil(t, pending.TokenInfo)

	forceGrokDevicePollReady(t, svc, started.SessionID)
	authorized, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.NoError(t, err)
	require.Equal(t, GrokDeviceStatusAuthorized, authorized.Status)
	require.NotNil(t, authorized.TokenInfo)
	require.Equal(t, "device-access", authorized.TokenInfo.AccessToken)
	require.Equal(t, "grok@example.com", authorized.TokenInfo.Email)
	require.Equal(t, 2, client.devicePollCount)

	_, err = svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.Error(t, err)
}

func TestGrokOAuthService_DeviceFlow_LocalSlowDownDoesNotPollUpstream(t *testing.T) {
	client := &grokOAuthClientStub{
		devicePollErrs: []error{&grokoauth.DeviceTokenError{Status: "authorization_pending"}},
	}
	svc := NewGrokOAuthService(nil, client, &config.Config{})

	started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{})
	require.NoError(t, err)
	pending, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.NoError(t, err)
	require.Equal(t, GrokDeviceStatusPending, pending.Status)

	slowDown, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.NoError(t, err)
	require.Equal(t, GrokDeviceStatusSlowDown, slowDown.Status)
	require.Equal(t, started.Interval, slowDown.Interval)
	require.Equal(t, 1, client.devicePollCount)
}

func TestGrokOAuthService_DeviceFlow_SlowDownIncreasesInterval(t *testing.T) {
	client := &grokOAuthClientStub{
		devicePollErrs: []error{&grokoauth.DeviceTokenError{Status: "slow_down"}},
	}
	svc := NewGrokOAuthService(nil, client, &config.Config{})

	started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{})
	require.NoError(t, err)
	result, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.NoError(t, err)
	require.Equal(t, GrokDeviceStatusSlowDown, result.Status)
	require.Equal(t, 6, result.Interval)
}

func TestGrokOAuthService_DeviceFlow_DeniedExpiredAndSessionExpired(t *testing.T) {
	for _, tt := range []struct {
		name       string
		errStatus  string
		wantStatus string
	}{
		{name: "denied", errStatus: "access_denied", wantStatus: GrokDeviceStatusDenied},
		{name: "expired token", errStatus: "expired_token", wantStatus: GrokDeviceStatusExpired},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &grokOAuthClientStub{
				devicePollErrs: []error{&grokoauth.DeviceTokenError{Status: tt.errStatus}},
			}
			svc := NewGrokOAuthService(nil, client, &config.Config{})
			started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{})
			require.NoError(t, err)

			result, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, result.Status)

			_, err = svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
			require.Error(t, err)
		})
	}

	t.Run("local session expired returns expired once", func(t *testing.T) {
		client := &grokOAuthClientStub{}
		svc := NewGrokOAuthService(nil, client, &config.Config{})
		started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{})
		require.NoError(t, err)
		forceGrokDeviceExpired(t, svc, started.SessionID)

		result, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
		require.NoError(t, err)
		require.Equal(t, GrokDeviceStatusExpired, result.Status)
		require.Equal(t, 0, client.devicePollCount)
	})
}

func TestGrokOAuthService_DeviceFlow_UsesSessionProxyAndPollOverride(t *testing.T) {
	client := &grokOAuthClientStub{}
	proxyRepo := &grokProxyRepoStub{proxies: map[int64]*Proxy{
		1: {Protocol: "http", Host: "proxy-one.test", Port: 8080, Username: "u", Password: "p"},
		2: {Protocol: "http", Host: "proxy-two.test", Port: 8081},
	}}
	svc := NewGrokOAuthService(proxyRepo, client, &config.Config{})
	proxyID := int64(1)
	overrideID := int64(2)

	started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{ProxyID: &proxyID})
	require.NoError(t, err)
	require.Equal(t, "http://u:p@proxy-one.test:8080", client.startProxyURL)

	authorized, err := svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID, ProxyID: &overrideID})
	require.NoError(t, err)
	require.Equal(t, GrokDeviceStatusAuthorized, authorized.Status)
	require.Equal(t, "http://proxy-two.test:8081", client.pollProxyURL)
}

func TestGrokOAuthService_DeviceFlow_ClientErrorPropagates(t *testing.T) {
	client := &grokOAuthClientStub{
		devicePollErrs: []error{errors.New("network")},
	}
	svc := NewGrokOAuthService(nil, client, &config.Config{})

	started, err := svc.StartDeviceFlow(context.Background(), &GrokStartDeviceFlowInput{})
	require.NoError(t, err)
	_, err = svc.PollDeviceToken(context.Background(), &GrokPollDeviceTokenInput{SessionID: started.SessionID})
	require.ErrorContains(t, err, "network")
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
	require.Equal(t, defaultGrokCLIBaseURL, info.BaseURL)

	credentials := svc.BuildAccountCredentials(info)
	require.Equal(t, defaultGrokCLIBaseURL, credentials["base_url"])
}

func TestGrokTokenRefresherRefreshStoresCLIBaseURL(t *testing.T) {
	client := &grokOAuthClientStub{}
	svc := NewGrokOAuthService(nil, client, &config.Config{})
	refresher := NewGrokTokenRefresher(svc)

	credentials, err := refresher.Refresh(context.Background(), &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "old-access",
			"refresh_token": "refresh-token",
			"client_id":     "client-1",
			"scope":         "openid api:access",
			"base_url":      "https://api.x.ai/v1",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "refreshed-access", credentials["access_token"])
	require.Equal(t, defaultGrokCLIBaseURL, credentials["base_url"])
}

func forceGrokDevicePollReady(t *testing.T, svc *GrokOAuthService, sessionID string) {
	t.Helper()
	session, ok := svc.sessionStore.Get(sessionID)
	require.True(t, ok)
	session.DeviceLastPollAt = time.Now().Add(-time.Duration(session.DeviceIntervalSeconds+1) * time.Second)
	svc.sessionStore.Set(sessionID, session)
}

func forceGrokDeviceExpired(t *testing.T, svc *GrokOAuthService, sessionID string) {
	t.Helper()
	session, ok := svc.sessionStore.Get(sessionID)
	require.True(t, ok)
	session.DeviceExpiresAt = time.Now().Add(-time.Second)
	svc.sessionStore.Set(sessionID, session)
}
