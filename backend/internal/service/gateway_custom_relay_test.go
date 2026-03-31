package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type relayHTTPUpstreamRecorder struct {
	lastReq      *http.Request
	lastBody     []byte
	lastProxyURL string
	resp         *http.Response
	err          error
}

func (u *relayHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return u.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (u *relayHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *TLSFingerprintProfile) (*http.Response, error) {
	u.lastReq = req
	u.lastProxyURL = proxyURL
	if req != nil && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		u.lastBody = body
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if u.err != nil {
		return nil, u.err
	}
	return u.resp, nil
}

func newCustomRelayOAuthAccount() *Account {
	proxyID := int64(12)
	return &Account{
		ID:          411,
		Name:        "anthropic-custom-relay",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
		},
		Extra: map[string]any{
			"custom_base_url_enabled": true,
			"custom_base_url":         "https://relay.example.com/base",
		},
		ProxyID: &proxyID,
		Proxy: &Proxy{
			Protocol: "http",
			Host:     "proxy.example.com",
			Port:     8080,
			Username: "user",
			Password: "pass",
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func newGatewayServiceWithOpenURLPolicy() *GatewayService {
	return &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
	}
}

func TestBuildUpstreamRequest_UsesCustomRelayURLForOAuthAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	req, err := newGatewayServiceWithOpenURLPolicy().buildUpstreamRequest(
		context.Background(),
		c,
		newCustomRelayOAuthAccount(),
		[]byte(`{"model":"claude-sonnet-4"}`),
		"oauth-token",
		"oauth",
		"claude-sonnet-4",
		false,
		false,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		"https://relay.example.com/base/v1/messages?beta=true&proxy=http%3A%2F%2Fuser%3Apass%40proxy.example.com%3A8080",
		req.URL.String(),
	)
	require.Equal(t, "Bearer oauth-token", req.Header.Get("authorization"))
}

func TestBuildCountTokensRequest_UsesCustomRelayURLForOAuthAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

	req, err := newGatewayServiceWithOpenURLPolicy().buildCountTokensRequest(
		context.Background(),
		c,
		newCustomRelayOAuthAccount(),
		[]byte(`{"model":"claude-sonnet-4"}`),
		"oauth-token",
		"oauth",
		"claude-sonnet-4",
		false,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		"https://relay.example.com/base/v1/messages/count_tokens?beta=true&proxy=http%3A%2F%2Fuser%3Apass%40proxy.example.com%3A8080",
		req.URL.String(),
	)
}

func TestGatewayService_Forward_CustomRelayUsesQueryProxyInsteadOfHTTPProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	upstream := &relayHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-custom-relay"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"id":"msg_1","type":"message","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}

	svc := newGatewayServiceWithOpenURLPolicy()
	svc.httpUpstream = upstream
	svc.rateLimitService = &RateLimitService{}

	result, err := svc.Forward(context.Background(), c, newCustomRelayOAuthAccount(), &ParsedRequest{
		Body:   []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`),
		Model:  "claude-sonnet-4",
		Stream: false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, upstream.lastProxyURL)
	require.Equal(
		t,
		"https://relay.example.com/base/v1/messages?beta=true&proxy=http%3A%2F%2Fuser%3Apass%40proxy.example.com%3A8080",
		upstream.lastReq.URL.String(),
	)
}

func TestGatewayService_ForwardCountTokens_CustomRelayUsesQueryProxyInsteadOfHTTPProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

	upstream := &relayHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"input_tokens":42}`)),
		},
	}

	svc := newGatewayServiceWithOpenURLPolicy()
	svc.cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc.httpUpstream = upstream
	svc.rateLimitService = &RateLimitService{}

	err := svc.ForwardCountTokens(context.Background(), c, newCustomRelayOAuthAccount(), &ParsedRequest{
		Body:  []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`),
		Model: "claude-sonnet-4",
	})
	require.NoError(t, err)
	require.Empty(t, upstream.lastProxyURL)
	require.Equal(
		t,
		"https://relay.example.com/base/v1/messages/count_tokens?beta=true&proxy=http%3A%2F%2Fuser%3Apass%40proxy.example.com%3A8080",
		upstream.lastReq.URL.String(),
	)
}
