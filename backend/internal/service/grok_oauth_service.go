package service

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
)

type GrokOAuthService struct {
	sessionStore *grokoauth.SessionStore
	proxyRepo    ProxyRepository
	oauthClient  GrokOAuthClient
	cfg          *config.Config
}

func NewGrokOAuthService(proxyRepo ProxyRepository, oauthClient GrokOAuthClient, cfg *config.Config) *GrokOAuthService {
	return &GrokOAuthService{
		sessionStore: grokoauth.NewSessionStore(),
		proxyRepo:    proxyRepo,
		oauthClient:  oauthClient,
		cfg:          cfg,
	}
}

type GrokAuthURLResult struct {
	AuthURL     string `json:"auth_url"`
	SessionID   string `json:"session_id"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

type GrokGenerateAuthURLInput struct {
	ProxyID     *int64
	RedirectURI string
	BaseURL     string
}

type GrokExchangeCodeInput struct {
	SessionID   string
	Code        string
	State       string
	RedirectURI string
	ProxyID     *int64
}

type GrokTokenInfo struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token,omitempty"`
	IDToken       string `json:"id_token,omitempty"`
	TokenType     string `json:"token_type,omitempty"`
	ExpiresIn     int64  `json:"expires_in,omitempty"`
	ExpiresAt     int64  `json:"expires_at,omitempty"`
	Scope         string `json:"scope,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	Email         string `json:"email,omitempty"`
	Subject       string `json:"subject,omitempty"`
	Name          string `json:"name,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

func (s *GrokOAuthService) GenerateAuthURL(ctx context.Context, input *GrokGenerateAuthURLInput) (*GrokAuthURLResult, error) {
	if s == nil || s.oauthClient == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_UNAVAILABLE", "Grok OAuth service is unavailable")
	}
	state, err := grokoauth.GenerateState()
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_STATE_FAILED", "failed to generate Grok OAuth state").WithCause(err)
	}
	verifier, err := grokoauth.GenerateCodeVerifier()
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_VERIFIER_FAILED", "failed to generate Grok OAuth verifier").WithCause(err)
	}
	sessionID, err := grokoauth.GenerateSessionID()
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_OAUTH_SESSION_FAILED", "failed to generate Grok OAuth session").WithCause(err)
	}

	redirectURI := firstNonEmptyString(strings.TrimSpace(input.RedirectURI), s.oauthRedirectURI())
	baseURL := s.oauthBaseURL(input.BaseURL)
	proxyURL, err := s.resolveProxyURL(ctx, input.ProxyID)
	if err != nil {
		return nil, err
	}
	clientID := s.oauthClientID()
	scope := s.oauthScope()
	authURL, err := grokoauth.BuildAuthorizationURL(s.oauthAuthorizeURL(), clientID, scope, redirectURI, state, grokoauth.GenerateCodeChallenge(verifier))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_AUTH_URL_FAILED", "failed to build Grok OAuth authorization URL").WithCause(err)
	}

	s.sessionStore.Set(sessionID, &grokoauth.OAuthSession{
		State:        state,
		CodeVerifier: verifier,
		ClientID:     clientID,
		Scope:        scope,
		RedirectURI:  redirectURI,
		ProxyURL:     proxyURL,
		BaseURL:      baseURL,
		CreatedAt:    time.Now(),
	})
	return &GrokAuthURLResult{
		AuthURL:     authURL,
		SessionID:   sessionID,
		RedirectURI: redirectURI,
		State:       state,
	}, nil
}

func (s *GrokOAuthService) ExchangeCode(ctx context.Context, input *GrokExchangeCodeInput) (*GrokTokenInfo, error) {
	session, ok := s.sessionStore.Get(strings.TrimSpace(input.SessionID))
	if !ok {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_SESSION_NOT_FOUND", "Grok OAuth session not found or expired")
	}
	codeInput := grokoauth.ParseAuthorizationInput(input.Code)
	code := firstNonEmptyString(codeInput.Code, input.Code)
	state := firstNonEmptyString(input.State, codeInput.State)
	if strings.TrimSpace(code) == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_CODE_REQUIRED", "Grok OAuth code is required")
	}
	if strings.TrimSpace(state) == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_STATE_REQUIRED", "Grok OAuth state is required")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(state)), []byte(session.State)) != 1 {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_INVALID_STATE", "invalid Grok OAuth state")
	}

	proxyURL := session.ProxyURL
	if input.ProxyID != nil {
		var err error
		proxyURL, err = s.resolveProxyURL(ctx, input.ProxyID)
		if err != nil {
			return nil, err
		}
	}
	redirectURI := firstNonEmptyString(input.RedirectURI, session.RedirectURI)
	tokenResp, err := s.oauthClient.ExchangeCode(ctx, s.oauthTokenURL(), strings.TrimSpace(code), session.CodeVerifier, redirectURI, session.ClientID, proxyURL)
	if err != nil {
		return nil, err
	}
	tokenInfo := s.tokenInfoFromResponse(tokenResp, session.ClientID, session.Scope, session.BaseURL)
	s.enrichUserInfo(ctx, tokenInfo, proxyURL)
	s.sessionStore.Delete(strings.TrimSpace(input.SessionID))
	return tokenInfo, nil
}

func (s *GrokOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*GrokTokenInfo, error) {
	if account == nil || account.Platform != PlatformGrok || account.Type != AccountTypeOAuth {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_INVALID_ACCOUNT", "account is not a Grok OAuth account")
	}
	refreshToken := strings.TrimSpace(account.GetCredential("refresh_token"))
	if refreshToken == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_REFRESH_TOKEN_MISSING", "Grok OAuth account does not have a refresh_token")
	}
	proxyURL, err := s.resolveProxyURL(ctx, account.ProxyID)
	if err != nil {
		return nil, err
	}
	clientID := firstNonEmptyString(account.GetCredential("client_id"), s.oauthClientID())
	scope := firstNonEmptyString(account.GetCredential("scope"), s.oauthScope())
	tokenResp, err := s.oauthClient.RefreshToken(ctx, s.oauthTokenURL(), refreshToken, clientID, scope, proxyURL)
	if err != nil {
		return nil, err
	}
	tokenInfo := s.tokenInfoFromResponse(tokenResp, clientID, scope, firstNonEmptyString(account.GetCredential("base_url"), s.oauthBaseURL("")))
	tokenInfo.Email = account.GetCredential("email")
	tokenInfo.Subject = account.GetCredential("subject")
	tokenInfo.Name = account.GetCredential("name")
	s.enrichUserInfo(ctx, tokenInfo, proxyURL)
	return tokenInfo, nil
}

func (s *GrokOAuthService) BuildAccountCredentials(tokenInfo *GrokTokenInfo) map[string]any {
	creds := map[string]any{
		"access_token": strings.TrimSpace(tokenInfo.AccessToken),
		"base_url":     strings.TrimRight(firstNonEmptyString(tokenInfo.BaseURL, s.oauthBaseURL("")), "/"),
	}
	if tokenInfo.ExpiresAt > 0 {
		creds["expires_at"] = time.Unix(tokenInfo.ExpiresAt, 0).UTC().Format(time.RFC3339)
	}
	if v := strings.TrimSpace(tokenInfo.RefreshToken); v != "" {
		creds["refresh_token"] = v
	}
	if v := strings.TrimSpace(tokenInfo.IDToken); v != "" {
		creds["id_token"] = v
	}
	if v := strings.TrimSpace(tokenInfo.TokenType); v != "" {
		creds["token_type"] = v
	}
	if v := strings.TrimSpace(tokenInfo.Scope); v != "" {
		creds["scope"] = v
	}
	if v := strings.TrimSpace(tokenInfo.ClientID); v != "" {
		creds["client_id"] = v
	}
	if v := strings.TrimSpace(tokenInfo.Email); v != "" {
		creds["email"] = v
	}
	if v := strings.TrimSpace(tokenInfo.Subject); v != "" {
		creds["subject"] = v
	}
	if v := strings.TrimSpace(tokenInfo.Name); v != "" {
		creds["name"] = v
	}
	return creds
}

func (s *GrokOAuthService) BuildAccountExtra(tokenInfo *GrokTokenInfo) map[string]any {
	extra := map[string]any{
		"provider": "xai",
		"source":   "grok_browser_oauth",
	}
	if tokenInfo != nil {
		if v := strings.TrimSpace(tokenInfo.Email); v != "" {
			extra["email"] = v
		}
		if v := strings.TrimSpace(tokenInfo.Subject); v != "" {
			extra["subject"] = v
		}
		if v := strings.TrimSpace(tokenInfo.Name); v != "" {
			extra["display_name"] = v
		}
	}
	return extra
}

func (s *GrokOAuthService) Stop() {
	if s != nil && s.sessionStore != nil {
		s.sessionStore.Stop()
	}
}

func (s *GrokOAuthService) tokenInfoFromResponse(resp *grokoauth.TokenResponse, clientID string, scope string, baseURL string) *GrokTokenInfo {
	if resp == nil {
		resp = &grokoauth.TokenResponse{}
	}
	expiresIn := resp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = int64(time.Hour.Seconds())
	}
	return &GrokTokenInfo{
		AccessToken:  strings.TrimSpace(resp.AccessToken),
		RefreshToken: strings.TrimSpace(resp.RefreshToken),
		IDToken:      strings.TrimSpace(resp.IDToken),
		TokenType:    strings.TrimSpace(resp.TokenType),
		ExpiresIn:    expiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
		Scope:        firstNonEmptyString(resp.Scope, scope),
		ClientID:     strings.TrimSpace(clientID),
		BaseURL:      strings.TrimRight(firstNonEmptyString(baseURL, s.oauthBaseURL("")), "/"),
	}
}

func (s *GrokOAuthService) enrichUserInfo(ctx context.Context, tokenInfo *GrokTokenInfo, proxyURL string) {
	if tokenInfo == nil || strings.TrimSpace(tokenInfo.AccessToken) == "" || s.oauthClient == nil {
		return
	}
	userInfo, err := s.oauthClient.FetchUserInfo(ctx, s.oauthUserInfoURL(), tokenInfo.AccessToken, proxyURL)
	if err != nil || userInfo == nil {
		return
	}
	tokenInfo.Email = firstNonEmptyString(tokenInfo.Email, userInfo.Email)
	tokenInfo.Subject = firstNonEmptyString(tokenInfo.Subject, userInfo.Sub)
	tokenInfo.Name = firstNonEmptyString(tokenInfo.Name, userInfo.Name)
	tokenInfo.EmailVerified = userInfo.EmailVerified
}

func (s *GrokOAuthService) resolveProxyURL(ctx context.Context, proxyID *int64) (string, error) {
	if proxyID == nil || s == nil || s.proxyRepo == nil {
		return "", nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
	if err != nil {
		return "", infraerrors.New(http.StatusBadRequest, "GROK_OAUTH_PROXY_NOT_FOUND", "proxy not found").WithCause(err)
	}
	if proxy == nil {
		return "", nil
	}
	return strings.TrimSpace(proxy.URL()), nil
}

func (s *GrokOAuthService) oauthAuthorizeURL() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.AuthorizeURL) != "" {
		return strings.TrimSpace(s.cfg.Grok.OAuth.AuthorizeURL)
	}
	return grokoauth.DefaultAuthorizeURL
}

func (s *GrokOAuthService) oauthTokenURL() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.TokenURL) != "" {
		return strings.TrimSpace(s.cfg.Grok.OAuth.TokenURL)
	}
	return grokoauth.DefaultTokenURL
}

func (s *GrokOAuthService) oauthUserInfoURL() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.UserInfoURL) != "" {
		return strings.TrimSpace(s.cfg.Grok.OAuth.UserInfoURL)
	}
	return grokoauth.DefaultUserInfoURL
}

func (s *GrokOAuthService) oauthClientID() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.ClientID) != "" {
		return strings.TrimSpace(s.cfg.Grok.OAuth.ClientID)
	}
	return grokoauth.DefaultClientID
}

func (s *GrokOAuthService) oauthScope() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.Scopes) != "" {
		return strings.TrimSpace(s.cfg.Grok.OAuth.Scopes)
	}
	return grokoauth.DefaultScope
}

func (s *GrokOAuthService) oauthRedirectURI() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.RedirectURI) != "" {
		return strings.TrimSpace(s.cfg.Grok.OAuth.RedirectURI)
	}
	return grokoauth.DefaultRedirectURI
}

func (s *GrokOAuthService) oauthBaseURL(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return strings.TrimRight(trimmed, "/")
	}
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Grok.OAuth.BaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(s.cfg.Grok.OAuth.BaseURL), "/")
	}
	return grokoauth.DefaultBaseURL
}
