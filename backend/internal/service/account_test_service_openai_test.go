//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)
	return c, rec
}

type openAIAccountTestRepo struct {
	mockAccountRepoForGemini
	updatedExtra     map[string]any
	updateExtraCalls []map[string]any
	rateLimitedID    int64
	rateLimitedAt    *time.Time
	setErrorCalls    []string
}

func (r *openAIAccountTestRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updatedExtra = updates
	r.updateExtraCalls = append(r.updateExtraCalls, updates)
	return nil
}

func (r *openAIAccountTestRepo) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	r.rateLimitedID = id
	r.rateLimitedAt = &resetAt
	return nil
}

func (r *openAIAccountTestRepo) SetError(_ context.Context, _ int64, errorMsg string) error {
	r.setErrorCalls = append(r.setErrorCalls, errorMsg)
	return nil
}

func TestAccountTestService_OpenAISuccessPersistsSnapshotFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	resp.Header.Set("x-codex-primary-used-percent", "88")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "42")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          89,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, repo.updatedExtra)
	require.Equal(t, 42.0, repo.updatedExtra["codex_5h_used_percent"])
	require.Equal(t, 88.0, repo.updatedExtra["codex_7d_used_percent"])
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAI429PersistsSnapshotAndRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached"}}`)
	resp.Header.Set("x-codex-primary-used-percent", "100")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "100")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          88,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Len(t, repo.updateExtraCalls, 3)
	require.Equal(t, 100.0, repo.updateExtraCalls[0]["codex_5h_used_percent"])
	require.Equal(t, 100.0, repo.updateExtraCalls[0]["codex_7d_used_percent"])
	require.Equal(t, AccountRateLimitReasonUsage7d, repo.updateExtraCalls[1]["rate_limit_reason"])
	require.Equal(t, AccountRateLimitReasonUsage7d, repo.updateExtraCalls[2]["rate_limit_reason"])
	require.Equal(t, int64(88), repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.NotNil(t, account.RateLimitResetAt)
	if account.RateLimitResetAt != nil && repo.rateLimitedAt != nil {
		require.WithinDuration(t, *repo.rateLimitedAt, *account.RateLimitResetAt, time.Second)
	}
}

func TestAccountTestService_OpenAIUnauthorizedDetailMarksAccountError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusUnauthorized, `{"detail":"Unauthorized"}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          87,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")

	require.Error(t, err)
	require.Len(t, repo.setErrorCalls, 1)
	require.Contains(t, repo.setErrorCalls[0], `Authentication failed (401): {"detail":"Unauthorized"}`)
}

func TestAccountTestService_OpenAISuccessProbesKnownModelsInBackground(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newGatewayTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	resp.Header.Set("x-codex-primary-used-percent", "88")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "42")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	modelsResp := newJSONResponse(http.StatusOK, `{"data":[{"id":"gpt-5.4"},{"id":"gpt-4.1-mini"}]}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp, modelsResp}}
	importSvc := NewAccountModelImportService(nil, nil, upstream, nil)
	svc := &AccountTestService{
		accountRepo:               repo,
		accountModelImportService: importSvc,
		httpUpstream:              upstream,
		backgroundRunner: func(fn func()) {
			fn()
		},
	}
	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.Len(t, repo.updateExtraCalls, 2)
	require.Equal(t, OpenAIKnownModelsSourceTestProbe, repo.updateExtraCalls[1]["openai_known_models_source"])
	require.Equal(t, []string{"gpt-5.4", "gpt-4.1-mini"}, repo.updateExtraCalls[1]["openai_known_models"])
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAISuccessProbeFailureKeepsExistingKnownModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newGatewayTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	modelsResp := newJSONResponse(http.StatusInternalServerError, `{"error":"boom"}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp, modelsResp}}
	importSvc := NewAccountModelImportService(nil, nil, upstream, nil)
	svc := &AccountTestService{
		accountRepo:               repo,
		accountModelImportService: importSvc,
		httpUpstream:              upstream,
		backgroundRunner: func(fn func()) {
			fn()
		},
	}
	account := &Account{
		ID:          91,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
		Extra: map[string]any{
			"openai_known_models":            []string{"existing-model"},
			"openai_known_models_source":     OpenAIKnownModelsSourceImportModels,
			"openai_known_models_updated_at": "2026-03-10T10:00:00Z",
		},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.Len(t, repo.updateExtraCalls, 0)
	require.Equal(t, []string{"existing-model"}, account.Extra["openai_known_models"])
	require.Equal(t, OpenAIKnownModelsSourceImportModels, account.Extra["openai_known_models_source"])
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestEnsureOpenAIOAuthTestTargetExtra_FillsMissingDefault(t *testing.T) {
	extra := EnsureOpenAIOAuthTestTargetExtra(nil)
	require.Equal(t, PlatformOpenAI, extra[gatewayExtraTestProviderKey])
	require.Equal(t, OpenAIOAuthDefaultTestModelID, extra[gatewayExtraTestModelIDKey])
}

func TestEnsureOpenAIOAuthTestTargetExtra_PreservesExplicitSetting(t *testing.T) {
	extra := EnsureOpenAIOAuthTestTargetExtra(map[string]any{
		gatewayExtraTestProviderKey: PlatformAnthropic,
		gatewayExtraTestModelIDKey:  "claude-sonnet-4.5",
	})
	require.Equal(t, PlatformAnthropic, extra[gatewayExtraTestProviderKey])
	require.Equal(t, "claude-sonnet-4.5", extra[gatewayExtraTestModelIDKey])
}

func TestDefaultOpenAIOAuthTestModelID_PrefersGPT54WithoutSnapshot(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	require.Equal(t, OpenAIOAuthDefaultTestModelID, defaultOpenAIOAuthTestModelID(context.Background(), account, nil))
}

func TestDefaultOpenAIOAuthTestModelID_FallsBackWhenKnownModelsExcludeGPT54(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"openai_known_models": []string{"gpt-4.1-mini", "o4-mini"},
		},
	}

	require.Equal(t, "gpt-4.1-mini", defaultOpenAIOAuthTestModelID(context.Background(), account, nil))
}

func TestAccountTestService_OpenAIProtocolGatewayChatPreferenceUsesChatCompletionsHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Body = io.NopCloser(strings.NewReader("data: {\"id\":\"chatcmpl_1\",\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{httpUpstream: upstream, cfg: &config.Config{}}
	account := &Account{
		ID:          92,
		Platform:    PlatformProtocolGateway,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-token", "base_url": "https://api.openai.com"},
		Extra: map[string]any{
			"gateway_protocol":              GatewayProtocolOpenAI,
			"gateway_openai_request_format": GatewayOpenAIRequestFormatChatCompletions,
		},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", PlatformOpenAI, "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "/v1/chat/completions", upstream.requests[0].URL.Path)
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_RunTestBackgroundDetailed_InheritsGatewayOpenAIRequestFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Body = io.NopCloser(strings.NewReader("data: {\"id\":\"chatcmpl_bg_1\",\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"))

	account := &Account{
		ID:          93,
		Platform:    PlatformProtocolGateway,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-token", "base_url": "https://api.openai.com"},
		Extra: map[string]any{
			"gateway_protocol":              GatewayProtocolOpenAI,
			"gateway_openai_request_format": GatewayOpenAIRequestFormatChatCompletions,
		},
	}
	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				account.ID: account,
			},
		},
	}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.RunTestBackgroundDetailed(context.Background(), ScheduledTestExecutionInput{
		AccountID: account.ID,
		ModelID:   "gpt-5.4",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "success", result.Status)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "/v1/chat/completions", upstream.requests[0].URL.Path)
}
