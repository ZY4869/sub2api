package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	pkgkiro "github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

const kiroHTTPTimeout = 45 * time.Second

func NewKiroOAuthClient() service.KiroOAuthClient {
	return &kiroOAuthClient{}
}

type kiroOAuthClient struct{}

func (c *kiroOAuthClient) RegisterAuthCodeClient(ctx context.Context, redirectURI, issuerURL, region, proxyURL string) (*service.KiroClientRegistration, error) {
	payload := map[string]any{
		"clientName":   "Kiro IDE",
		"clientType":   "public",
		"scopes":       strings.Split(pkgkiro.OIDCScopes, ","),
		"grantTypes":   []string{"authorization_code", "refresh_token"},
		"redirectUris": []string{redirectURI},
		"issuerUrl":    issuerURL,
	}
	var resp struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := c.doJSON(ctx, http.MethodPost, pkgkiro.OIDCEndpoint(region)+"/client/register", payload, proxyURL, setKiroOIDCHeaders, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.ClientID) == "" || strings.TrimSpace(resp.ClientSecret) == "" {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_CLIENT_REGISTER_INVALID", "kiro oidc client registration response is missing client credentials")
	}
	return &service.KiroClientRegistration{
		ClientID:     strings.TrimSpace(resp.ClientID),
		ClientSecret: strings.TrimSpace(resp.ClientSecret),
	}, nil
}

func (c *kiroOAuthClient) ExchangeSocialCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL string) (*service.KiroTokenInfo, error) {
	return c.exchangeSocialToken(ctx, map[string]string{
		"code":          code,
		"code_verifier": codeVerifier,
		"redirect_uri":  redirectURI,
	}, proxyURL)
}

func (c *kiroOAuthClient) RefreshSocialToken(ctx context.Context, refreshToken, proxyURL string) (*service.KiroTokenInfo, error) {
	return c.exchangeSocialToken(ctx, map[string]string{
		"refreshToken": refreshToken,
	}, proxyURL, "/refreshToken")
}

func (c *kiroOAuthClient) ExchangeOIDCCode(ctx context.Context, clientID, clientSecret, code, codeVerifier, redirectURI, region, proxyURL string) (*service.KiroTokenInfo, error) {
	payload := map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"code":         code,
		"codeVerifier": codeVerifier,
		"redirectUri":  redirectURI,
		"grantType":    "authorization_code",
	}
	return c.exchangeOIDCToken(ctx, payload, region, proxyURL)
}

func (c *kiroOAuthClient) RefreshOIDCToken(ctx context.Context, clientID, clientSecret, refreshToken, region, startURL, proxyURL string) (*service.KiroTokenInfo, error) {
	payload := map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"refreshToken": refreshToken,
		"grantType":    "refresh_token",
	}
	tokenInfo, err := c.exchangeOIDCToken(ctx, payload, region, proxyURL)
	if err != nil {
		return nil, err
	}
	tokenInfo.StartURL = strings.TrimSpace(startURL)
	return tokenInfo, nil
}

func (c *kiroOAuthClient) FetchOIDCUserInfo(ctx context.Context, accessToken, region, proxyURL string) (*service.KiroTokenInfo, error) {
	req, err := c.newRequest(ctx, http.MethodGet, pkgkiro.OIDCEndpoint(region)+"/userinfo", nil, proxyURL)
	if err != nil {
		return nil, infraerrors.InternalServer("KIRO_OAUTH_USERINFO_BUILD_FAILED", "failed to build kiro oidc userinfo request").WithCause(err)
	}
	setKiroOIDCHeaders(req)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	body, statusCode, err := c.do(req, proxyURL)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("KIRO_OAUTH_USERINFO_FAILED", "failed to request kiro oidc userinfo").WithCause(err)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_USERINFO_FAILED", fmt.Sprintf("kiro oidc userinfo request failed with status %d", statusCode))
	}

	var resp struct {
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Username          string `json:"username"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, infraerrors.BadRequest("KIRO_OAUTH_USERINFO_INVALID", "kiro oidc userinfo response is invalid").WithCause(err)
	}
	return &service.KiroTokenInfo{
		Email:       strings.TrimSpace(resp.Email),
		Username:    firstNonEmptyString(resp.PreferredUsername, resp.Username),
		DisplayName: strings.TrimSpace(resp.Name),
	}, nil
}

func (c *kiroOAuthClient) exchangeSocialToken(ctx context.Context, payload map[string]string, proxyURL string, path ...string) (*service.KiroTokenInfo, error) {
	tokenPath := "/oauth/token"
	if len(path) > 0 && strings.TrimSpace(path[0]) != "" {
		tokenPath = path[0]
	}
	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ProfileArn   string `json:"profileArn"`
		ExpiresIn    int64  `json:"expiresIn"`
	}
	if err := c.doJSON(ctx, http.MethodPost, pkgkiro.SocialAuthEndpoint+tokenPath, payload, proxyURL, setKiroSocialHeaders, &resp); err != nil {
		return nil, err
	}
	return buildKiroTokenInfo(resp.AccessToken, resp.RefreshToken, resp.ProfileArn, resp.ExpiresIn), nil
}

func (c *kiroOAuthClient) exchangeOIDCToken(ctx context.Context, payload map[string]string, region, proxyURL string) (*service.KiroTokenInfo, error) {
	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int64  `json:"expiresIn"`
	}
	if err := c.doJSON(ctx, http.MethodPost, pkgkiro.OIDCEndpoint(region)+"/token", payload, proxyURL, setKiroOIDCHeaders, &resp); err != nil {
		return nil, err
	}
	tokenInfo := buildKiroTokenInfo(resp.AccessToken, resp.RefreshToken, "", resp.ExpiresIn)
	tokenInfo.Region = strings.TrimSpace(region)
	return tokenInfo, nil
}

func (c *kiroOAuthClient) doJSON(ctx context.Context, method, endpoint string, payload any, proxyURL string, applyHeaders func(*http.Request), out any) error {
	req, err := c.newRequest(ctx, method, endpoint, payload, proxyURL)
	if err != nil {
		return infraerrors.InternalServer("KIRO_OAUTH_REQUEST_BUILD_FAILED", "failed to build kiro oauth request").WithCause(err)
	}
	if applyHeaders != nil {
		applyHeaders(req)
	}
	body, statusCode, err := c.do(req, proxyURL)
	if err != nil {
		return infraerrors.ServiceUnavailable("KIRO_OAUTH_REQUEST_FAILED", "kiro oauth request failed").WithCause(err)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return infraerrors.BadRequest("KIRO_OAUTH_UPSTREAM_REJECTED", fmt.Sprintf("kiro oauth request failed with status %d: %s", statusCode, sanitizeKiroBody(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return infraerrors.BadRequest("KIRO_OAUTH_INVALID_RESPONSE", "kiro oauth response is invalid").WithCause(err)
	}
	return nil
}

func (c *kiroOAuthClient) newRequest(ctx context.Context, method, endpoint string, payload any, proxyURL string) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	_ = proxyURL
	return req, nil
}

func (c *kiroOAuthClient) do(req *http.Request, proxyURL string) ([]byte, int, error) {
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              strings.TrimSpace(proxyURL),
		Timeout:               kiroHTTPTimeout,
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
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	return body, resp.StatusCode, nil
}

func buildKiroTokenInfo(accessToken, refreshToken, profileArn string, expiresIn int64) *service.KiroTokenInfo {
	if expiresIn <= 0 {
		expiresIn = int64(time.Hour.Seconds())
	}
	return &service.KiroTokenInfo{
		AccessToken:  strings.TrimSpace(accessToken),
		RefreshToken: strings.TrimSpace(refreshToken),
		ProfileArn:   strings.TrimSpace(profileArn),
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second).UTC().Format(time.RFC3339),
	}
}

func setKiroSocialHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "KiroIDE-sub2api-admin")
}

func setKiroOIDCHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/3.799.0 KiroIDE")
	req.Header.Set("User-Agent", "aws-sdk-js/3.799.0 ua/2.1 os/windows#10 lang/js md/nodejs#20 api/sso-oidc#3.799.0 m/E KiroIDE")
	req.Header.Set("amz-sdk-invocation-id", uuid.NewString())
	req.Header.Set("amz-sdk-request", "attempt=1; max=4")
}

func sanitizeKiroBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) > 512 {
		return trimmed[:512]
	}
	return trimmed
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
