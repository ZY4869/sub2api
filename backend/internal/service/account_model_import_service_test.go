package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/stretchr/testify/require"
)

type accountModelImportSettingRepoStub struct {
	values       map[string]string
	failContains string
}

func newAccountModelImportSettingRepoStub() *accountModelImportSettingRepoStub {
	return &accountModelImportSettingRepoStub{values: make(map[string]string)}
}

func (s *accountModelImportSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *accountModelImportSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *accountModelImportSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.failContains != "" && strings.Contains(value, s.failContains) {
		return errors.New("persist failed")
	}
	s.values[key] = value
	return nil
}

func (s *accountModelImportSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func expectedNormalizedGeminiCLIDefaultModelIDs() []string {
	ids := make([]string, 0, len(geminicli.DefaultModels))
	for _, model := range geminicli.DefaultModels {
		ids = append(ids, model.ID)
	}
	normalized, _ := normalizeImportedModelIDs(ids)
	return normalized
}

func (s *accountModelImportSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	for key, value := range settings {
		if err := s.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *accountModelImportSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *accountModelImportSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type accountModelImportHTTPUpstreamStub struct {
	statusCode int
	body       string
	err        error
	headers    http.Header

	lastReq                  *http.Request
	lastProxyURL             string
	lastAccountID            int64
	lastAccountConcurrency   int
	lastEnableTLSFingerprint bool
}

func (s *accountModelImportHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return s.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (s *accountModelImportHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *TLSFingerprintProfile) (*http.Response, error) {
	s.lastReq = req
	s.lastProxyURL = proxyURL
	s.lastAccountID = accountID
	s.lastAccountConcurrency = accountConcurrency
	s.lastEnableTLSFingerprint = tlsProfile != nil
	if s.err != nil {
		return nil, s.err
	}
	statusCode := s.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	respHeaders := make(http.Header)
	for key, values := range s.headers {
		for _, value := range values {
			respHeaders.Add(key, value)
		}
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     respHeaders,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Request:    req,
	}, nil
}

type accountModelImportGeminiTokenCacheStub struct {
	token string
}

func (s *accountModelImportGeminiTokenCacheStub) GetAccessToken(ctx context.Context, cacheKey string) (string, error) {
	return s.token, nil
}

func (s *accountModelImportGeminiTokenCacheStub) SetAccessToken(ctx context.Context, cacheKey string, token string, ttl time.Duration) error {
	return nil
}

func (s *accountModelImportGeminiTokenCacheStub) DeleteAccessToken(ctx context.Context, cacheKey string) error {
	return nil
}

func (s *accountModelImportGeminiTokenCacheStub) AcquireRefreshLock(ctx context.Context, cacheKey string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (s *accountModelImportGeminiTokenCacheStub) ReleaseRefreshLock(ctx context.Context, cacheKey string) error {
	return nil
}

func newTestGeminiCompatService(upstream HTTPUpstream) *GeminiMessagesCompatService {
	return newTestGeminiCompatServiceWithToken(upstream, "")
}

func newTestGeminiCompatServiceWithToken(upstream HTTPUpstream, accessToken string) *GeminiMessagesCompatService {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	svc := &GeminiMessagesCompatService{
		httpUpstream: upstream,
		cfg:          cfg,
	}
	if strings.TrimSpace(accessToken) != "" {
		svc.tokenProvider = &GeminiTokenProvider{tokenCache: &accountModelImportGeminiTokenCacheStub{token: accessToken}}
	}
	return svc
}

func TestImportAccountModels_ImportsAndDeduplicatesOpenAIModels(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"gpt-test-model-a"},{"id":"gpt-test-model-a"},{"id":" gpt-test-model-b "},{"id":""}]}`,
	}
	svc := NewAccountModelImportService(catalogService, nil, upstream, nil)
	account := &Account{
		ID:       101,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "token-1",
			"base_url": "https://example.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-test-model-a", "gpt-test-model-b"}, result.DetectedModels)
	require.Equal(t, []string{"gpt-test-model-a", "gpt-test-model-b"}, result.ImportedModels)
	require.Equal(t, 2, result.ImportedCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Empty(t, result.FailedModels)
	require.Equal(t, 2, countImportModelResults(result.ModelResults, "imported"))
	require.Equal(t, 1, countImportModelResults(result.ModelResults, "skipped"))
	requireImportModelReason(t, result.ModelResults, "gpt-test-model-a", "imported", "imported_new")
	requireImportModelReason(t, result.ModelResults, "gpt-test-model-b", "imported", "imported_new")
	requireImportModelReason(t, result.ModelResults, "gpt-test-model-a", "skipped", "duplicate_canonical")
	requireImportModelRegistry(t, result.ModelResults, "gpt-test-model-a", "imported", "gpt-test-model-a")
	requireImportModelRegistry(t, result.ModelResults, "gpt-test-model-b", "imported", "gpt-test-model-b")
	requireImportModelRegistry(t, result.ModelResults, "gpt-test-model-a", "skipped", "gpt-test-model-a")

	result, err = svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Zero(t, result.ImportedCount)
	require.Equal(t, 3, countImportModelResults(result.ModelResults, "skipped"))
	requireImportModelReason(t, result.ModelResults, "gpt-test-model-a", "skipped", "already_exists")
	requireImportModelReason(t, result.ModelResults, "gpt-test-model-b", "skipped", "already_exists")
	requireImportModelRegistry(t, result.ModelResults, "gpt-test-model-a", "skipped", "gpt-test-model-a")
	requireImportModelRegistry(t, result.ModelResults, "gpt-test-model-b", "skipped", "gpt-test-model-b")

	stored := repo.values[SettingKeyModelRegistryEntries]
	require.NotEmpty(t, stored)
	ids := registryEntryIDsFromJSON(t, stored)
	require.Contains(t, ids, "gpt-test-model-a")
	require.Contains(t, ids, "gpt-test-model-b")
}

func TestImportAccountModels_ContinuesOnCatalogUpsertFailure(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	repo.failContains = "gpt-test-model-b"
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"gpt-test-model-a"},{"id":"gpt-test-model-b"}]}`,
	}
	svc := NewAccountModelImportService(catalogService, nil, upstream, nil)
	account := &Account{
		ID:       102,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "token-2",
			"base_url": "https://example.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Equal(t, 1, result.ImportedCount)
	require.Equal(t, []string{"gpt-test-model-a"}, result.ImportedModels)
	require.Len(t, result.FailedModels, 1)
	require.Equal(t, "gpt-test-model-b", result.FailedModels[0].Model)
	require.Contains(t, result.FailedModels[0].Error, "persist")
	require.Equal(t, 1, countImportModelResults(result.ModelResults, "imported"))
	require.Equal(t, 1, countImportModelResults(result.ModelResults, "failed"))
	requireImportModelReason(t, result.ModelResults, "gpt-test-model-b", "failed", "persist_failed")
}

func TestImportAccountModels_ReturnsClearErrorForUnsupportedSoraOAuth(t *testing.T) {
	svc := NewAccountModelImportService(nil, nil, nil, nil)
	account := &Account{
		ID:       103,
		Platform: PlatformSora,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"access_token": "tok",
		},
	}

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusBadRequest), appErr.Code)
	require.Equal(t, "current Sora OAuth account type does not support real model probing", appErr.Message)
}

func TestImportAccountModels_RequiresCatalogService(t *testing.T) {
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"gpt-test-model-a"}]}`,
	}
	svc := NewAccountModelImportService(nil, nil, upstream, nil)
	account := &Account{
		ID:       105,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "token-4",
			"base_url": "https://example.test",
		},
	}

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusInternalServerError), appErr.Code)
	require.Equal(t, "model catalog service is unavailable", appErr.Message)
}

func TestImportAccountModels_ReturnsClearErrorForUnauthorizedUpstream(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusUnauthorized,
		body:       `{"error":"invalid_api_key"}`,
	}
	svc := NewAccountModelImportService(catalogService, nil, upstream, nil)
	account := &Account{
		ID:       104,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "token-3",
			"base_url": "https://example.test",
		},
	}

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusBadRequest), appErr.Code)
	require.Contains(t, appErr.Message, "status 401")
	require.NotEqual(t, infraerrors.UnknownMessage, appErr.Message)
}

func TestImportAccountModels_ImportsAnthropicModelsWithExpectedHeaders(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"claude-test-model-a"},{"id":"claude-test-model-b"}]}`,
	}
	svc := NewAccountModelImportService(catalogService, nil, upstream, nil)
	account := &Account{
		ID:       106,
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "anthropic-key",
			"base_url": "https://anthropic.example.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Equal(t, []string{"claude-test-model-a", "claude-test-model-b"}, result.DetectedModels)
	require.Equal(t, 2, result.ImportedCount)
	require.Equal(t, 2, countImportModelResults(result.ModelResults, "imported"))
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://anthropic.example.test/v1/models", upstream.lastReq.URL.String())
	require.Equal(t, "anthropic-key", upstream.lastReq.Header.Get("x-api-key"))
	require.Equal(t, "2023-06-01", upstream.lastReq.Header.Get("anthropic-version"))
	require.Equal(t, claude.APIKeyBetaHeader, upstream.lastReq.Header.Get("anthropic-beta"))
	require.Equal(t, claude.DefaultHeaders["User-Agent"], upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, claude.DefaultHeaders["X-Stainless-Lang"], upstream.lastReq.Header.Get("X-Stainless-Lang"))
	require.False(t, upstream.lastEnableTLSFingerprint)

	stored := repo.values[SettingKeyModelRegistryEntries]
	require.Contains(t, stored, "claude-test-model-a")
	require.Contains(t, stored, "claude-test-model-b")
}

func TestImportAccountModels_ImportsKiroModelsFromBuiltinCatalog(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusUnauthorized,
		body:       `{"error":"should not be called"}`,
	}
	svc := NewAccountModelImportService(catalogService, nil, upstream, nil)
	account := &Account{
		ID:       114,
		Platform: PlatformKiro,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"access_token": "kiro-token",
			"profile_arn":  "arn:aws:codewhisperer:us-east-1:123456789012:profile/test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Nil(t, upstream.lastReq)
	expectedDetected, _ := normalizeImportedModelIDs(KiroBuiltinModelIDs())
	require.Equal(t, expectedDetected, result.DetectedModels)
	require.Equal(t, KiroBuiltinCatalogSource, result.ProbeSource)
	require.Equal(t, KiroBuiltinCatalogNotice, result.ProbeNotice)
	require.GreaterOrEqual(t, len(result.ModelResults), len(result.DetectedModels))
	require.Equal(t, len(result.DetectedModels), countImportModelResults(result.ModelResults, "imported"))
}

func TestImportAccountModels_ImportsGeminiModelsFromAIStudioListing(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"models":[{"name":"models/gemini-test-model-a"},{"name":"gemini-test-model-b"}]}`,
	}
	geminiCompatService := newTestGeminiCompatService(upstream)
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	account := &Account{
		ID:       107,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "gemini-key",
			"base_url": "http://gemini.local.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Equal(t, []string{"gemini-test-model-a", "gemini-test-model-b"}, result.DetectedModels)
	require.Equal(t, 2, result.ImportedCount)
	require.Equal(t, 2, countImportModelResults(result.ModelResults, "imported"))
	require.Equal(t, accountModelProbeSourceUpstream, result.ProbeSource)
	require.Empty(t, result.ProbeNotice)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "http://gemini.local.test/v1beta/models", upstream.lastReq.URL.String())
	require.Equal(t, "gemini-key", upstream.lastReq.Header.Get("x-goog-api-key"))
	require.Equal(t, int64(107), upstream.lastAccountID)

	stored := repo.values[SettingKeyModelRegistryEntries]
	require.Contains(t, stored, "gemini-test-model-a")
	require.Contains(t, stored, "gemini-test-model-b")
}

func TestProbeAccountModels_UsesGeminiVertexPublisherListing(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"models":[{"name":"projects/vertex-project/locations/us-central1/publishers/google/models/gemini-2.5-pro"},{"name":"publishers/google/models/gemini-2.0-flash-lite"}]}`,
	}
	geminiCompatService := newTestGeminiCompatServiceWithToken(upstream, "vertex-access-token")
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	account := &Account{
		ID:       115,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type":        "vertex_ai",
			"vertex_project_id": "vertex-project",
			"vertex_location":   "us-central1",
		},
	}

	result, err := svc.ProbeAccountModels(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, []string{"gemini-2.5-pro", "gemini-2.0-flash-lite"}, result.DetectedModels)
	require.Len(t, result.Models, 2)
	require.Equal(t, accountModelProbeSourceUpstream, result.ProbeSource)
	require.NotNil(t, upstream.lastReq)
	require.Equal(
		t,
		"https://us-central1-aiplatform.googleapis.com/v1/projects/vertex-project/locations/us-central1/publishers/google/models",
		upstream.lastReq.URL.String(),
	)
	require.Equal(t, "Bearer vertex-access-token", upstream.lastReq.Header.Get("Authorization"))
	require.Empty(t, upstream.lastReq.Header.Get("x-goog-api-key"))
}

func TestProbeAccountModels_GeminiVertexRequiresProjectLocationAndToken(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"models":[{"name":"projects/x/locations/us-central1/publishers/google/models/gemini-2.5-pro"}]}`,
	}

	t.Run("missing project", func(t *testing.T) {
		geminiCompatService := newTestGeminiCompatServiceWithToken(upstream, "vertex-access-token")
		svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
		account := &Account{
			ID:       116,
			Platform: PlatformGemini,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"oauth_type":      "vertex_ai",
				"vertex_location": "us-central1",
			},
		}

		_, err := svc.ProbeAccountModels(context.Background(), account)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing vertex_project_id")
	})

	t.Run("missing location", func(t *testing.T) {
		geminiCompatService := newTestGeminiCompatServiceWithToken(upstream, "vertex-access-token")
		svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
		account := &Account{
			ID:       117,
			Platform: PlatformGemini,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"oauth_type":        "vertex_ai",
				"vertex_project_id": "vertex-project",
			},
		}

		_, err := svc.ProbeAccountModels(context.Background(), account)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing vertex_location")
	})

	t.Run("missing token", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.URLAllowlist.AllowInsecureHTTP = true
		geminiCompatService := &GeminiMessagesCompatService{
			httpUpstream: upstream,
			cfg:          cfg,
			tokenProvider: &GeminiTokenProvider{
				tokenCache: &accountModelImportGeminiTokenCacheStub{},
			},
		}
		svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
		account := &Account{
			ID:       118,
			Platform: PlatformGemini,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"oauth_type":        "vertex_ai",
				"vertex_project_id": "vertex-project",
				"vertex_location":   "us-central1",
			},
		}

		_, err := svc.ProbeAccountModels(context.Background(), account)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vertex ai credentials missing service account JSON and legacy access_token")
	})
}

func TestImportAccountModels_ImportsGeminiCodeAssistFallbackModelsOnInsufficientScope(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":{"status":"PERMISSION_DENIED","message":"Request had insufficient authentication scopes.","details":[{"reason":"ACCESS_TOKEN_SCOPE_INSUFFICIENT"}]}}`,
		headers: http.Header{
			"Www-Authenticate": []string{`Bearer error="insufficient_scope"`},
		},
	}
	geminiCompatService := newTestGeminiCompatServiceWithToken(upstream, "oauth-token")
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	account := &Account{
		ID:       110,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type": "code_assist",
			"project_id": "project-123",
			"base_url":   "http://gemini.local.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "http://gemini.local.test/v1beta/models", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, accountModelProbeSourceGeminiCLIDefaultFallback, result.ProbeSource)
	require.NotEmpty(t, result.ProbeNotice)

	expected := expectedNormalizedGeminiCLIDefaultModelIDs()
	require.Equal(t, expected, result.DetectedModels)
	require.GreaterOrEqual(t, result.ImportedCount+result.SkippedCount, len(expected))
	require.Equal(t, len(expected), len(result.ModelResults))
	require.GreaterOrEqual(t, countImportModelResults(result.ModelResults, "imported")+countImportModelResults(result.ModelResults, "merged")+countImportModelResults(result.ModelResults, "skipped"), len(expected))

	// 默认 fallback 模型可能已在 seed registry 中，无需写入 runtime entries。
	// 若写入发生，也应保证 JSON 可解析。
	if stored := strings.TrimSpace(repo.values[SettingKeyModelRegistryEntries]); stored != "" {
		var entries []any
		require.NoError(t, json.Unmarshal([]byte(stored), &entries))
	}
}

func TestImportAccountModels_ImportsLegacyGeminiOAuthFallbackModelsOnInsufficientScope(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":{"status":"PERMISSION_DENIED","message":"Request had insufficient authentication scopes.","details":[{"reason":"ACCESS_TOKEN_SCOPE_INSUFFICIENT"}]}}`,
		headers: http.Header{
			"Www-Authenticate": []string{`Bearer error="insufficient_scope"`},
		},
	}
	geminiCompatService := newTestGeminiCompatServiceWithToken(upstream, "oauth-token")
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	account := &Account{
		ID:       113,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"project_id": "project-legacy-cli",
			"base_url":   "http://gemini.local.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, accountModelProbeSourceGeminiCLIDefaultFallback, result.ProbeSource)
	require.Equal(t, expectedNormalizedGeminiCLIDefaultModelIDs(), result.DetectedModels)
	require.NotEmpty(t, result.ProbeNotice)

	// 默认 fallback 模型可能已在 seed registry 中，无需写入 runtime entries。
	// 若写入发生，也应保证 JSON 可解析。
	if stored := strings.TrimSpace(repo.values[SettingKeyModelRegistryEntries]); stored != "" {
		var entries []any
		require.NoError(t, json.Unmarshal([]byte(stored), &entries))
	}
}

func TestImportAccountModels_GeminiAPIKey403DoesNotFallbackToDefaultModels(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":{"message":"Request had insufficient authentication scopes."}}`,
		headers: http.Header{
			"Www-Authenticate": []string{`Bearer error="insufficient_scope"`},
		},
	}
	geminiCompatService := newTestGeminiCompatService(upstream)
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	account := &Account{
		ID:       111,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "gemini-key",
			"base_url": "http://gemini.local.test",
		},
	}

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "gemini-key", upstream.lastReq.Header.Get("x-goog-api-key"))

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusBadRequest), appErr.Code)
	require.Contains(t, appErr.Message, "status 403")
	require.Empty(t, repo.values[SettingKeyModelRegistryEntries])
}

func TestImportAccountModels_GeminiCodeAssistNonScope403StillFails(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":{"message":"access denied by upstream policy"}}`,
	}
	geminiCompatService := newTestGeminiCompatServiceWithToken(upstream, "oauth-token")
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	account := &Account{
		ID:       112,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type": "code_assist",
			"project_id": "project-123",
			"base_url":   "http://gemini.local.test",
		},
	}

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusBadRequest), appErr.Code)
	require.Contains(t, appErr.Message, "status 403")
	require.Contains(t, appErr.Message, "access denied by upstream policy")
	require.Empty(t, repo.values[SettingKeyModelRegistryEntries])
}

func TestImportAccountModels_ImportsAntigravityOAuthModels(t *testing.T) {
	originalBaseURLs := append([]string(nil), antigravity.BaseURLs...)
	originalBaseURL := antigravity.BaseURL
	originalAvailability := antigravity.DefaultURLAvailability
	antigravity.DefaultURLAvailability = antigravity.NewURLAvailability(antigravity.URLAvailabilityTTL)
	defer func() {
		antigravity.BaseURLs = originalBaseURLs
		antigravity.BaseURL = originalBaseURL
		antigravity.DefaultURLAvailability = originalAvailability
	}()

	var lastAuthorization string
	var lastUserAgent string
	var lastProject string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1internal:fetchAvailableModels", r.URL.Path)
		lastAuthorization = r.Header.Get("Authorization")
		lastUserAgent = r.Header.Get("User-Agent")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()
		var req antigravity.FetchAvailableModelsRequest
		require.NoError(t, json.Unmarshal(body, &req))
		lastProject = req.Project
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":{"claude-sonnet-4-5-20250929":{},"claude-sonnet-4-5":{},"gemini-test-model-a":{}}}`))
	}))
	defer server.Close()
	antigravity.BaseURLs = []string{server.URL}
	antigravity.BaseURL = server.URL

	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	svc := NewAccountModelImportService(catalogService, nil, nil, nil)
	account := &Account{
		ID:       108,
		Platform: PlatformAntigravity,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"access_token": "antigravity-token",
			"project_id":   "project-123",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Equal(t, []string{"claude-sonnet-4.5", "gemini-test-model-a"}, result.DetectedModels)
	require.Equal(t, 2, result.ImportedCount)
	require.Equal(t, 1, countImportModelResults(result.ModelResults, "merged"))
	requireImportModelRegistry(t, result.ModelResults, "claude-sonnet-4-5", "merged", "claude-sonnet-4-5")
	requireImportModelRegistry(t, result.ModelResults, "gemini-test-model-a", "imported", "gemini-test-model-a")
	require.Equal(t, "Bearer antigravity-token", lastAuthorization)
	require.Equal(t, antigravity.GetUserAgent(), lastUserAgent)
	require.Equal(t, "project-123", lastProject)

	result, err = svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Zero(t, result.ImportedCount)

	stored := repo.values[SettingKeyModelRegistryEntries]
	require.Contains(t, stored, "claude-sonnet-4-5")
	require.Contains(t, stored, "gemini-test-model-a")
}

func TestImportAccountModels_AntigravityAPIKeyDelegatesToAnthropicProbe(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"claude-test-model-from-antigravity"}]}`,
	}
	svc := NewAccountModelImportService(catalogService, nil, upstream, nil)
	account := &Account{
		ID:       109,
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "antigravity-api-key",
			"base_url": "https://antigravity-compatible.example.test",
		},
	}

	result, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.NoError(t, err)
	require.Equal(t, []string{"claude-test-model-from-antigravity"}, result.DetectedModels)
	require.Equal(t, 1, result.ImportedCount)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://antigravity-compatible.example.test/antigravity/v1/models", upstream.lastReq.URL.String())
	require.Equal(t, "antigravity-api-key", upstream.lastReq.Header.Get("x-api-key"))
	require.Equal(t, claude.APIKeyBetaHeader, upstream.lastReq.Header.Get("anthropic-beta"))
	require.Equal(t, "2023-06-01", upstream.lastReq.Header.Get("anthropic-version"))
}

func countImportModelResults(results []AccountModelImportModelResult, status string) int {
	count := 0
	for _, result := range results {
		if result.Status == status {
			count++
		}
	}
	return count
}

func requireImportModelReason(t *testing.T, results []AccountModelImportModelResult, sourceModel string, status string, reason string) {
	t.Helper()
	for _, result := range results {
		if result.SourceModel == sourceModel && result.Status == status && result.ReasonCode == reason {
			return
		}
	}
	t.Fatalf("expected model result %q with status=%q reason=%q, got %#v", sourceModel, status, reason, results)
}

func requireImportModelRegistry(t *testing.T, results []AccountModelImportModelResult, sourceModel string, status string, registryModel string) {
	t.Helper()
	for _, result := range results {
		if result.SourceModel == sourceModel && result.Status == status && result.RegistryModel == registryModel {
			return
		}
	}
	t.Fatalf("expected model result %q with status=%q registry_model=%q, got %#v", sourceModel, status, registryModel, results)
}

func registryEntryIDsFromJSON(t *testing.T, payload string) []string {
	t.Helper()
	type registryEntry struct {
		ID string `json:"id"`
	}
	var entries []registryEntry
	require.NoError(t, json.Unmarshal([]byte(payload), &entries))
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}
