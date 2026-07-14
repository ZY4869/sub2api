package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/imroc/req/v3"
)

func NewGrokOAuthClient() service.GrokOAuthClient {
	return &grokOAuthClient{}
}

type grokOAuthClient struct{}

func (c *grokOAuthClient) ExchangeCode(ctx context.Context, tokenURL string, code string, codeVerifier string, redirectURI string, clientID string, proxyURL string) (*grokoauth.TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", strings.TrimSpace(clientID))
	form.Set("code", strings.TrimSpace(code))
	form.Set("redirect_uri", strings.TrimSpace(redirectURI))
	form.Set("code_verifier", strings.TrimSpace(codeVerifier))
	return c.doToken(ctx, tokenURL, form, proxyURL, "GROK_OAUTH_TOKEN_EXCHANGE_FAILED")
}

func (c *grokOAuthClient) StartDeviceFlow(ctx context.Context, deviceURL string, clientID string, scope string, proxyURL string) (*grokoauth.DeviceAuthorizationResponse, error) {
	client, err := createGrokOAuthReqClient(proxyURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_CLIENT_INIT_FAILED", "failed to create Grok OAuth HTTP client").WithCause(err)
	}

	form := url.Values{}
	form.Set("client_id", strings.TrimSpace(clientID))
	if strings.TrimSpace(scope) != "" {
		form.Set("scope", strings.TrimSpace(scope))
	}

	var result grokoauth.DeviceAuthorizationResponse
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "sub2api-admin").
		SetFormDataFromValues(form).
		SetSuccessResult(&result).
		Post(strings.TrimSpace(deviceURL))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_DEVICE_START_FAILED", "Grok OAuth device request failed").WithCause(err)
	}
	if !resp.IsSuccessState() {
		message := summarizeGrokTokenError(resp.StatusCode, resp.String())
		return nil, infraerrors.Newf(http.StatusBadGateway, "GROK_OAUTH_DEVICE_START_FAILED", "%s", message)
	}
	return &result, nil
}

func (c *grokOAuthClient) PollDeviceToken(ctx context.Context, tokenURL string, deviceCode string, clientID string, proxyURL string) (*grokoauth.TokenResponse, error) {
	client, err := createGrokOAuthReqClient(proxyURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_CLIENT_INIT_FAILED", "failed to create Grok OAuth HTTP client").WithCause(err)
	}

	form := url.Values{}
	form.Set("grant_type", grokoauth.DeviceGrantType)
	form.Set("client_id", strings.TrimSpace(clientID))
	form.Set("device_code", strings.TrimSpace(deviceCode))

	var tokenResp grokoauth.TokenResponse
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "sub2api-admin").
		SetFormDataFromValues(form).
		SetSuccessResult(&tokenResp).
		Post(strings.TrimSpace(tokenURL))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_REQUEST_FAILED", "Grok OAuth device token request failed").WithCause(err)
	}
	if !resp.IsSuccessState() {
		status, description := parseGrokOAuthTokenError(resp.String())
		switch strings.ToLower(status) {
		case "authorization_pending", "slow_down", "access_denied", "expired_token":
			return nil, &grokoauth.DeviceTokenError{Status: strings.ToLower(status), Description: description}
		}
		message := summarizeGrokTokenError(resp.StatusCode, resp.String())
		return nil, infraerrors.Newf(http.StatusBadGateway, "GROK_OAUTH_DEVICE_TOKEN_FAILED", "%s", message)
	}
	return &tokenResp, nil
}

func (c *grokOAuthClient) RefreshToken(ctx context.Context, tokenURL string, refreshToken string, clientID string, scope string, proxyURL string) (*grokoauth.TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", strings.TrimSpace(clientID))
	form.Set("refresh_token", strings.TrimSpace(refreshToken))
	if strings.TrimSpace(scope) != "" {
		form.Set("scope", strings.TrimSpace(scope))
	}
	return c.doToken(ctx, tokenURL, form, proxyURL, "GROK_OAUTH_TOKEN_REFRESH_FAILED")
}

func (c *grokOAuthClient) FetchUserInfo(ctx context.Context, userInfoURL string, accessToken string, proxyURL string) (*grokoauth.UserInfo, error) {
	client, err := createGrokOAuthReqClient(proxyURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_CLIENT_INIT_FAILED", "failed to create Grok OAuth HTTP client").WithCause(err)
	}

	var userInfo grokoauth.UserInfo
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+strings.TrimSpace(accessToken)).
		SetSuccessResult(&userInfo).
		Get(strings.TrimSpace(userInfoURL))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_USERINFO_FAILED", "Grok OAuth userinfo request failed").WithCause(err)
	}
	if !resp.IsSuccessState() {
		return nil, infraerrors.Newf(http.StatusBadGateway, "GROK_OAUTH_USERINFO_FAILED", "Grok OAuth userinfo failed: status %d", resp.StatusCode)
	}
	return &userInfo, nil
}

func (c *grokOAuthClient) doToken(ctx context.Context, tokenURL string, form url.Values, proxyURL string, code string) (*grokoauth.TokenResponse, error) {
	client, err := createGrokOAuthReqClient(proxyURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_CLIENT_INIT_FAILED", "failed to create Grok OAuth HTTP client").WithCause(err)
	}

	var tokenResp grokoauth.TokenResponse
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "sub2api-admin").
		SetFormDataFromValues(form).
		SetSuccessResult(&tokenResp).
		Post(strings.TrimSpace(tokenURL))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_REQUEST_FAILED", "Grok OAuth token request failed").WithCause(err)
	}
	if !resp.IsSuccessState() {
		message := summarizeGrokTokenError(resp.StatusCode, resp.String())
		return nil, infraerrors.Newf(http.StatusBadGateway, code, "%s", message)
	}
	return &tokenResp, nil
}

func summarizeGrokTokenError(status int, body string) string {
	payload := grokTokenErrorPayload{}
	_ = json.Unmarshal([]byte(body), &payload)
	parts := []string{fmt.Sprintf("Grok OAuth token request failed: status %d", status)}
	errCode := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload.Error)))
	errDescription := strings.TrimSpace(firstNonEmptyRepoString(payload.ErrorDescription, payload.Message))
	switch {
	case status == http.StatusBadRequest && errCode == "authorization_pending":
		return "Grok OAuth device authorization is still pending"
	case status == http.StatusBadRequest && errCode == "slow_down":
		return "Grok OAuth device polling is too frequent"
	case status == http.StatusBadRequest && errCode == "access_denied":
		return "Grok OAuth device authorization was denied"
	case status == http.StatusBadRequest && errCode == "expired_token":
		return "Grok OAuth device code expired, generate a new device code in Sub2api and try again"
	case status == http.StatusBadRequest && errCode == "invalid_grant":
		hint := "authorization code is invalid, expired, already used, or does not match this Sub2api OAuth session; generate a new authorization link or device code in Sub2api and try again"
		if containsAnyFold(errDescription, "redirect") {
			hint = "redirect URI mismatch; generate a new authorization link in Sub2api and use the callback URL from that same session"
		} else if containsAnyFold(errDescription, "pkce", "code_verifier", "verifier") {
			hint = "PKCE verifier mismatch; generate a new authorization link in Sub2api and use the callback URL from that same session"
		}
		parts = append(parts, hint)
	case status == http.StatusBadRequest && errCode != "":
		parts = append(parts, truncateOAuthError(errCode, 80))
	}
	if errDescription != "" {
		parts = append(parts, truncateOAuthError(errDescription, 180))
	} else if payload.Error != nil && errCode != "" && status != http.StatusBadRequest {
		parts = append(parts, truncateOAuthError(fmt.Sprint(payload.Error), 180))
	}
	return strings.Join(parts, ", ")
}

type grokTokenErrorPayload struct {
	Error            any    `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"`
}

func parseGrokOAuthTokenError(body string) (string, string) {
	payload := grokTokenErrorPayload{}
	_ = json.Unmarshal([]byte(body), &payload)
	return strings.TrimSpace(fmt.Sprint(payload.Error)), strings.TrimSpace(firstNonEmptyRepoString(payload.ErrorDescription, payload.Message))
}

func firstNonEmptyRepoString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func containsAnyFold(value string, needles ...string) bool {
	lower := strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(lower, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func truncateOAuthError(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func createGrokOAuthReqClient(proxyURL string) (*req.Client, error) {
	return getSharedReqClient(reqClientOptions{
		ProxyURL: proxyURL,
		Timeout:  60 * time.Second,
	})
}
