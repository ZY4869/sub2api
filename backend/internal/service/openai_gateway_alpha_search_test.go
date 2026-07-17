package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newOpenAIAlphaSearchTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/alpha/search", strings.NewReader(`{"query":"hello"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "sdk-test")
	return c, rec
}

func TestOpenAIGatewayServiceForwardAlphaSearch_APIKeyEndpointUnsupportedReturnsFailover(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusNotFound,
		body:       `{"error":{"message":"unknown endpoint"}}`,
	}
	svc := &OpenAIGatewayService{
		cfg:                  &config.Config{},
		httpUpstream:         upstream,
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := &Account{
		ID:          7101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	c, rec := newOpenAIAlphaSearchTestContext()

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"query":"hello"}`))

	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusNotFound, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "unknown endpoint")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())
	require.Empty(t, rec.Result().Header.Get("Content-Type"))
}

func TestOpenAIGatewayServiceForwardAlphaSearch_OAuthEndpointUnsupportedPassesThrough(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusNotFound,
		body:       `{"error":{"message":"unknown endpoint"}}`,
	}
	svc := &OpenAIGatewayService{
		cfg:                  &config.Config{},
		httpUpstream:         upstream,
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := &Account{
		ID:          7102,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "oauth-token"},
	}
	c, rec := newOpenAIAlphaSearchTestContext()

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"query":"hello"}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "unknown endpoint")
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
}
