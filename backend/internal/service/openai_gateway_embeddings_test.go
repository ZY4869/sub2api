package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIEmbeddingsHTTPUpstreamStub struct {
	statusCode int
	body       string
	header     http.Header

	lastReq  *http.Request
	lastBody string
}

func (s *openAIEmbeddingsHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return s.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (s *openAIEmbeddingsHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *tlsfingerprint.Profile) (*http.Response, error) {
	s.lastReq = req
	if req != nil && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		s.lastBody = string(body)
		req.Body = io.NopCloser(strings.NewReader(s.lastBody))
	}
	header := s.header
	if header == nil {
		header = http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"req_embed"}}
	}
	return &http.Response{
		StatusCode: s.statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

func newOpenAIEmbeddingsTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "sdk-test")
	return c, rec
}

func TestOpenAIGatewayServiceForwardEmbeddings_ForwardsMappedModelAndUsage(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusOK,
		body:       `{"object":"list","data":[],"model":"text-embedding-3-small","usage":{"input_tokens":11,"input_tokens_details":{"cached_tokens":3}}}`,
	}
	svc := &OpenAIGatewayService{
		cfg:                  &config.Config{},
		httpUpstream:         upstream,
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := &Account{
		ID:       42,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "sk-test",
			"model_mapping": map[string]any{
				"embed-public": "text-embedding-3-small",
			},
		},
	}
	c, rec := newOpenAIEmbeddingsTestContext()

	result, err := svc.ForwardEmbeddings(context.Background(), c, account, []byte(`{"model":"embed-public","input":"hello"}`))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, result)
	require.Equal(t, 11, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.CacheReadInputTokens)
	require.Equal(t, "embed-public", result.Model)
	require.Equal(t, "text-embedding-3-small", result.UpstreamModel)
	require.Equal(t, "https://api.openai.com/v1/embeddings", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-test", upstream.lastReq.Header.Get("Authorization"))
	require.Contains(t, upstream.lastBody, `"model":"text-embedding-3-small"`)
	require.Contains(t, rec.Body.String(), `"model":"embed-public"`)
}

func TestOpenAIGatewayServiceForwardEmbeddings_RejectsOAuthAccount(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	c, rec := newOpenAIEmbeddingsTestContext()

	result, err := svc.ForwardEmbeddings(context.Background(), c, account, []byte(`{"model":"embed-public","input":"hello"}`))

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "OpenAI embeddings require an OpenAI API-key account")
}

func TestOpenAIGatewayServiceForwardEmbeddings_RejectsNonOpenAIAPIKeyAccounts(t *testing.T) {
	tests := []struct {
		name     string
		platform string
	}{
		{name: "deepseek", platform: PlatformDeepSeek},
		{name: "openrouter", platform: PlatformOpenRouter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &OpenAIGatewayService{cfg: &config.Config{}}
			account := &Account{Platform: tt.platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "sk-test"}}
			c, rec := newOpenAIEmbeddingsTestContext()

			result, err := svc.ForwardEmbeddings(context.Background(), c, account, []byte(`{"model":"embed-public","input":"hello"}`))

			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, http.StatusForbidden, rec.Code)
			require.Contains(t, rec.Body.String(), "OpenAI embeddings require an OpenAI API-key account")
		})
	}
}

func TestOpenAIGatewayServiceForwardEmbeddings_Upstream5xxReturnsFailoverError(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusInternalServerError,
		body:       `{"error":{"message":"server down"}}`,
	}
	svc := &OpenAIGatewayService{
		cfg:                  &config.Config{},
		httpUpstream:         upstream,
		rateLimitService:     NewRateLimitService(stubOpenAIAccountRepo{}, nil, &config.Config{}, nil, nil),
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := &Account{
		ID:       44,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}
	c, rec := newOpenAIEmbeddingsTestContext()

	result, err := svc.ForwardEmbeddings(context.Background(), c, account, []byte(`{"model":"text-embedding-3-small","input":"hello"}`))

	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusInternalServerError, failoverErr.StatusCode)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestOpenAIGatewayServiceForwardEmbeddings_ExtractsPromptTokenUsageFallback(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusOK,
		body:       `{"object":"list","data":[],"model":"text-embedding-3-small","usage":{"prompt_tokens":7,"total_tokens":9}}`,
	}
	svc := &OpenAIGatewayService{
		cfg:                  &config.Config{},
		httpUpstream:         upstream,
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := &Account{
		ID:          45,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	c, _ := newOpenAIEmbeddingsTestContext()

	result, err := svc.ForwardEmbeddings(context.Background(), c, account, []byte(`{"model":"text-embedding-3-small","input":"hello"}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 7, result.Usage.InputTokens)
}
