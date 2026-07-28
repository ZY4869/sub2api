package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type grokAccountTestHTTPUpstream struct {
	responses []*http.Response
	requests  []*http.Request
	callCount int
}

func (s *grokAccountTestHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	var bodyBytes []byte
	if req != nil && req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	if req != nil {
		cloned := req.Clone(req.Context())
		cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		s.requests = append(s.requests, cloned)
	}
	idx := s.callCount
	s.callCount++
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	return s.responses[idx], nil
}

func (s *grokAccountTestHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *TLSFingerprintProfile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestAccountTestServiceGrokOAuthProbesThenMakesRealResponsesCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &grokAccountTestHTTPUpstream{responses: []*http.Response{
		{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"not found"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_ok","usage":{"input_tokens":1,"output_tokens":1},"output_text":"OK"}`)),
		},
	}}
	importSvc := NewAccountModelImportService(NewModelCatalogService(newAccountModelImportSettingRepoStub(), nil, nil, nil, nil), nil, upstream, nil)
	grokSvc := &GrokGatewayService{httpUpstream: upstream, cfg: &config.Config{}}
	testSvc := &AccountTestService{accountModelImportService: importSvc, grokGatewayService: grokSvc, cfg: &config.Config{}}
	account := &Account{
		ID:       2415,
		Name:     "grok-oauth",
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://api.x.ai/v1",
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/grok/accounts/2415/test", nil)

	err := testSvc.testGrokOfficialConnection(c, account, GrokModelBuild45)

	require.NoError(t, err)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/models", upstream.requests[0].URL.String())
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/responses", upstream.requests[1].URL.String())
	require.Equal(t, grokUpstreamUserAgent, upstream.requests[1].Header.Get("User-Agent"))
	require.Equal(t, grokCLIVersion, upstream.requests[1].Header.Get("X-Grok-Client-Version"))
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
	require.Contains(t, rec.Body.String(), "Grok real model call OK")
	require.Contains(t, rec.Body.String(), `"type":"content","text":"OK"`)
	require.NotContains(t, rec.Body.String(), `"type":"content","text":"Grok official runtime connectivity OK`)
	require.NotContains(t, rec.Body.String(), `"type":"content","text":"Detected`)
	bodyBytes, err := io.ReadAll(upstream.requests[1].Body)
	require.NoError(t, err)
	require.Equal(t, GrokModelBuild45, gjson.GetBytes(bodyBytes, "model").String())
	require.Equal(t, "Output exactly: OK", gjson.GetBytes(bodyBytes, "input").String())
}

func TestAccountTestServiceGrokRealResponses404Fails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &grokAccountTestHTTPUpstream{responses: []*http.Response{
		{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"model listing not found"}}`)),
		},
		{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"The requested resource was not found."}}`)),
		},
	}}
	importSvc := NewAccountModelImportService(NewModelCatalogService(newAccountModelImportSettingRepoStub(), nil, nil, nil, nil), nil, upstream, nil)
	grokSvc := &GrokGatewayService{httpUpstream: upstream, cfg: &config.Config{}}
	testSvc := &AccountTestService{accountModelImportService: importSvc, grokGatewayService: grokSvc, cfg: &config.Config{}}
	account := &Account{
		ID:       2415,
		Name:     "grok-oauth",
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://api.x.ai/v1",
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/grok/accounts/2415/test", nil)

	err := testSvc.testGrokOfficialConnection(c, account, GrokModelBuild45)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Grok real model call failed")
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/responses", upstream.requests[1].URL.String())
	require.Contains(t, rec.Body.String(), `"type":"error"`)
	require.Contains(t, rec.Body.String(), "upstream status 404")
}

func TestAccountTestServiceGrokRealResponses402PausesAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &grokAccountTestHTTPUpstream{responses: []*http.Response{
		{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"model listing not found"}}`)),
		},
		{
			StatusCode: http.StatusPaymentRequired,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"insufficient balance"}}`)),
		},
	}}
	importSvc := NewAccountModelImportService(NewModelCatalogService(newAccountModelImportSettingRepoStub(), nil, nil, nil, nil), nil, upstream, nil)
	grokSvc := &GrokGatewayService{httpUpstream: upstream, cfg: &config.Config{}}
	repo := &grokAccountTestRateLimitRepo{}
	testSvc := &AccountTestService{
		accountModelImportService: importSvc,
		grokGatewayService:        grokSvc,
		rateLimitService:          NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
		cfg:                       &config.Config{},
	}
	account := &Account{
		ID:       2418,
		Name:     "grok-oauth",
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://api.x.ai/v1",
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/grok/accounts/2418/test", nil)

	err := testSvc.testGrokOfficialConnection(c, account, GrokModelBuild45)

	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream status 402")
	require.Equal(t, 1, repo.setErrorCalls)
	require.Contains(t, repo.lastErrorMessage, "Payment required (402)")
	require.Contains(t, repo.lastErrorMessage, "insufficient balance")
}

type grokAccountTestRateLimitRepo struct {
	AccountRepository
	setErrorCalls    int
	lastErrorMessage string
}

func (r *grokAccountTestRateLimitRepo) SetError(_ context.Context, _ int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorMessage = errorMsg
	return nil
}

func TestAccountTestServiceGrokAPIKeyAcceptsV1BaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &grokAccountTestHTTPUpstream{responses: []*http.Response{
		{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"not found"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_ok","output":[{"content":[{"type":"output_text","text":"OK"}]}]}`)),
		},
	}}
	importSvc := NewAccountModelImportService(NewModelCatalogService(newAccountModelImportSettingRepoStub(), nil, nil, nil, nil), nil, upstream, nil)
	grokSvc := &GrokGatewayService{httpUpstream: upstream, cfg: &config.Config{}}
	testSvc := &AccountTestService{accountModelImportService: importSvc, grokGatewayService: grokSvc, cfg: &config.Config{}}
	account := &Account{
		ID:       2416,
		Name:     "grok-apikey",
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "xai-key",
			"base_url": "https://api.x.ai/v1",
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/grok/accounts/2416/test", nil)

	err := testSvc.testGrokOfficialConnection(c, account, GrokModelBuild45)

	require.NoError(t, err)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://api.x.ai/v1/models", upstream.requests[0].URL.String())
	require.Equal(t, "https://api.x.ai/v1/responses", upstream.requests[1].URL.String())
	require.Empty(t, upstream.requests[1].Header.Get("X-Grok-Client-Version"))
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
}

func TestAccountTestServiceGrokSSODiagnosticsDoNotUseContentEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &grokAccountTestHTTPUpstream{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(`data: {"result":{"response":{"token":"OK","responseId":"resp_sso","conversationId":"conv_sso"}}}

data: [DONE]
`)),
		},
	}}
	grokSvc := &GrokGatewayService{
		reverseClient: NewGrokReverseClient(upstream, &config.Config{}),
		cfg:           &config.Config{},
	}
	testSvc := &AccountTestService{grokGatewayService: grokSvc, cfg: &config.Config{}}
	account := &Account{
		ID:       2417,
		Name:     "grok-sso",
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
		Credentials: map[string]any{
			"sso_token": "legacy-token",
		},
		Extra: map[string]any{
			"grok_tier": GrokTierBasic,
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/grok/accounts/2417/test", nil)

	err := testSvc.testGrokSSOConnection(c, account, GrokModelBuild45)

	require.NoError(t, err)
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
	require.Contains(t, rec.Body.String(), `"type":"status","text":"Reverse runtime connectivity probe started"`)
	require.Contains(t, rec.Body.String(), `"type":"log","text":"Tier:`)
	require.NotContains(t, rec.Body.String(), `"type":"content"`)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://grok.com/rest/app-chat/conversations/new", upstream.requests[0].URL.String())
}
