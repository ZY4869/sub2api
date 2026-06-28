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
	type errorPayload struct {
		Error            any    `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	payload := errorPayload{}
	_ = json.Unmarshal([]byte(body), &payload)
	parts := []string{fmt.Sprintf("Grok OAuth token request failed: status %d", status)}
	if value := strings.TrimSpace(payload.ErrorDescription); value != "" {
		parts = append(parts, truncateOAuthError(value, 180))
	} else if value := strings.TrimSpace(payload.Message); value != "" {
		parts = append(parts, truncateOAuthError(value, 180))
	} else if payload.Error != nil {
		parts = append(parts, truncateOAuthError(fmt.Sprint(payload.Error), 180))
	}
	return strings.Join(parts, ", ")
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
