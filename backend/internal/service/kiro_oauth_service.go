package service

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	pkgkiro "github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
)

type KiroAuthURLResult struct {
	AuthURL     string `json:"auth_url"`
	SessionID   string `json:"session_id"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

type KiroGenerateAuthURLInput struct {
	ProxyID     *int64
	RedirectURI string
	Method      string
	StartURL    string
	Region      string
}

type KiroExchangeCodeInput struct {
	SessionID string
	Code      string
	State     string
	ProxyID   *int64
}

type KiroTokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	AuthMethod   string `json:"auth_method,omitempty"`
	Provider     string `json:"provider,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	ClientIDHash string `json:"client_id_hash,omitempty"`
	StartURL     string `json:"start_url,omitempty"`
	Region       string `json:"region,omitempty"`
	ProfileArn   string `json:"profile_arn,omitempty"`
	Email        string `json:"email,omitempty"`
	Username     string `json:"username,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
}

type KiroClientRegistration struct {
	ClientID     string
	ClientSecret string
}

type KiroOAuthClient interface {
	RegisterAuthCodeClient(ctx context.Context, redirectURI, issuerURL, region, proxyURL string) (*KiroClientRegistration, error)
	ExchangeSocialCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL string) (*KiroTokenInfo, error)
	RefreshSocialToken(ctx context.Context, refreshToken, proxyURL string) (*KiroTokenInfo, error)
	ExchangeOIDCCode(ctx context.Context, clientID, clientSecret, code, codeVerifier, redirectURI, region, proxyURL string) (*KiroTokenInfo, error)
	RefreshOIDCToken(ctx context.Context, clientID, clientSecret, refreshToken, region, startURL, proxyURL string) (*KiroTokenInfo, error)
	FetchOIDCUserInfo(ctx context.Context, accessToken, region, proxyURL string) (*KiroTokenInfo, error)
}

type KiroOAuthService struct {
	sessionStore *pkgkiro.SessionStore
	proxyRepo    ProxyRepository
	oauthClient  KiroOAuthClient
}

func NewKiroOAuthService(proxyRepo ProxyRepository, oauthClient KiroOAuthClient) *KiroOAuthService {
	return &KiroOAuthService{
		sessionStore: pkgkiro.NewSessionStore(),
		proxyRepo:    proxyRepo,
		oauthClient:  oauthClient,
	}
}

func (s *KiroOAuthService) GenerateAuthURL(ctx context.Context, input *KiroGenerateAuthURLInput) (*KiroAuthURLResult, error) {
	method := normalizeRequestedKiroAuthMethod(input.Method)
	state, err := pkgkiro.GenerateState()
	if err != nil {
		return nil, infraerrors.InternalServer("KIRO_OAUTH_STATE_FAILED", "failed to generate kiro oauth state").WithCause(err)
	}
	sessionID, err := pkgkiro.GenerateSessionID()
	if err != nil {
		return nil, infraerrors.InternalServer("KIRO_OAUTH_SESSION_FAILED", "failed to generate kiro oauth session").WithCause(err)
	}
	codeVerifier, err := pkgkiro.GenerateCodeVerifier()
	if err != nil {
		return nil, infraerrors.InternalServer("KIRO_OAUTH_VERIFIER_FAILED", "failed to generate kiro oauth code verifier").WithCause(err)
	}
	redirectURI := strings.TrimSpace(input.RedirectURI)
	if redirectURI == "" {
		redirectURI = pkgkiro.DefaultRedirectURI
	}
	proxyURL, err := s.resolveProxyURL(ctx, input.ProxyID)
	if err != nil {
		return nil, err
	}

	session := &pkgkiro.OAuthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		ProxyURL:     proxyURL,
		Method:       method,
		Region:       normalizeKiroRequestedRegion(method, input.Region),
		StartURL:     strings.TrimSpace(input.StartURL),
		CreatedAt:    time.Now(),
	}

	authURL, err := s.buildAuthURL(ctx, session, pkgkiro.GenerateCodeChallenge(codeVerifier))
	if err != nil {
		return nil, err
	}

	s.sessionStore.Set(sessionID, session)
	return &KiroAuthURLResult{
		AuthURL:     authURL,
		SessionID:   sessionID,
		RedirectURI: redirectURI,
		State:       state,
	}, nil
}

func (s *KiroOAuthService) ExchangeCode(ctx context.Context, input *KiroExchangeCodeInput) (*KiroTokenInfo, error) {
	session, ok := s.sessionStore.Get(strings.TrimSpace(input.SessionID))
	if !ok {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_SESSION_NOT_FOUND", "kiro oauth session not found or expired")
	}
	if strings.TrimSpace(input.Code) == "" {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_CODE_REQUIRED", "kiro oauth code is required")
	}
	if strings.TrimSpace(input.State) == "" {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_STATE_REQUIRED", "kiro oauth state is required")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(input.State)), []byte(session.State)) != 1 {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_INVALID_STATE", "invalid kiro oauth state")
	}

	var tokenInfo *KiroTokenInfo
	var err error
	switch session.Method {
	case pkgkiro.OAuthMethodBuilder, pkgkiro.OAuthMethodIDC:
		tokenInfo, err = s.oauthClient.ExchangeOIDCCode(ctx, session.ClientID, session.ClientSecret, strings.TrimSpace(input.Code), session.CodeVerifier, session.RedirectURI, session.Region, session.ProxyURL)
	default:
		tokenInfo, err = s.oauthClient.ExchangeSocialCode(ctx, strings.TrimSpace(input.Code), session.CodeVerifier, session.RedirectURI, session.ProxyURL)
	}
	if err != nil {
		return nil, err
	}

	tokenInfo.AuthMethod = resolvedKiroAuthMethod(session.Method, tokenInfo.AuthMethod)
	tokenInfo.Provider = resolvedKiroProvider(session.Method, tokenInfo.Provider)
	tokenInfo.Region = resolvedKiroRegion(session.Method, session.Region, tokenInfo.Region)
	tokenInfo.StartURL = firstNonEmpty(tokenInfo.StartURL, session.StartURL)
	tokenInfo.ClientID = firstNonEmpty(tokenInfo.ClientID, session.ClientID)
	tokenInfo.ClientSecret = firstNonEmpty(tokenInfo.ClientSecret, session.ClientSecret)
	if tokenInfo.ProfileArn == "" {
		tokenInfo.ProfileArn = ""
	}
	s.maybeEnrichOIDCUserInfo(ctx, session, tokenInfo)
	s.sessionStore.Delete(strings.TrimSpace(input.SessionID))
	return tokenInfo, nil
}

func (s *KiroOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*KiroTokenInfo, error) {
	if account == nil || account.Platform != PlatformKiro || account.Type != AccountTypeOAuth {
		return nil, infraerrors.BadRequest("KIRO_INVALID_ACCOUNT", "account is not a kiro oauth account")
	}
	refreshToken := strings.TrimSpace(account.GetCredential("refresh_token"))
	if refreshToken == "" {
		return nil, infraerrors.BadRequest("KIRO_REFRESH_UNSUPPORTED", "kiro account does not have a refresh_token; please re-import or re-authorize")
	}

	authMethod := inferKiroRefreshAuthMethod(account)
	if authMethod == "" {
		return nil, infraerrors.BadRequest("KIRO_REFRESH_UNSUPPORTED", "kiro oauth account is missing auth metadata required for refresh")
	}
	proxyURL, err := s.resolveProxyURL(ctx, account.ProxyID)
	if err != nil {
		return nil, err
	}

	var tokenInfo *KiroTokenInfo
	if authMethod == pkgkiro.OAuthMethodBuilder || authMethod == pkgkiro.OAuthMethodIDC {
		clientID := strings.TrimSpace(account.GetCredential("client_id"))
		clientSecret := strings.TrimSpace(account.GetCredential("client_secret"))
		if clientID == "" || clientSecret == "" {
			return nil, infraerrors.BadRequest("KIRO_REFRESH_UNSUPPORTED", "kiro oauth account is missing client credentials required for refresh")
		}
		tokenInfo, err = s.oauthClient.RefreshOIDCToken(ctx, clientID, clientSecret, refreshToken, normalizeKiroRegion(account.GetCredential("region")), strings.TrimSpace(account.GetCredential("start_url")), proxyURL)
	} else {
		tokenInfo, err = s.oauthClient.RefreshSocialToken(ctx, refreshToken, proxyURL)
	}
	if err != nil {
		return nil, err
	}

	tokenInfo.AuthMethod = authMethod
	tokenInfo.Provider = resolvedKiroProvider(authMethod, account.GetExtraString("provider"))
	tokenInfo.ClientID = firstNonEmpty(tokenInfo.ClientID, account.GetCredential("client_id"))
	tokenInfo.ClientSecret = firstNonEmpty(tokenInfo.ClientSecret, account.GetCredential("client_secret"))
	tokenInfo.ClientIDHash = firstNonEmpty(tokenInfo.ClientIDHash, account.GetCredential("client_id_hash"))
	tokenInfo.StartURL = firstNonEmpty(tokenInfo.StartURL, account.GetCredential("start_url"))
	tokenInfo.Region = resolvedKiroRegion(authMethod, account.GetCredential("region"), tokenInfo.Region)
	tokenInfo.ProfileArn = firstNonEmpty(tokenInfo.ProfileArn, account.GetCredential("profile_arn"))
	s.maybeEnrichOIDCUserInfo(ctx, &pkgkiro.OAuthSession{Method: authMethod, Region: tokenInfo.Region, ProxyURL: proxyURL}, tokenInfo)
	return tokenInfo, nil
}

func (s *KiroOAuthService) BuildAccountCredentials(tokenInfo *KiroTokenInfo) map[string]any {
	creds := map[string]any{
		"access_token": strings.TrimSpace(tokenInfo.AccessToken),
	}
	if v := strings.TrimSpace(tokenInfo.RefreshToken); v != "" {
		creds["refresh_token"] = v
	}
	if v := strings.TrimSpace(tokenInfo.ExpiresAt); v != "" {
		creds["expires_at"] = v
	}
	if v := strings.TrimSpace(tokenInfo.AuthMethod); v != "" {
		creds["auth_method"] = v
	}
	if v := strings.TrimSpace(tokenInfo.ClientID); v != "" {
		creds["client_id"] = v
	}
	if v := strings.TrimSpace(tokenInfo.ClientSecret); v != "" {
		creds["client_secret"] = v
	}
	if v := strings.TrimSpace(tokenInfo.ClientIDHash); v != "" {
		creds["client_id_hash"] = v
	}
	if v := strings.TrimSpace(tokenInfo.StartURL); v != "" {
		creds["start_url"] = v
	}
	if v := strings.TrimSpace(tokenInfo.Region); v != "" {
		creds["region"] = v
	}
	if v := strings.TrimSpace(tokenInfo.ProfileArn); v != "" {
		creds["profile_arn"] = v
	}
	return creds
}

func (s *KiroOAuthService) BuildAccountExtra(tokenInfo *KiroTokenInfo) map[string]any {
	extra := map[string]any{
		"provider": firstNonEmpty(strings.ToLower(strings.TrimSpace(tokenInfo.Provider)), "kiro"),
		"source":   "kiro_browser_oauth",
	}
	if v := strings.TrimSpace(tokenInfo.Email); v != "" {
		extra["email"] = v
	}
	if v := strings.TrimSpace(tokenInfo.Username); v != "" {
		extra["username"] = v
	}
	if v := strings.TrimSpace(tokenInfo.DisplayName); v != "" {
		extra["display_name"] = v
	}
	return extra
}

func (s *KiroOAuthService) buildAuthURL(ctx context.Context, session *pkgkiro.OAuthSession, codeChallenge string) (string, error) {
	switch session.Method {
	case pkgkiro.OAuthMethodBuilder:
		fallthrough
	case pkgkiro.OAuthMethodIDC:
		registration, err := s.oauthClient.RegisterAuthCodeClient(ctx, session.RedirectURI, resolvedKiroIssuerURL(session.Method, session.StartURL), session.Region, session.ProxyURL)
		if err != nil {
			return "", err
		}
		session.ClientID = registration.ClientID
		session.ClientSecret = registration.ClientSecret
		return pkgkiro.BuildOIDCAuthURL(session.Region, registration.ClientID, session.RedirectURI, session.State, codeChallenge), nil
	default:
		return pkgkiro.BuildSocialAuthURL(session.Method, session.RedirectURI, codeChallenge, session.State)
	}
}

func (s *KiroOAuthService) maybeEnrichOIDCUserInfo(ctx context.Context, session *pkgkiro.OAuthSession, tokenInfo *KiroTokenInfo) {
	if tokenInfo == nil || session == nil {
		return
	}
	if session.Method != pkgkiro.OAuthMethodBuilder && session.Method != pkgkiro.OAuthMethodIDC {
		return
	}
	userInfo, err := s.oauthClient.FetchOIDCUserInfo(ctx, tokenInfo.AccessToken, session.Region, session.ProxyURL)
	if err != nil || userInfo == nil {
		return
	}
	tokenInfo.Email = firstNonEmpty(tokenInfo.Email, userInfo.Email)
	tokenInfo.Username = firstNonEmpty(tokenInfo.Username, userInfo.Username)
	tokenInfo.DisplayName = firstNonEmpty(tokenInfo.DisplayName, userInfo.DisplayName)
}

func (s *KiroOAuthService) resolveProxyURL(ctx context.Context, proxyID *int64) (string, error) {
	if proxyID == nil || s == nil || s.proxyRepo == nil {
		return "", nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
	if err != nil {
		return "", infraerrors.BadRequest("KIRO_OAUTH_PROXY_NOT_FOUND", "proxy not found").WithCause(err)
	}
	if proxy == nil {
		return "", nil
	}
	return strings.TrimSpace(proxy.URL()), nil
}

func normalizeRequestedKiroAuthMethod(method string) string {
	if normalized := normalizeStoredKiroAuthMethod(method); normalized != "" {
		return normalized
	}
	return pkgkiro.OAuthMethodBuilder
}

func normalizeStoredKiroAuthMethod(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case pkgkiro.OAuthMethodGitHub:
		return pkgkiro.OAuthMethodGitHub
	case pkgkiro.OAuthMethodGoogle:
		return pkgkiro.OAuthMethodGoogle
	case pkgkiro.OAuthMethodBuilder, "builder-id":
		return pkgkiro.OAuthMethodBuilder
	case pkgkiro.OAuthMethodIDC:
		return pkgkiro.OAuthMethodIDC
	default:
		return ""
	}
}

func resolvedKiroAuthMethod(method string, fallback string) string {
	if normalized := normalizeStoredKiroAuthMethod(method); normalized != "" {
		return normalized
	}
	return normalizeStoredKiroAuthMethod(fallback)
}

func inferKiroRefreshAuthMethod(account *Account) string {
	if account == nil {
		return ""
	}
	if normalized := resolvedKiroAuthMethod(account.GetCredential("auth_method"), ""); normalized != "" {
		return normalized
	}
	if strings.TrimSpace(account.GetCredential("client_id")) != "" && strings.TrimSpace(account.GetCredential("client_secret")) != "" {
		if strings.TrimSpace(account.GetCredential("start_url")) != "" {
			return pkgkiro.OAuthMethodIDC
		}
		return pkgkiro.OAuthMethodBuilder
	}
	switch strings.ToLower(strings.TrimSpace(account.GetExtraString("provider"))) {
	case "google":
		return pkgkiro.OAuthMethodGoogle
	case "aws":
		if strings.TrimSpace(account.GetCredential("start_url")) != "" {
			return pkgkiro.OAuthMethodIDC
		}
		return pkgkiro.OAuthMethodBuilder
	case "github":
		return pkgkiro.OAuthMethodGitHub
	default:
		return pkgkiro.OAuthMethodGitHub
	}
}

func resolvedKiroProvider(method string, current string) string {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current)
	}
	switch normalizeStoredKiroAuthMethod(method) {
	case pkgkiro.OAuthMethodGoogle:
		return "google"
	case pkgkiro.OAuthMethodBuilder, pkgkiro.OAuthMethodIDC:
		return "aws"
	default:
		return "github"
	}
}

func resolvedKiroRegion(method string, primary string, fallback string) string {
	isOIDC := normalizeStoredKiroAuthMethod(method) == pkgkiro.OAuthMethodBuilder || normalizeStoredKiroAuthMethod(method) == pkgkiro.OAuthMethodIDC
	if v := strings.TrimSpace(primary); v != "" && isOIDC {
		return v
	}
	if v := strings.TrimSpace(fallback); v != "" && isOIDC {
		return v
	}
	if isOIDC {
		return pkgkiro.DefaultIDCRegion
	}
	return ""
}

func resolvedKiroIssuerURL(method string, startURL string) string {
	if normalizeStoredKiroAuthMethod(method) == pkgkiro.OAuthMethodIDC {
		return firstNonEmpty(strings.TrimSpace(startURL), pkgkiro.BuilderIDStartURL)
	}
	return pkgkiro.BuilderIDStartURL
}

func normalizeKiroRegion(region string) string {
	trimmed := strings.TrimSpace(region)
	if trimmed == "" {
		return pkgkiro.DefaultIDCRegion
	}
	return trimmed
}

func normalizeKiroRequestedRegion(method string, region string) string {
	if normalizeStoredKiroAuthMethod(method) != pkgkiro.OAuthMethodBuilder && normalizeStoredKiroAuthMethod(method) != pkgkiro.OAuthMethodIDC {
		return strings.TrimSpace(region)
	}
	return normalizeKiroRegion(region)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
