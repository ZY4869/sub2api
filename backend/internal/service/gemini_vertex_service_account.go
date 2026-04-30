package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/golang-jwt/jwt/v5"
)

const (
	vertexServiceAccountScope     = "https://www.googleapis.com/auth/cloud-platform"
	vertexServiceAccountTokenPath = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	// Security: never trust token_uri from uploaded JSON. Always exchange tokens
	// against the official Google OAuth token endpoint to avoid SSRF.
	vertexServiceAccountTokenURL = "https://oauth2.googleapis.com/token"
)

type vertexServiceAccountCredentials struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

type vertexServiceAccountTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func parseVertexServiceAccountCredentials(raw string) (*vertexServiceAccountCredentials, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("missing vertex_service_account_json")
	}
	var creds vertexServiceAccountCredentials
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, fmt.Errorf("invalid vertex_service_account_json: %w", err)
	}
	if strings.TrimSpace(creds.Type) != "service_account" {
		return nil, fmt.Errorf("vertex_service_account_json must be a service_account credential")
	}
	if strings.TrimSpace(creds.ClientEmail) == "" {
		return nil, fmt.Errorf("vertex_service_account_json missing client_email")
	}
	if strings.TrimSpace(creds.PrivateKey) == "" {
		return nil, fmt.Errorf("vertex_service_account_json missing private_key")
	}
	if strings.TrimSpace(creds.TokenURI) == "" {
		return nil, fmt.Errorf("vertex_service_account_json missing token_uri")
	}
	return &creds, nil
}

func parseVertexServiceAccountPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid vertex service account private_key PEM")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("vertex service account private_key is not RSA")
		}
		return rsaKey, nil
	}
	if rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rsaKey, nil
	}
	return nil, fmt.Errorf("failed to parse vertex service account private_key")
}

func buildVertexServiceAccountAssertion(creds *vertexServiceAccountCredentials, now time.Time) (string, error) {
	if creds == nil {
		return "", fmt.Errorf("vertex service account credentials are nil")
	}
	privateKey, err := parseVertexServiceAccountPrivateKey(creds.PrivateKey)
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"iss":   creds.ClientEmail,
		"sub":   creds.ClientEmail,
		"aud":   vertexServiceAccountTokenURL,
		"scope": vertexServiceAccountScope,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if strings.TrimSpace(creds.PrivateKeyID) != "" {
		token.Header["kid"] = strings.TrimSpace(creds.PrivateKeyID)
	}
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign vertex service account assertion: %w", err)
	}
	return signed, nil
}

func (p *GeminiTokenProvider) resolveVertexProxyURL(ctx context.Context, account *Account) string {
	if account == nil || account.ProxyID == nil || p == nil || p.geminiOAuthService == nil || p.geminiOAuthService.proxyRepo == nil {
		return ""
	}
	proxy, err := p.geminiOAuthService.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil || proxy == nil {
		return ""
	}
	return proxy.URL()
}

func (p *GeminiTokenProvider) getVertexServiceAccountAccessToken(ctx context.Context, account *Account, cacheKey string) (string, error) {
	if account == nil {
		return "", fmt.Errorf("account is nil")
	}
	creds, err := parseVertexServiceAccountCredentials(account.GetCredential("vertex_service_account_json"))
	if err != nil {
		return "", err
	}
	if p.tokenCache != nil {
		if token, cacheErr := p.tokenCache.GetAccessToken(ctx, cacheKey); cacheErr == nil && strings.TrimSpace(token) != "" {
			return token, nil
		}
	}

	locked := false
	if p.tokenCache != nil {
		locked, err = p.tokenCache.AcquireRefreshLock(ctx, cacheKey, 30*time.Second)
		if err == nil && locked {
			defer func() { _ = p.tokenCache.ReleaseRefreshLock(ctx, cacheKey) }()
		}
		if err == nil && !locked {
			for i := 0; i < 5; i++ {
				time.Sleep(200 * time.Millisecond)
				if token, cacheErr := p.tokenCache.GetAccessToken(ctx, cacheKey); cacheErr == nil && strings.TrimSpace(token) != "" {
					return token, nil
				}
			}
		}
	}

	token, ttl, err := p.exchangeVertexServiceAccountToken(ctx, account, creds)
	if err != nil {
		return "", err
	}
	if p.tokenCache != nil && strings.TrimSpace(token) != "" && ttl > 0 {
		_ = p.tokenCache.SetAccessToken(ctx, cacheKey, token, ttl)
	}
	return token, nil
}

func (p *GeminiTokenProvider) exchangeVertexServiceAccountToken(ctx context.Context, account *Account, creds *vertexServiceAccountCredentials) (string, time.Duration, error) {
	assertion, err := buildVertexServiceAccountAssertion(creds, time.Now())
	if err != nil {
		return "", 0, err
	}

	form := url.Values{}
	form.Set("grant_type", vertexServiceAccountTokenPath)
	form.Set("assertion", assertion)

	proxyURL := p.resolveVertexProxyURL(ctx, account)
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               20 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ValidateResolvedIP:    true,
	})
	if err != nil {
		return "", 0, fmt.Errorf("build vertex service account http client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vertexServiceAccountTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("build vertex service account token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("request vertex service account token: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", 0, fmt.Errorf("vertex service account token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp vertexServiceAccountTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("parse vertex service account token response: %w", err)
	}
	accessToken := strings.TrimSpace(tokenResp.AccessToken)
	if accessToken == "" {
		return "", 0, fmt.Errorf("vertex service account token exchange returned an empty access_token")
	}

	expiresIn := time.Duration(tokenResp.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}
	ttl := expiresIn - geminiTokenCacheSkew
	if ttl <= 0 {
		ttl = expiresIn
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return accessToken, ttl, nil
}
