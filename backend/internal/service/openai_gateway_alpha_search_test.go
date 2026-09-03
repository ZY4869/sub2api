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
	"github.com/tidwall/gjson"
)

type alphaSearchPATMetadataRepo struct {
	stubOpenAIAccountRepo
	updateCredentialsCalls int
	lastCredentials        map[string]any
}

func (r *alphaSearchPATMetadataRepo) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	r.updateCredentialsCalls++
	r.lastCredentials = cloneAccountCredentialsMap(credentials)
	return nil
}

func newOpenAIAlphaSearchTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/alpha/search?feature=standalone&locale=zh-CN", strings.NewReader(`{"query":"hello"}`))
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

func TestOpenAIGatewayServiceForwardAlphaSearchRejectsNonOpenAIAccount(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{statusCode: http.StatusOK, body: `{}`}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := &Account{
		ID:          7100,
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "deepseek-test"},
	}
	c, rec := newOpenAIAlphaSearchTestContext()

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"query":"hello"}`))

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "OpenAI OAuth or API key")
	require.Nil(t, upstream.lastReq)
	require.Empty(t, rec.Body.String())
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
	require.Equal(t, "feature=standalone&locale=zh-CN", upstream.lastReq.URL.RawQuery)
}

func TestResolveOpenAIAlphaSearchTargetURL(t *testing.T) {
	oauth := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "oauth-token"}}
	got, err := resolveOpenAIAlphaSearchTargetURL(oauth, nil)
	require.NoError(t, err)
	require.Equal(t, chatgptCodexAlphaSearchURL, got)

	apiKey := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "sk-test"}}
	got, err = resolveOpenAIAlphaSearchTargetURL(apiKey, nil)
	require.NoError(t, err)
	require.Equal(t, openaiPlatformAlphaSearchURL, got)

	custom := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://relay.example/v1"}}
	got, err = resolveOpenAIAlphaSearchTargetURL(custom, nil)
	require.NoError(t, err)
	require.Equal(t, "https://relay.example/v1/alpha/search", got)
}

func TestOpenAIGatewayServiceForwardAlphaSearch_PATUsesResponsesWebSearch(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusOK,
		header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"req-pat-search"}},
		body: "event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"answer"}` + "\n\n" +
			"event: response.output_text.annotation.added\n" +
			`data: {"type":"response.output_text.annotation.added","annotation":{"type":"url_citation","url":"https://example.com/source","title":"Example"}}` + "\n\n" +
			"event: response.completed\n" +
			`data: {"type":"response.completed","response":{"output":[{"type":"message","content":[{"type":"output_text","text":"answer"}]}]}}` + "\n\n",
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, responseHeaderFilter: compileResponseHeaderFilter(&config.Config{})}
	account := &Account{
		ID: 7103, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "at-test-token", "auth_mode": "personalAccessToken", "chatgpt_account_id": "chatgpt-account"},
	}
	c, rec := newOpenAIAlphaSearchTestContext()
	c.Request.Header.Set("X-Codex-Turn-Metadata", `{"turn_id":"turn-1"}`)
	c.Request.Header.Set("Accept-Language", "zh-CN")

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"model":"gpt-5.6-sol","commands":{"search_query":[{"q":"latest news"}]},"settings":{"search_context_size":"high"}}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, EndpointResponses, result.UpstreamEndpoint)
	require.Equal(t, chatgptCodexURL, upstream.lastReq.URL.String())
	require.Equal(t, "Bearer at-test-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "chatgpt-account", upstream.lastReq.Header.Get("ChatGPT-Account-ID"))
	require.Equal(t, "text/event-stream", upstream.lastReq.Header.Get("Accept"))
	require.Equal(t, "responses=experimental", upstream.lastReq.Header.Get("OpenAI-Beta"))
	require.Equal(t, `{"turn_id":"turn-1"}`, upstream.lastReq.Header.Get("X-Codex-Turn-Metadata"))
	require.Empty(t, upstream.lastReq.Header.Get("Accept-Language"))
	require.True(t, gjson.Get(upstream.lastBody, "stream").Bool())
	require.False(t, gjson.Get(upstream.lastBody, "store").Bool())
	require.Equal(t, "web_search", gjson.Get(upstream.lastBody, "tools.0.type").String())
	require.Equal(t, "high", gjson.Get(upstream.lastBody, "tools.0.search_context_size").String())
	require.Contains(t, gjson.Get(upstream.lastBody, "input.0.content.0.text").String(), "search_query")
	require.NotContains(t, gjson.Get(upstream.lastBody, "input.0.content.0.text").String(), "gpt-5.6-sol")
	require.JSONEq(t, `{"output":"answer","results":[{"type":"text_result","ref_id":"turn0search0","url":"https://example.com/source","title":"Example"}]}`, rec.Body.String())
}

func TestConvertResponsesSSEToAlphaSearchJSONUsesCompletedTextWhenNoDelta(t *testing.T) {
	body := []byte("event: response.completed\n" + `data: {"type":"response.completed","response":{"output":[{"type":"message","content":[{"type":"output_text","text":"completed answer"}]}]}}` + "\n\n")
	require.JSONEq(t, `{"output":"completed answer"}`, string(convertResponsesSSEToAlphaSearchJSON(body)))
}

func TestOpenAIGatewayServiceForwardAlphaSearch_PATUnauthorizedIsRequestScoped(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{statusCode: http.StatusUnauthorized, body: `{"error":{"message":"unauthorized"}}`}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := &Account{ID: 7104, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "at-test-token", "auth_mode": "personalAccessToken"}}
	c, rec := newOpenAIAlphaSearchTestContext()

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"model":"gpt-5.6-sol","commands":{"search_query":[{"q":"news"}]}}`))

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusUnauthorized, failoverErr.StatusCode)
	require.False(t, rec.Code == http.StatusUnauthorized)
	require.Equal(t, chatgptCodexURL, upstream.lastReq.URL.String())
}

func TestOpenAIGatewayServiceForwardAlphaSearch_PATNonSSE2xxIsProtocolFailover(t *testing.T) {
	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusOK,
		header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"req-invalid-sse"}},
		body:       `{"output":"looks successful"}`,
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := &Account{ID: 7105, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "at-test-token", "auth_mode": "personalAccessToken", "chatgpt_account_id": "chatgpt-account"}}
	c, rec := newOpenAIAlphaSearchTestContext()

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"model":"gpt-5.6-sol","commands":{"search_query":[{"q":"news"}]}}`))

	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, failoverErr.Message, "invalid SSE")
	require.Empty(t, rec.Body.String())
}

func TestOpenAIGatewayServiceForwardAlphaSearch_PATPersistsWhoamiMetadataOnce(t *testing.T) {
	var whoamiAuthorization string
	whoami := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		whoamiAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"user@example.com","chatgpt_user_id":"user-123","chatgpt_account_id":"acct-123","chatgpt_plan_type":"plus","chatgpt_account_is_fedramp":false}`))
	}))
	defer whoami.Close()
	previousWhoami := openAICodexPATWhoamiURL
	openAICodexPATWhoamiURL = whoami.URL
	defer func() { openAICodexPATWhoamiURL = previousWhoami }()

	upstream := &openAIEmbeddingsHTTPUpstreamStub{
		statusCode: http.StatusOK,
		header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		body:       "data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n",
	}
	account := &Account{ID: 7106, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{
		"access_token": "at-metadata", "auth_mode": "personalAccessToken", "refresh_token": "must-not-persist", "expires_at": "2099-01-01T00:00:00Z",
	}}
	repo := &alphaSearchPATMetadataRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{*account}}}
	oauthSvc := NewOpenAIOAuthService(nil, nil)
	defer oauthSvc.Stop()
	tokenProvider := NewOpenAITokenProvider(repo, nil, oauthSvc)
	svc := &OpenAIGatewayService{accountRepo: repo, openAITokenProvider: tokenProvider, cfg: &config.Config{}, httpUpstream: upstream}
	c, rec := newOpenAIAlphaSearchTestContext()

	result, err := svc.ForwardAlphaSearch(context.Background(), c, account, []byte(`{"commands":{"search_query":[{"q":"news"}]}}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Bearer at-metadata", whoamiAuthorization)
	require.JSONEq(t, `{"output":"ok"}`, rec.Body.String())
	require.Equal(t, 1, repo.updateCredentialsCalls)
	require.Equal(t, "acct-123", repo.lastCredentials["chatgpt_account_id"])
	require.Equal(t, "user-123", repo.lastCredentials["chatgpt_user_id"])
	require.Equal(t, OpenAIAuthModePersonalAccessToken, repo.lastCredentials["auth_mode"])
	require.NotContains(t, repo.lastCredentials, "refresh_token")
	require.NotContains(t, repo.lastCredentials, "expires_at")
}
