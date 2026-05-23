package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
)

const (
	socialOAuthCookieMaxAgeSec   = 10 * 60
	socialOAuthDefaultRedirectTo = "/dashboard"
	socialOAuthDefaultBindTo     = "/profile"
	socialOAuthDefaultFrontendCB = "/auth/social/callback"
	socialOAuthBindMode          = "bind"
	socialOAuthLoginMode         = "login"
)

var (
	socialOAuthExchangeCodeFn  = exchangeSocialOAuthCode
	socialOAuthFetchUserInfoFn = fetchSocialUserInfo
)

type socialOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type socialOAuthUserInfo struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
	InternalOnly   *bool
}

type completeSocialOAuthRequest struct {
	PendingOAuthToken string `json:"pending_oauth_token" binding:"required"`
	InvitationCode    string `json:"invitation_code" binding:"required"`
	AffCode           string `json:"aff_code"`
}

func (h *AuthHandler) SetAuthIdentityService(identityService *service.AuthIdentityService) {
	h.identities = identityService
}

func (h *AuthHandler) SetUserAttributeService(userAttributeService *service.UserAttributeService) {
	h.userAttrs = userAttributeService
}

// SocialOAuthStart starts a GitHub/Google OAuth flow.
// GET /api/v1/auth/oauth/:provider/start
func (h *AuthHandler) SocialOAuthStart(c *gin.Context) {
	provider := service.NormalizeOAuthProvider(c.Param("provider"))
	if provider == "" {
		response.ErrorFrom(c, service.ErrOAuthProviderUnsupported)
		return
	}
	cfg, err := h.getSocialOAuthConfig(c.Request.Context(), provider)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	mode := normalizeSocialOAuthMode(c.Query("mode"))
	redirectTo := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirectTo == "" {
		if mode == socialOAuthBindMode {
			redirectTo = socialOAuthDefaultBindTo
		} else {
			redirectTo = socialOAuthDefaultRedirectTo
		}
	}
	affCode := sanitizeAffiliateCode(c.Query("aff_code"))

	bindUserID := int64(0)
	if mode == socialOAuthBindMode {
		bindUser, bindErr := h.resolveOAuthBindUser(c)
		if bindErr != nil {
			response.ErrorFrom(c, bindErr)
			return
		}
		bindUserID = bindUser.ID
	}

	state, err := oauth.GenerateState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	codeChallenge := ""
	codeVerifier := ""
	if cfg.UsePKCE {
		codeVerifier, err = oauth.GenerateCodeVerifier()
		if err != nil {
			response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_PKCE_GEN_FAILED", "failed to generate oauth verifier").WithCause(err))
			return
		}
		codeChallenge = oauth.GenerateCodeChallenge(codeVerifier)
	}

	secureCookie := isRequestHTTPS(c)
	cookiePath := socialOAuthCookiePath(provider)
	setOAuthCookie(c, cookiePath, socialOAuthStateCookieName(provider), encodeCookieValue(state), socialOAuthCookieMaxAgeSec, secureCookie)
	setOAuthCookie(c, cookiePath, socialOAuthRedirectCookieName(provider), encodeCookieValue(redirectTo), socialOAuthCookieMaxAgeSec, secureCookie)
	setOAuthCookie(c, cookiePath, socialOAuthModeCookieName(provider), encodeCookieValue(mode), socialOAuthCookieMaxAgeSec, secureCookie)
	if affCode != "" {
		setOAuthCookie(c, cookiePath, socialOAuthAffCodeCookieName(provider), encodeCookieValue(affCode), socialOAuthCookieMaxAgeSec, secureCookie)
	}
	if bindUserID > 0 {
		setOAuthCookie(c, cookiePath, socialOAuthBindUserCookieName(provider), encodeCookieValue(strconv.FormatInt(bindUserID, 10)), socialOAuthCookieMaxAgeSec, secureCookie)
	}
	if codeVerifier != "" {
		setOAuthCookie(c, cookiePath, socialOAuthVerifierCookieName(provider), encodeCookieValue(codeVerifier), socialOAuthCookieMaxAgeSec, secureCookie)
	}

	authURL, err := buildSocialAuthorizeURL(cfg, state, codeChallenge)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BUILD_URL_FAILED", "failed to build oauth authorization url").WithCause(err))
		return
	}

	slog.Info("social_oauth_start",
		"provider", provider,
		"mode", mode,
		"redirect", redirectTo,
		"bind_user_id", bindUserID,
	)
	c.Redirect(http.StatusFound, authURL)
}

// SocialOAuthCallback completes a GitHub/Google OAuth callback and redirects back to frontend.
// GET /api/v1/auth/oauth/:provider/callback
func (h *AuthHandler) SocialOAuthCallback(c *gin.Context) {
	provider := service.NormalizeOAuthProvider(c.Param("provider"))
	if provider == "" {
		response.ErrorFrom(c, service.ErrOAuthProviderUnsupported)
		return
	}
	cfg, err := h.getSocialOAuthConfig(c.Request.Context(), provider)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	frontendCallback := strings.TrimSpace(cfg.FrontendRedirectURL)
	if frontendCallback == "" {
		frontendCallback = socialOAuthDefaultFrontendCB
	}
	if providerErr := strings.TrimSpace(c.Query("error")); providerErr != "" {
		redirectOAuthError(c, frontendCallback, "provider_error", providerErr, c.Query("error_description"))
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		redirectOAuthError(c, frontendCallback, "missing_params", "missing code/state", "")
		return
	}

	secureCookie := isRequestHTTPS(c)
	defer clearSocialOAuthCookies(c, provider, secureCookie)

	expectedState, err := readOAuthCookieDecoded(c, socialOAuthStateCookieName(provider))
	if err != nil || expectedState == "" || expectedState != state {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}

	redirectTo, _ := readOAuthCookieDecoded(c, socialOAuthRedirectCookieName(provider))
	redirectTo = sanitizeFrontendRedirectPath(redirectTo)
	if redirectTo == "" {
		redirectTo = socialOAuthDefaultRedirectTo
	}
	mode, _ := readOAuthCookieDecoded(c, socialOAuthModeCookieName(provider))
	mode = normalizeSocialOAuthMode(mode)
	affCode, _ := readOAuthCookieDecoded(c, socialOAuthAffCodeCookieName(provider))
	affCode = sanitizeAffiliateCode(affCode)

	bindUserID := int64(0)
	if mode == socialOAuthBindMode {
		if raw, readErr := readOAuthCookieDecoded(c, socialOAuthBindUserCookieName(provider)); readErr == nil {
			if parsed, parseErr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); parseErr == nil && parsed > 0 {
				bindUserID = parsed
			}
		}
		if bindUserID <= 0 {
			redirectOAuthError(c, frontendCallback, "bind_user_required", "bind mode requires authenticated user", "")
			return
		}
	}

	codeVerifier := ""
	if cfg.UsePKCE {
		codeVerifier, _ = readOAuthCookieDecoded(c, socialOAuthVerifierCookieName(provider))
		if codeVerifier == "" {
			redirectOAuthError(c, frontendCallback, "missing_verifier", "missing pkce verifier", "")
			return
		}
	}

	tokenResp, err := socialOAuthExchangeCodeFn(c.Request.Context(), cfg, code, codeVerifier)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "token_exchange_failed", "failed to exchange oauth code", singleLine(err.Error()))
		return
	}
	userInfo, err := socialOAuthFetchUserInfoFn(c.Request.Context(), cfg, tokenResp)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "userinfo_failed", "failed to fetch user info", singleLine(err.Error()))
		return
	}

	if h.identities == nil {
		redirectOAuthError(c, frontendCallback, "service_unavailable", "oauth identities service unavailable", "")
		return
	}

	identity := &service.AuthIdentity{
		Provider:       provider,
		ProviderUserID: userInfo.ProviderUserID,
		Email:          userInfo.Email,
		EmailVerified:  userInfo.EmailVerified,
		DisplayName:    userInfo.DisplayName,
		AvatarURL:      userInfo.AvatarURL,
	}
	result, resolveErr := h.identities.ResolveLoginOrBind(c.Request.Context(), mode, bindUserID, identity, "", affCode)
	if resolveErr != nil {
		redirectOAuthError(c, frontendCallback, "login_failed", infraerrors.Reason(resolveErr), infraerrors.Message(resolveErr))
		return
	}
	h.syncDingTalkInternalOnly(c.Request.Context(), userInfo, result)

	slog.Info("social_oauth_callback",
		"provider", provider,
		"mode", mode,
		"bind_user_id", bindUserID,
		"outcome", result.Outcome,
		"email_verified", userInfo.EmailVerified,
		"has_email", strings.TrimSpace(userInfo.Email) != "",
	)

	fragment := url.Values{}
	switch result.Outcome {
	case "pending":
		fragment.Set("error", "invitation_required")
		fragment.Set("pending_oauth_token", result.Pending)
		fragment.Set("provider", provider)
		fragment.Set("mode", mode)
		fragment.Set("redirect", redirectTo)
	case "bind_success":
		fragment.Set("result", "bind_success")
		fragment.Set("provider", provider)
		fragment.Set("mode", mode)
		fragment.Set("redirect", redirectTo)
	default:
		if result.TokenPair == nil {
			redirectOAuthError(c, frontendCallback, "login_failed", "missing token pair", "")
			return
		}
		fragment.Set("access_token", result.TokenPair.AccessToken)
		fragment.Set("refresh_token", result.TokenPair.RefreshToken)
		fragment.Set("expires_in", strconv.FormatInt(int64(result.TokenPair.ExpiresIn), 10))
		fragment.Set("token_type", "Bearer")
		fragment.Set("provider", provider)
		fragment.Set("mode", mode)
		fragment.Set("redirect", redirectTo)
	}
	redirectWithFragment(c, frontendCallback, fragment)
}

// CompleteSocialOAuthRegistration completes a pending GitHub/Google registration.
// POST /api/v1/auth/oauth/:provider/complete
func (h *AuthHandler) CompleteSocialOAuthRegistration(c *gin.Context) {
	provider := service.NormalizeOAuthProvider(c.Param("provider"))
	if provider == "" {
		response.ErrorFrom(c, service.ErrOAuthProviderUnsupported)
		return
	}
	if h.identities == nil {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return
	}

	var req completeSocialOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	claims, err := h.authService.VerifyPendingOAuthClaims(req.PendingOAuthToken)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if service.NormalizeOAuthProvider(claims.Provider) != provider {
		response.ErrorFrom(c, infraerrors.BadRequest("OAUTH_PROVIDER_MISMATCH", "oauth provider mismatch"))
		return
	}
	affCode := claims.AffCode
	if candidate := sanitizeAffiliateCode(req.AffCode); candidate != "" {
		affCode = candidate
	}

	tokenPair, user, err := h.authService.LoginOrRegisterOAuthWithTokenPair(
		c.Request.Context(),
		strings.TrimSpace(claims.Email),
		strings.TrimSpace(claims.Username),
		req.InvitationCode,
		affCode,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if user == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_USER_MISSING", "oauth user missing"))
		return
	}
	identity := &service.AuthIdentity{
		Provider:       provider,
		ProviderUserID: strings.TrimSpace(claims.ProviderUserID),
		Email:          strings.TrimSpace(claims.Email),
		EmailVerified:  claims.EmailVerified,
		DisplayName:    strings.TrimSpace(claims.DisplayName),
		AvatarURL:      strings.TrimSpace(claims.AvatarURL),
	}
	if _, bindErr := h.identities.BindIdentity(c.Request.Context(), user.ID, identity); bindErr != nil {
		response.ErrorFrom(c, bindErr)
		return
	}
	response.Success(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) getSocialOAuthConfig(ctx context.Context, provider string) (service.SocialOAuthConfig, error) {
	if h != nil && h.settingSvc != nil {
		return h.settingSvc.GetSocialOAuthConfig(ctx, provider)
	}
	return service.SocialOAuthConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "oauth config not loaded")
}

func (h *AuthHandler) resolveOAuthBindUser(c *gin.Context) (*service.User, error) {
	if h == nil || h.authService == nil || h.userService == nil {
		return nil, service.ErrServiceUnavailable
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return nil, service.ErrAuthIdentityBindRequired
	}
	tokenString, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		tokenString, ok = strings.CutPrefix(authHeader, "bearer ")
	}
	tokenString = strings.TrimSpace(tokenString)
	if !ok || tokenString == "" {
		return nil, service.ErrAuthIdentityBindRequired
	}
	claims, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims == nil || claims.UserID <= 0 {
		return nil, service.ErrAuthIdentityBindRequired
	}
	return h.userService.GetByID(c.Request.Context(), claims.UserID)
}

func (h *AuthHandler) syncDingTalkInternalOnly(ctx context.Context, userInfo *socialOAuthUserInfo, result *service.SocialIdentityResult) {
	if userInfo == nil || userInfo.InternalOnly == nil || result == nil || result.User == nil {
		return
	}
	h.syncInternalOnlyAttribute(ctx, result.User.ID, userInfo.InternalOnly)
}

func (h *AuthHandler) syncInternalOnlyAttribute(ctx context.Context, userID int64, value *bool) {
	if h == nil || h.userAttrs == nil || userID <= 0 || value == nil {
		return
	}
	raw := "false"
	if *value {
		raw = "true"
	}
	if err := h.userAttrs.SetUserAttributeByKey(ctx, userID, "internal_only", raw); err != nil {
		slog.Warn("social_oauth_internal_only_sync_failed", "user_id", userID, "error", err)
	}
}

func normalizeSocialOAuthMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case socialOAuthBindMode:
		return socialOAuthBindMode
	default:
		return socialOAuthLoginMode
	}
}

func socialOAuthCookiePath(provider string) string {
	provider = service.NormalizeOAuthProvider(provider)
	if provider == "" {
		provider = "unknown"
	}
	return "/api/v1/auth/oauth/" + provider
}

func socialOAuthStateCookieName(provider string) string {
	return provider + "_oauth_state"
}

func socialOAuthVerifierCookieName(provider string) string {
	return provider + "_oauth_verifier"
}

func socialOAuthRedirectCookieName(provider string) string {
	return provider + "_oauth_redirect"
}

func socialOAuthAffCodeCookieName(provider string) string {
	return provider + "_oauth_aff_code"
}

func socialOAuthModeCookieName(provider string) string {
	return provider + "_oauth_mode"
}

func socialOAuthBindUserCookieName(provider string) string {
	return provider + "_oauth_bind_user_id"
}

func clearSocialOAuthCookies(c *gin.Context, provider string, secure bool) {
	cookiePath := socialOAuthCookiePath(provider)
	clearOAuthCookie(c, cookiePath, socialOAuthStateCookieName(provider), secure)
	clearOAuthCookie(c, cookiePath, socialOAuthVerifierCookieName(provider), secure)
	clearOAuthCookie(c, cookiePath, socialOAuthRedirectCookieName(provider), secure)
	clearOAuthCookie(c, cookiePath, socialOAuthAffCodeCookieName(provider), secure)
	clearOAuthCookie(c, cookiePath, socialOAuthModeCookieName(provider), secure)
	clearOAuthCookie(c, cookiePath, socialOAuthBindUserCookieName(provider), secure)
}

func setOAuthCookie(c *gin.Context, path string, name string, value string, maxAgeSec int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthCookie(c *gin.Context, path string, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthCookieDecoded(c *gin.Context, name string) (string, error) {
	ck, err := c.Request.Cookie(name)
	if err != nil {
		return "", err
	}
	return decodeCookieValue(ck.Value)
}

func buildSocialAuthorizeURL(cfg service.SocialOAuthConfig, state string, codeChallenge string) (string, error) {
	u, err := url.Parse(cfg.AuthorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize url: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURL)
	if strings.TrimSpace(cfg.Scopes) != "" {
		q.Set("scope", cfg.Scopes)
	}
	q.Set("state", state)
	if cfg.UsePKCE {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}
	if cfg.Provider == service.AuthProviderGoogle {
		q.Set("access_type", "offline")
		q.Set("include_granted_scopes", "true")
		q.Set("prompt", "select_account")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func exchangeSocialOAuthCode(ctx context.Context, cfg service.SocialOAuthConfig, code string, codeVerifier string) (*socialOAuthTokenResponse, error) {
	if cfg.Provider == service.AuthProviderDingTalk {
		return exchangeDingTalkOAuthCode(ctx, cfg, code)
	}

	client := req.C().SetTimeout(30 * time.Second)
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURL)
	if cfg.UsePKCE && codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	if strings.EqualFold(strings.TrimSpace(cfg.TokenAuthMethod), "client_secret_basic") {
	} else {
		form.Set("client_secret", cfg.ClientSecret)
	}

	r := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json")
	if strings.EqualFold(strings.TrimSpace(cfg.TokenAuthMethod), "client_secret_basic") {
		r.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	}
	resp, err := r.SetFormDataFromValues(form).Post(cfg.TokenURL)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(resp.String())
	if !resp.IsSuccessState() {
		providerErr, providerDesc := parseOAuthProviderError(body)
		return nil, fmt.Errorf("token exchange failed: status=%d error=%s description=%s", resp.StatusCode, providerErr, providerDesc)
	}

	var parsed socialOAuthTokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		values, parseErr := url.ParseQuery(body)
		if parseErr != nil {
			return nil, fmt.Errorf("parse token response: %w", err)
		}
		parsed.AccessToken = strings.TrimSpace(values.Get("access_token"))
		parsed.TokenType = strings.TrimSpace(values.Get("token_type"))
		parsed.RefreshToken = strings.TrimSpace(values.Get("refresh_token"))
		parsed.IDToken = strings.TrimSpace(values.Get("id_token"))
		parsed.Scope = strings.TrimSpace(values.Get("scope"))
		if raw := strings.TrimSpace(values.Get("expires_in")); raw != "" {
			if expiresIn, convErr := strconv.ParseInt(raw, 10, 64); convErr == nil {
				parsed.ExpiresIn = expiresIn
			}
		}
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return nil, fmt.Errorf("oauth access token missing")
	}
	return &parsed, nil
}

func exchangeDingTalkOAuthCode(ctx context.Context, cfg service.SocialOAuthConfig, code string) (*socialOAuthTokenResponse, error) {
	client := req.C().SetTimeout(30 * time.Second)
	payload := map[string]string{
		"clientId":     cfg.ClientID,
		"clientSecret": cfg.ClientSecret,
		"code":         strings.TrimSpace(code),
		"grantType":    "authorization_code",
	}
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(cfg.TokenURL)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(resp.String())
	if !resp.IsSuccessState() {
		providerErr, providerDesc := parseOAuthProviderError(body)
		return nil, fmt.Errorf("dingtalk token exchange failed: status=%d error=%s description=%s", resp.StatusCode, providerErr, providerDesc)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, fmt.Errorf("parse dingtalk token response: %w", err)
	}
	parsed := &socialOAuthTokenResponse{
		AccessToken:  strings.TrimSpace(firstNonEmpty(socialOAuthAnyString(raw["accessToken"]), socialOAuthAnyString(raw["access_token"]))),
		TokenType:    strings.TrimSpace(firstNonEmpty(socialOAuthAnyString(raw["tokenType"]), socialOAuthAnyString(raw["token_type"]), "Bearer")),
		RefreshToken: strings.TrimSpace(firstNonEmpty(socialOAuthAnyString(raw["refreshToken"]), socialOAuthAnyString(raw["refresh_token"]))),
		Scope:        strings.TrimSpace(socialOAuthAnyString(raw["scope"])),
	}
	parsed.ExpiresIn = firstPositiveInt64(raw["expireIn"], raw["expiresIn"], raw["expires_in"])
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("dingtalk access token missing")
	}
	return parsed, nil
}

func fetchSocialUserInfo(ctx context.Context, cfg service.SocialOAuthConfig, tokenResp *socialOAuthTokenResponse) (*socialOAuthUserInfo, error) {
	if tokenResp == nil {
		return nil, errors.New("oauth token response missing")
	}
	switch cfg.Provider {
	case service.AuthProviderGitHub:
		return fetchGitHubUserInfo(ctx, cfg, tokenResp.AccessToken)
	case service.AuthProviderGoogle:
		return fetchGenericSocialUserInfo(ctx, cfg, tokenResp.AccessToken)
	case service.AuthProviderDingTalk:
		return fetchDingTalkUserInfo(ctx, cfg, tokenResp.AccessToken)
	default:
		return nil, service.ErrOAuthProviderUnsupported
	}
}

func fetchGenericSocialUserInfo(ctx context.Context, cfg service.SocialOAuthConfig, accessToken string) (*socialOAuthUserInfo, error) {
	client := req.C().SetTimeout(30 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetBearerAuthToken(accessToken).
		SetHeader("Accept", "application/json").
		Get(cfg.UserInfoURL)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("userinfo status=%d", resp.StatusCode)
	}
	body := resp.String()
	return &socialOAuthUserInfo{
		ProviderUserID: strings.TrimSpace(getGJSON(body, cfg.UserInfoIDPath)),
		Email:          strings.TrimSpace(getGJSON(body, cfg.UserInfoEmailPath)),
		EmailVerified:  parseTruthyValue(getGJSON(body, cfg.UserInfoEmailVerifiedPath)),
		DisplayName: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, cfg.UserInfoUsernamePath),
			getGJSON(body, "preferred_username"),
			getGJSON(body, "name"),
		)),
		AvatarURL: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, cfg.UserInfoAvatarPath),
			getGJSON(body, "picture"),
		)),
	}, nil
}

func fetchGitHubUserInfo(ctx context.Context, cfg service.SocialOAuthConfig, accessToken string) (*socialOAuthUserInfo, error) {
	client := req.C().SetTimeout(30 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetBearerAuthToken(accessToken).
		SetHeader("Accept", "application/json").
		SetHeader("X-GitHub-Api-Version", "2022-11-28").
		Get(cfg.UserInfoURL)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("github userinfo status=%d", resp.StatusCode)
	}
	body := resp.String()
	info := &socialOAuthUserInfo{
		ProviderUserID: strings.TrimSpace(getGJSON(body, cfg.UserInfoIDPath)),
		Email:          strings.TrimSpace(getGJSON(body, cfg.UserInfoEmailPath)),
		EmailVerified:  parseTruthyValue(getGJSON(body, cfg.UserInfoEmailVerifiedPath)),
		DisplayName: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, "name"),
			getGJSON(body, cfg.UserInfoUsernamePath),
		)),
		AvatarURL: strings.TrimSpace(getGJSON(body, cfg.UserInfoAvatarPath)),
	}
	if info.Email != "" && info.EmailVerified {
		return info, nil
	}

	emailResp, err := client.R().
		SetContext(ctx).
		SetBearerAuthToken(accessToken).
		SetHeader("Accept", "application/json").
		SetHeader("X-GitHub-Api-Version", "2022-11-28").
		Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, err
	}
	if !emailResp.IsSuccessState() {
		return info, nil
	}

	var emailItems []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(emailResp.Bytes(), &emailItems); err != nil {
		return nil, err
	}
	for _, item := range emailItems {
		if strings.TrimSpace(item.Email) == "" || !item.Verified || !item.Primary {
			continue
		}
		info.Email = strings.TrimSpace(item.Email)
		info.EmailVerified = true
		return info, nil
	}
	for _, item := range emailItems {
		if strings.TrimSpace(item.Email) == "" || !item.Verified {
			continue
		}
		info.Email = strings.TrimSpace(item.Email)
		info.EmailVerified = true
		return info, nil
	}
	return info, nil
}

func fetchDingTalkUserInfo(ctx context.Context, cfg service.SocialOAuthConfig, accessToken string) (*socialOAuthUserInfo, error) {
	client := req.C().SetTimeout(30 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("x-acs-dingtalk-access-token", accessToken).
		Get(cfg.UserInfoURL)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("dingtalk userinfo status=%d", resp.StatusCode)
	}
	body := resp.String()
	info := &socialOAuthUserInfo{
		ProviderUserID: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, cfg.UserInfoIDPath),
			getGJSON(body, "union_id"),
			getGJSON(body, "openId"),
			getGJSON(body, "openid"),
			getGJSON(body, "open_id"),
		)),
		Email: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, cfg.UserInfoEmailPath),
			getGJSON(body, "email"),
		)),
		DisplayName: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, cfg.UserInfoUsernamePath),
			getGJSON(body, "nick"),
			getGJSON(body, "name"),
			getGJSON(body, "displayName"),
		)),
		AvatarURL: strings.TrimSpace(firstNonEmpty(
			getGJSON(body, cfg.UserInfoAvatarPath),
			getGJSON(body, "avatarUrl"),
			getGJSON(body, "avatar"),
		)),
	}
	info.EmailVerified = info.Email != ""
	if value, ok := parseOptionalTruthyValue(firstNonEmpty(
		getGJSON(body, "internal_only"),
		getGJSON(body, "internalOnly"),
		getGJSON(body, "internal"),
		getGJSON(body, "staff"),
	)); ok {
		info.InternalOnly = &value
	}
	return info, nil
}

func parseTruthyValue(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func parseOptionalTruthyValue(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true, true
	case "0", "false", "no":
		return false, true
	default:
		return false, false
	}
}

func socialOAuthAnyString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

func firstPositiveInt64(values ...any) int64 {
	for _, value := range values {
		switch v := value.(type) {
		case int64:
			if v > 0 {
				return v
			}
		case int:
			if v > 0 {
				return int64(v)
			}
		case float64:
			if v > 0 {
				return int64(v)
			}
		case json.Number:
			if parsed, err := v.Int64(); err == nil && parsed > 0 {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}
