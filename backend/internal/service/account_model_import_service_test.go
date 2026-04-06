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

	responseSequence []accountModelImportHTTPResponseStep
	sequenceIndex    int

	lastReq                  *http.Request
	lastProxyURL             string
	lastAccountID            int64
	lastAccountConcurrency   int
	lastEnableTLSFingerprint bool
	requestURLs              []string
}

type accountModelImportHTTPResponseStep struct {
	statusCode int
	body       string
	err        error
	headers    http.Header
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
	if req != nil && req.URL != nil {
		s.requestURLs = append(s.requestURLs, req.URL.String())
	}

	statusCode := s.statusCode
	body := s.body
	respErr := s.err
	respHeaders := s.headers

	if len(s.responseSequence) > 0 {
		index := s.sequenceIndex
		if index >= len(s.responseSequence) {
			index = len(s.responseSequence) - 1
		}
		step := s.responseSequence[index]
		s.sequenceIndex++
		statusCode = step.statusCode
		body = step.body
		respErr = step.err
		respHeaders = step.headers
	}

	if respErr != nil {
		return nil, respErr
	}
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	clonedHeaders := make(http.Header)
	for key, values := range respHeaders {
		for _, value := range values {
			clonedHeaders.Add(key, value)
		}
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     clonedHeaders,
		Body:       io.NopCloser(strings.NewReader(body)),
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

func TestImportAccountModels_ReturnsClearErrorForUnsupportedPlatform(t *testing.T) {
	svc := NewAccountModelImportService(nil, nil, nil, nil)
	account := &Account{
		ID:       103,
		Platform: "legacy_removed",
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
	require.Equal(t, "current account platform does not support model import", appErr.Message)
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
	require.Equal(t, int32(http.StatusUnauthorized), appErr.Code)
	require.Equal(t, accountModelImportReasonKindUnauthorized, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyUnauthorized, appErr.Metadata["hint_key"])
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
	require.Equal(t, "Kiro runtime verification is unavailable in this deployment; model list uses built-in candidate catalog", result.ProbeNotice)
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

func TestProbeAccountModels_UsesGeminiVertexServiceAccountCatalogViaCountTokens(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	vertexCatalog := newTestVertexCatalogProvider(&VertexCatalogResult{
		OfficialModels: []VertexCatalogModel{
			{
				ID:                 "gemini-2.5-flash",
				DisplayName:        "Gemini 2.5 Flash",
				UpstreamSource:     vertexCatalogOfficialSource,
				Availability:       vertexCatalogCallableAvailability,
				AvailabilityReason: vertexCatalogCallableReason,
			},
			{
				ID:                 "gemini-2.5-pro",
				DisplayName:        "Gemini 2.5 Pro",
				UpstreamSource:     vertexCatalogOfficialSource,
				Availability:       vertexCatalogFailedAvailability,
				AvailabilityReason: "status 403 PERMISSION_DENIED",
			},
		},
		VerifiedExtras: []VertexCatalogModel{
			{
				ID:                 "gemini-3.1-pro-preview",
				DisplayName:        "Gemini 3.1 Pro Preview",
				UpstreamSource:     vertexCatalogVerifiedExtraSource,
				Availability:       vertexCatalogCallableAvailability,
				AvailabilityReason: vertexCatalogCallableReason,
			},
		},
		CallableUnion: []VertexCatalogModel{
			{
				ID:                 "gemini-2.5-flash",
				DisplayName:        "Gemini 2.5 Flash",
				UpstreamSource:     vertexCatalogOfficialSource,
				Availability:       vertexCatalogCallableAvailability,
				AvailabilityReason: vertexCatalogCallableReason,
			},
			{
				ID:                 "gemini-3.1-pro-preview",
				DisplayName:        "Gemini 3.1 Pro Preview",
				UpstreamSource:     vertexCatalogVerifiedExtraSource,
				Availability:       vertexCatalogCallableAvailability,
				AvailabilityReason: vertexCatalogCallableReason,
			},
		},
	})
	geminiCompatService := newTestGeminiCompatService(nil)
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	svc.SetVertexCatalogService(vertexCatalog)
	account := &Account{
		ID:       115,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type":                  "vertex_ai",
			"vertex_project_id":           "vertex-project",
			"vertex_location":             "us-central1",
			"vertex_service_account_json": `{"type":"service_account","client_email":"svc@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
		},
	}

	result, err := svc.ProbeAccountModels(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, []bool{true}, vertexCatalog.forceRefreshCalls)
	require.Equal(t, accountModelProbeSourceVertexServiceAccountCatalog, result.ProbeSource)
	require.Contains(t, result.ProbeNotice, "official=2")
	require.Contains(t, result.ProbeNotice, "callable=2")
	require.Contains(t, result.ProbeNotice, "verified_extra=1")
	require.Equal(t, []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-3.1-pro-preview"}, result.DetectedModels)

	officialCallable, ok := findProbeModelByID(result.Models, "gemini-2.5-flash")
	require.True(t, ok)
	require.Equal(t, vertexCatalogOfficialSource, officialCallable.UpstreamSource)
	require.Equal(t, vertexCatalogCallableAvailability, officialCallable.Availability)

	officialUncallable, ok := findProbeModelByID(result.Models, "gemini-2.5-pro")
	require.True(t, ok)
	require.Equal(t, vertexCatalogOfficialSource, officialUncallable.UpstreamSource)
	require.Equal(t, vertexCatalogFailedAvailability, officialUncallable.Availability)
	require.Contains(t, officialUncallable.AvailabilityReason, "403")

	verifiedExtra, ok := findProbeModelByID(result.Models, "gemini-3.1-pro-preview")
	require.True(t, ok)
	require.Equal(t, vertexCatalogVerifiedExtraSource, verifiedExtra.UpstreamSource)
	require.Equal(t, vertexCatalogCallableAvailability, verifiedExtra.Availability)
}

func TestProbeAccountModels_UsesGeminiVertexExpressCatalog(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	vertexCatalog := newTestVertexCatalogProvider(&VertexCatalogResult{
		OfficialModels: []VertexCatalogModel{
			{
				ID:                 "gemini-2.0-flash",
				DisplayName:        "Gemini 2.0 Flash",
				UpstreamSource:     vertexCatalogOfficialSource,
				Availability:       vertexCatalogCallableAvailability,
				AvailabilityReason: vertexCatalogCallableReason,
			},
		},
		CallableUnion: []VertexCatalogModel{
			{
				ID:                 "gemini-2.0-flash",
				DisplayName:        "Gemini 2.0 Flash",
				UpstreamSource:     vertexCatalogOfficialSource,
				Availability:       vertexCatalogCallableAvailability,
				AvailabilityReason: vertexCatalogCallableReason,
			},
		},
	})
	geminiCompatService := newTestGeminiCompatService(nil)
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	svc.SetVertexCatalogService(vertexCatalog)
	account := &Account{
		ID:       119,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":            "vertex-express-key",
			"gemini_api_variant": GeminiAPIKeyVariantVertexExpress,
		},
	}

	result, err := svc.ProbeAccountModels(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, []bool{true}, vertexCatalog.forceRefreshCalls)
	require.Equal(t, accountModelProbeSourceVertexExpressCatalog, result.ProbeSource)
	require.Equal(t, []string{"gemini-2.0-flash"}, result.DetectedModels)
	detail, ok := findProbeModelByID(result.Models, "gemini-2.0-flash")
	require.True(t, ok)
	require.Equal(t, vertexCatalogOfficialSource, detail.UpstreamSource)
	require.Equal(t, vertexCatalogCallableAvailability, detail.Availability)
}

func TestProbeAccountModels_VertexCatalogErrorsPropagate(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	catalogService := NewModelCatalogService(repo, nil, nil, nil, nil)
	expectedErr := infraerrors.BadRequest("MODEL_IMPORT_UPSTREAM_FAILED", "vertex official model listing failed with status 403")
	vertexCatalog := &testVertexCatalogProvider{err: expectedErr}
	geminiCompatService := newTestGeminiCompatService(nil)
	svc := NewAccountModelImportService(catalogService, geminiCompatService, nil, nil)
	svc.SetVertexCatalogService(vertexCatalog)
	account := &Account{
		ID:       123,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type":                  "vertex_ai",
			"vertex_project_id":           "vertex-project",
			"vertex_location":             "global",
			"vertex_service_account_json": `{"type":"service_account","client_email":"svc@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
		},
	}

	_, err := svc.ProbeAccountModels(context.Background(), account)
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, []bool{true}, vertexCatalog.forceRefreshCalls)
}

/*
func TestImportAccountModels_GeminiCodeAssistInsufficientScopeReturnsStructuredError(t *testing.T) {
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

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "http://gemini.local.test/v1beta/models", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindGoogleScopeInsufficient, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyGoogleAccessTokenScope, appErr.Metadata["hint_key"])
	require.Equal(t, "gemini", appErr.Metadata["provider"])
	require.Equal(t, "gemini_code_assist_oauth", appErr.Metadata["auth_mode"])
	require.Contains(t, appErr.Message, "status 403")
	require.Empty(t, repo.values[SettingKeyModelRegistryEntries])

	// 默认 fallback 模型可能已在 seed registry 中，无需写入 runtime entries。
	// 若写入发生，也应保证 JSON 可解析。
}

}

func TestImportAccountModels_LegacyGeminiOAuthInsufficientScopeReturnsStructuredError(t *testing.T) {
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

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)
	require.NotNil(t, upstream.lastReq)
	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindGoogleScopeInsufficient, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyGoogleAccessTokenScope, appErr.Metadata["hint_key"])
	require.Equal(t, "gemini", appErr.Metadata["provider"])
	require.Equal(t, "gemini_code_assist_oauth", appErr.Metadata["auth_mode"])
	require.Empty(t, repo.values[SettingKeyModelRegistryEntries])

	// 默认 fallback 模型可能已在 seed registry 中，无需写入 runtime entries。
	// 若写入发生，也应保证 JSON 可解析。
}

}
*/

func TestImportAccountModels_GeminiCodeAssistInsufficientScopeReturnsStructuredError_CurrentBehavior(t *testing.T) {
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

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "http://gemini.local.test/v1beta/models", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindGoogleScopeInsufficient, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyGoogleAccessTokenScope, appErr.Metadata["hint_key"])
	require.Equal(t, "gemini", appErr.Metadata["provider"])
	require.Equal(t, "gemini_code_assist_oauth", appErr.Metadata["auth_mode"])
	require.Contains(t, appErr.Message, "status 403")
	require.Empty(t, repo.values[SettingKeyModelRegistryEntries])
}

func TestImportAccountModels_LegacyGeminiOAuthInsufficientScopeReturnsStructuredError_CurrentBehavior(t *testing.T) {
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

	_, err := svc.ImportAccountModels(context.Background(), account, "manual")
	require.Error(t, err)
	require.NotNil(t, upstream.lastReq)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindGoogleScopeInsufficient, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyGoogleAccessTokenScope, appErr.Metadata["hint_key"])
	require.Equal(t, "gemini", appErr.Metadata["provider"])
	require.Equal(t, "gemini_code_assist_oauth", appErr.Metadata["auth_mode"])
	require.Empty(t, repo.values[SettingKeyModelRegistryEntries])
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
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindGoogleScopeInsufficient, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyGoogleAccessTokenScope, appErr.Metadata["hint_key"])
	require.Equal(t, "api_key", appErr.Metadata["auth_mode"])
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
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindPermissionDenied, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyPermissionDenied, appErr.Metadata["hint_key"])
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
