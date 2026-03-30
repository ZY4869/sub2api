package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	httppool "github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/google/uuid"
)

const (
	copilotOAuthClientID      = "Iv1.b507a08c87ecfe98"
	copilotDefaultPollSeconds = 5
	copilotHTTPTimeout        = 30 * time.Second
)

var (
	copilotDeviceCodeURL       = "https://github.com/login/device/code"
	copilotOAuthTokenURL       = "https://github.com/login/oauth/access_token"
	copilotInternalTokenURL    = "https://api.github.com/copilot_internal/v2/token"
	copilotInternalUserInfoURL = "https://api.github.com/copilot_internal/user"
	copilotFallbackUserInfoURL = "https://api.github.com/user"
)

type CopilotDeviceFlowStartResult struct {
	SessionID               string `json:"session_id"`
	DeviceCode              string `json:"device_code,omitempty"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type CopilotDeviceFlowPollResult struct {
	SessionID string                 `json:"session_id"`
	Status    string                 `json:"status"`
	Interval  int                    `json:"interval"`
	ExpiresIn int                    `json:"expires_in,omitempty"`
	User      *CopilotGitHubUserInfo `json:"user,omitempty"`
}

type CopilotGitHubUserInfo struct {
	Login string `json:"login,omitempty"`
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

type CopilotAPITokenInfo struct {
	Token      string `json:"token"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
	APIBaseURL string `json:"api_base_url,omitempty"`
}

type CopilotAccountRefreshResult struct {
	Credentials map[string]any
	Extra       map[string]any
	TokenInfo   *CopilotAPITokenInfo
	UserInfo    *CopilotGitHubUserInfo
}

type copilotDeviceSession struct {
	ProxyID                 *int64
	ProxyURL                string
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresAt               time.Time
	IntervalSeconds         int
	GitHubAccessToken       string
	UserInfo                *CopilotGitHubUserInfo
}

type CopilotOAuthService struct {
	proxyRepo ProxyRepository
	sessions  sync.Map
}

func NewCopilotOAuthService(proxyRepo ProxyRepository) *CopilotOAuthService {
	return &CopilotOAuthService{proxyRepo: proxyRepo}
}

func (s *CopilotOAuthService) StartDeviceFlow(ctx context.Context, proxyID *int64) (*CopilotDeviceFlowStartResult, error) {
	proxyURL, err := s.resolveProxyURL(ctx, proxyID)
	if err != nil {
		return nil, infraerrors.BadRequest("COPILOT_PROXY_RESOLVE_FAILED", "failed to resolve proxy for Copilot device flow").WithCause(err)
	}

	payload := url.Values{}
	payload.Set("client_id", copilotOAuthClientID)
	payload.Set("scope", "read:user user:email")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, copilotDeviceCodeURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, infraerrors.InternalServer("COPILOT_DEVICE_FLOW_REQUEST_BUILD_FAILED", "failed to build Copilot device flow request").WithCause(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	respBody, statusCode, err := s.doCopilotHTTPRequest(req, proxyURL)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("COPILOT_DEVICE_FLOW_REQUEST_FAILED", "failed to request Copilot device code").WithCause(err)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_REQUEST_FAILED", fmt.Sprintf("copilot device flow start failed with status %d: %s", statusCode, truncateImportBody(respBody)))
	}

	var deviceCode struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(respBody, &deviceCode); err != nil {
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_INVALID_RESPONSE", "copilot device flow returned invalid JSON").WithCause(err)
	}
	if strings.TrimSpace(deviceCode.DeviceCode) == "" || strings.TrimSpace(deviceCode.UserCode) == "" || strings.TrimSpace(deviceCode.VerificationURI) == "" {
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_INVALID_RESPONSE", "copilot device flow response is missing required fields")
	}

	interval := deviceCode.Interval
	if interval <= 0 {
		interval = copilotDefaultPollSeconds
	}
	expiresIn := deviceCode.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = int((15 * time.Minute).Seconds())
	}
	sessionID := uuid.NewString()
	s.sessions.Store(sessionID, &copilotDeviceSession{
		ProxyID:                 proxyID,
		ProxyURL:                proxyURL,
		DeviceCode:              strings.TrimSpace(deviceCode.DeviceCode),
		UserCode:                strings.TrimSpace(deviceCode.UserCode),
		VerificationURI:         strings.TrimSpace(deviceCode.VerificationURI),
		VerificationURIComplete: strings.TrimSpace(deviceCode.VerificationURIComplete),
		ExpiresAt:               time.Now().Add(time.Duration(expiresIn) * time.Second),
		IntervalSeconds:         interval,
	})

	return &CopilotDeviceFlowStartResult{
		SessionID:               sessionID,
		DeviceCode:              strings.TrimSpace(deviceCode.DeviceCode),
		UserCode:                strings.TrimSpace(deviceCode.UserCode),
		VerificationURI:         strings.TrimSpace(deviceCode.VerificationURI),
		VerificationURIComplete: strings.TrimSpace(deviceCode.VerificationURIComplete),
		ExpiresIn:               expiresIn,
		Interval:                interval,
	}, nil
}

func (s *CopilotOAuthService) PollDeviceFlow(ctx context.Context, sessionID string) (*CopilotDeviceFlowPollResult, error) {
	session, err := s.getDeviceSession(sessionID)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		s.sessions.Delete(sessionID)
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_EXPIRED", "copilot device flow session has expired")
	}
	if strings.TrimSpace(session.GitHubAccessToken) != "" {
		return &CopilotDeviceFlowPollResult{
			SessionID: sessionID,
			Status:    "completed",
			Interval:  session.IntervalSeconds,
			User:      session.UserInfo,
		}, nil
	}

	token, pollStatus, err := s.exchangeDeviceCode(ctx, session.ProxyURL, session.DeviceCode)
	if err != nil {
		if pollStatus == "authorization_pending" || pollStatus == "slow_down" {
			if pollStatus == "slow_down" {
				session.IntervalSeconds += 5
			}
			return &CopilotDeviceFlowPollResult{
				SessionID: sessionID,
				Status:    "pending",
				Interval:  session.IntervalSeconds,
				ExpiresIn: remainingSeconds(session.ExpiresAt),
			}, nil
		}
		if pollStatus == "expired_token" {
			s.sessions.Delete(sessionID)
			return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_EXPIRED", "copilot device flow session has expired").WithCause(err)
		}
		if pollStatus == "access_denied" {
			s.sessions.Delete(sessionID)
			return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_ACCESS_DENIED", "copilot device flow authorization was denied").WithCause(err)
		}
		return nil, infraerrors.ServiceUnavailable("COPILOT_DEVICE_FLOW_POLL_FAILED", "failed to poll Copilot device flow").WithCause(err)
	}

	session.GitHubAccessToken = token
	userInfo, userErr := s.FetchGitHubUserInfo(ctx, token, session.ProxyURL)
	if userErr == nil {
		session.UserInfo = userInfo
	}

	return &CopilotDeviceFlowPollResult{
		SessionID: sessionID,
		Status:    "completed",
		Interval:  session.IntervalSeconds,
		User:      session.UserInfo,
	}, nil
}

func (s *CopilotOAuthService) GetAuthorizedDeviceSession(sessionID string) (string, *CopilotGitHubUserInfo, error) {
	session, err := s.getDeviceSession(sessionID)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(session.GitHubAccessToken) == "" {
		return "", nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_NOT_AUTHORIZED", "copilot device flow has not completed authorization yet")
	}
	return session.GitHubAccessToken, session.UserInfo, nil
}

func (s *CopilotOAuthService) ConsumeAuthorizedDeviceSession(sessionID string) (string, *CopilotGitHubUserInfo, error) {
	githubToken, userInfo, err := s.GetAuthorizedDeviceSession(sessionID)
	if err != nil {
		return "", nil, err
	}
	s.sessions.Delete(strings.TrimSpace(sessionID))
	return githubToken, userInfo, nil
}

func (s *CopilotOAuthService) DeleteDeviceSession(sessionID string) {
	if s == nil {
		return
	}
	s.sessions.Delete(strings.TrimSpace(sessionID))
}

func (s *CopilotOAuthService) BuildAccountCredentials(githubToken string) map[string]any {
	githubToken = strings.TrimSpace(githubToken)
	if githubToken == "" {
		return nil
	}
	return map[string]any{
		"access_token": githubToken,
	}
}

func (s *CopilotOAuthService) BuildAccountCredentialsWithTokenInfo(githubToken string, tokenInfo *CopilotAPITokenInfo) map[string]any {
	credentials := s.BuildAccountCredentials(githubToken)
	if credentials == nil {
		return nil
	}
	if tokenInfo != nil {
		if baseURL := trustedCopilotAPIBaseURL(strings.TrimSpace(tokenInfo.APIBaseURL)); baseURL != "" {
			credentials["base_url"] = baseURL
		}
	}
	return credentials
}

func (s *CopilotOAuthService) BuildAccountExtra(userInfo *CopilotGitHubUserInfo) map[string]any {
	if userInfo == nil {
		return nil
	}
	extra := map[string]any{}
	if login := strings.TrimSpace(userInfo.Login); login != "" {
		extra["username"] = login
	}
	if email := strings.TrimSpace(userInfo.Email); email != "" {
		extra["email"] = email
	}
	if name := strings.TrimSpace(userInfo.Name); name != "" {
		extra["display_name"] = name
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func (s *CopilotOAuthService) BuildAccountUpstreamExtra(tokenInfo *CopilotAPITokenInfo, probeSource string) map[string]any {
	info := s.buildResolvedUpstreamInfo(tokenInfo, probeSource)
	if info.URL == "" && info.Host == "" && info.Service == "" {
		return nil
	}
	return MergeUpstreamExtra(nil, info)
}

func (s *CopilotOAuthService) RefreshAccount(ctx context.Context, account *Account) (map[string]any, error) {
	result, err := s.RefreshAccountState(ctx, account)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.Extra, nil
}

func (s *CopilotOAuthService) RefreshAccountState(ctx context.Context, account *Account) (*CopilotAccountRefreshResult, error) {
	if !isCopilotOAuthAccount(account) {
		return nil, infraerrors.BadRequest("COPILOT_INVALID_ACCOUNT", "account is not a Copilot OAuth account")
	}
	tokenInfo, err := s.ExchangeGitHubTokenForCopilotToken(ctx, account)
	if err != nil {
		return nil, err
	}
	userInfo, err := s.FetchGitHubUserInfo(ctx, account.GetCredential("access_token"), s.resolveAccountProxyURL(ctx, account))
	if err != nil {
		userInfo = nil
	}
	credentials := MergeCredentials(account.Credentials, s.BuildAccountCredentialsWithTokenInfo(account.GetCredential("access_token"), tokenInfo))
	extra := mergeStringAnyMap(account.Extra, s.BuildAccountExtra(userInfo))
	extra = mergeStringAnyMap(extra, s.BuildAccountUpstreamExtra(tokenInfo, "copilot_refresh"))
	return &CopilotAccountRefreshResult{
		Credentials: credentials,
		Extra:       extra,
		TokenInfo:   tokenInfo,
		UserInfo:    userInfo,
	}, nil
}

func (s *CopilotOAuthService) ExchangeGitHubTokenForCopilotToken(ctx context.Context, account *Account) (*CopilotAPITokenInfo, error) {
	if !isCopilotOAuthAccount(account) {
		return nil, infraerrors.BadRequest("COPILOT_INVALID_ACCOUNT", "account is not a Copilot OAuth account")
	}
	githubToken := strings.TrimSpace(account.GetCredential("access_token"))
	if githubToken == "" {
		return nil, infraerrors.BadRequest("COPILOT_GITHUB_TOKEN_REQUIRED", "copilot account is missing GitHub access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, copilotInternalTokenURL, nil)
	if err != nil {
		return nil, infraerrors.InternalServer("COPILOT_TOKEN_REQUEST_BUILD_FAILED", "failed to build Copilot token exchange request").WithCause(err)
	}
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/json")
	applyCopilotDefaultHeaders(req.Header, account)

	respBody, statusCode, err := s.doCopilotHTTPRequest(req, s.resolveAccountProxyURL(ctx, account))
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("COPILOT_TOKEN_EXCHANGE_FAILED", "failed to exchange GitHub token for Copilot API token").WithCause(err)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, infraerrors.BadRequest("COPILOT_TOKEN_EXCHANGE_FAILED", fmt.Sprintf("copilot token exchange failed with status %d: %s", statusCode, truncateImportBody(respBody)))
	}

	var tokenResp struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
		Endpoints struct {
			API string `json:"api"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, infraerrors.BadRequest("COPILOT_TOKEN_EXCHANGE_INVALID_RESPONSE", "copilot token exchange returned invalid JSON").WithCause(err)
	}
	if strings.TrimSpace(tokenResp.Token) == "" {
		return nil, infraerrors.BadRequest("COPILOT_TOKEN_EXCHANGE_INVALID_RESPONSE", "copilot token exchange returned empty API token")
	}

	return &CopilotAPITokenInfo{
		Token:      strings.TrimSpace(tokenResp.Token),
		ExpiresAt:  tokenResp.ExpiresAt,
		APIBaseURL: trustedCopilotAPIBaseURL(strings.TrimSpace(tokenResp.Endpoints.API)),
	}, nil
}

func (s *CopilotOAuthService) buildResolvedUpstreamInfo(tokenInfo *CopilotAPITokenInfo, probeSource string) ResolvedUpstreamInfo {
	if tokenInfo == nil {
		return ResolvedUpstreamInfo{}
	}
	info := ResolveUpstreamInfo(tokenInfo.APIBaseURL, PlatformCopilot, probeSource)
	info.ProbedAt = time.Now().UTC()
	return info
}

func (s *CopilotOAuthService) FetchGitHubUserInfo(ctx context.Context, githubToken string, proxyURL string) (*CopilotGitHubUserInfo, error) {
	userInfo, err := s.fetchGitHubUserInfoFromURL(ctx, githubToken, proxyURL, copilotInternalUserInfoURL)
	if err == nil {
		return userInfo, nil
	}
	return s.fetchGitHubUserInfoFromURL(ctx, githubToken, proxyURL, copilotFallbackUserInfoURL)
}

func (s *CopilotOAuthService) resolveProxyURL(ctx context.Context, proxyID *int64) (string, error) {
	if proxyID == nil || s == nil || s.proxyRepo == nil {
		return "", nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
	if err != nil || proxy == nil {
		return "", err
	}
	return strings.TrimSpace(proxy.URL()), nil
}

func (s *CopilotOAuthService) resolveAccountProxyURL(ctx context.Context, account *Account) string {
	if account == nil {
		return ""
	}
	if account.Proxy != nil {
		return strings.TrimSpace(account.Proxy.URL())
	}
	proxyURL, _ := s.resolveProxyURL(ctx, account.ProxyID)
	return proxyURL
}

func (s *CopilotOAuthService) doCopilotHTTPRequest(req *http.Request, proxyURL string) ([]byte, int, error) {
	client, err := httppool.GetClient(httppool.Options{
		ProxyURL:              strings.TrimSpace(proxyURL),
		Timeout:               copilotHTTPTimeout,
		ResponseHeaderTimeout: 20 * time.Second,
	})
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	return body, resp.StatusCode, nil
}

func (s *CopilotOAuthService) getDeviceSession(sessionID string) (*copilotDeviceSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_SESSION_REQUIRED", "copilot device flow session_id is required")
	}
	raw, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, infraerrors.BadRequest("COPILOT_DEVICE_FLOW_SESSION_NOT_FOUND", "copilot device flow session was not found")
	}
	session, ok := raw.(*copilotDeviceSession)
	if !ok || session == nil {
		return nil, infraerrors.InternalServer("COPILOT_DEVICE_FLOW_SESSION_INVALID", "copilot device flow session is invalid")
	}
	return session, nil
}

func (s *CopilotOAuthService) exchangeDeviceCode(ctx context.Context, proxyURL string, deviceCode string) (string, string, error) {
	payload := url.Values{}
	payload.Set("client_id", copilotOAuthClientID)
	payload.Set("device_code", strings.TrimSpace(deviceCode))
	payload.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, copilotOAuthTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	body, _, err := s.doCopilotHTTPRequest(req, proxyURL)
	if err != nil {
		return "", "", err
	}

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(tokenResp.Error) != "" {
		return "", strings.TrimSpace(tokenResp.Error), errors.New(strings.TrimSpace(tokenResp.ErrorDescription))
	}
	return strings.TrimSpace(tokenResp.AccessToken), "", nil
}

func (s *CopilotOAuthService) fetchGitHubUserInfoFromURL(ctx context.Context, githubToken string, proxyURL string, endpoint string) (*CopilotGitHubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+strings.TrimSpace(githubToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", copilotGitHubAPIVersion)
	req.Header.Set("User-Agent", copilotDefaultUserAgent)

	respBody, statusCode, err := s.doCopilotHTTPRequest(req, proxyURL)
	if err != nil {
		return nil, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("status %d: %s", statusCode, truncateImportBody(respBody))
	}

	var userInfo CopilotGitHubUserInfo
	if err := json.Unmarshal(respBody, &userInfo); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userInfo.Login) == "" {
		return nil, errors.New("github user info response missing login")
	}
	return &userInfo, nil
}

func remainingSeconds(expiresAt time.Time) int {
	seconds := int(time.Until(expiresAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}
