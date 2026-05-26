package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newDocumentAIHandlerContext(method, target string, body []byte, contentType string, apiKey *service.APIKey) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx.Request = req
	if apiKey != nil {
		ctx.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
	}
	return ctx, rec
}

func decodeResponseEnvelope(t *testing.T, rec *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func decodeDocumentAIResponseData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	resp := decodeResponseEnvelope(t, rec)
	payload, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(payload, &out))
	return out
}

type documentAIHandlerSettingRepoStub struct {
	enabled bool
}

func (s *documentAIHandlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *documentAIHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key != service.SettingKeyDocumentAIEnabled {
		return "", service.ErrSettingNotFound
	}
	if s.enabled {
		return "true", nil
	}
	return "false", nil
}

func (s *documentAIHandlerSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *documentAIHandlerSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *documentAIHandlerSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *documentAIHandlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *documentAIHandlerSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type documentAIHandlerRepoStub struct {
	job                    *service.DocumentAIJob
	createCalls            int
	updateAfterSubmitCalls int
}

func (r *documentAIHandlerRepoStub) Create(_ context.Context, job *service.DocumentAIJob) error {
	r.createCalls++
	if job == nil {
		return nil
	}
	cloned := *job
	r.job = &cloned
	return nil
}

func (r *documentAIHandlerRepoStub) GetByJobIDForUser(_ context.Context, jobID string, userID int64) (*service.DocumentAIJob, error) {
	if r.job == nil || r.job.JobID != jobID || r.job.UserID != userID {
		return nil, sql.ErrNoRows
	}
	cloned := *r.job
	return &cloned, nil
}

func (r *documentAIHandlerRepoStub) UpdateAfterSubmit(_ context.Context, jobID string, providerJobID *string, providerBatchID *string, status string, providerRaw *string) error {
	r.updateAfterSubmitCalls++
	if r.job == nil || r.job.JobID != jobID {
		return sql.ErrNoRows
	}
	r.job.ProviderJobID = providerJobID
	r.job.ProviderBatchID = providerBatchID
	r.job.Status = status
	r.job.ProviderResultJSON = providerRaw
	return nil
}

func (r *documentAIHandlerRepoStub) ListPollable(context.Context, int) ([]service.DocumentAIJob, error) {
	panic("unexpected ListPollable call")
}

func (r *documentAIHandlerRepoStub) MarkRunning(context.Context, string, *string) error {
	panic("unexpected MarkRunning call")
}

func (r *documentAIHandlerRepoStub) MarkSucceeded(context.Context, string, *string, *string) error {
	panic("unexpected MarkSucceeded call")
}

func (r *documentAIHandlerRepoStub) MarkFailed(context.Context, string, *string, string, string) error {
	panic("unexpected MarkFailed call")
}

func (r *documentAIHandlerRepoStub) TouchLastPolledAt(context.Context, string) error {
	panic("unexpected TouchLastPolledAt call")
}

func newEnabledDocumentAIHandler(repo service.DocumentAIJobRepository) *DocumentAIHandler {
	settingService := service.NewSettingService(&documentAIHandlerSettingRepoStub{enabled: true}, &config.Config{})
	documentAIService := service.NewDocumentAIService(repo, nil, nil, nil, nil, settingService, nil)
	return NewDocumentAIHandler(documentAIService)
}

type documentAIHandlerUserRepoStub struct {
	service.UserRepository
	user *service.User
}

func (s *documentAIHandlerUserRepoStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	if s.user != nil {
		return s.user, nil
	}
	return &service.User{ID: id, Balance: 1, Status: service.StatusActive}, nil
}

type documentAIHandlerHoldRepoStub struct {
	reserveCalls int
	releaseCalls int
	reserveErr   error
	hold         *service.BillingHold
}

func (s *documentAIHandlerHoldRepoStub) Reserve(_ context.Context, hold *service.BillingHold) (*service.BillingHold, error) {
	s.reserveCalls++
	if s.reserveErr != nil {
		return nil, s.reserveErr
	}
	if hold == nil {
		return nil, service.ErrInvalidBillingAmount
	}
	cloned := *hold
	cloned.Status = service.BillingHoldStatusHeld
	s.hold = &cloned
	return &cloned, nil
}

func (s *documentAIHandlerHoldRepoStub) Settle(context.Context, string, int64, float64) (*service.BillingHold, error) {
	return nil, service.ErrBillingHoldNotFound
}

func (s *documentAIHandlerHoldRepoStub) Release(_ context.Context, requestID string, apiKeyID int64) (*service.BillingHold, error) {
	s.releaseCalls++
	if s.hold == nil || s.hold.RequestID != requestID || s.hold.APIKeyID != apiKeyID {
		return nil, service.ErrBillingHoldNotFound
	}
	released := *s.hold
	released.Status = service.BillingHoldStatusReleased
	s.hold = &released
	return &released, nil
}

type documentAIHandlerAccountRepoStub struct {
	service.AccountRepository
	accounts []service.Account
}

func (s *documentAIHandlerAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]service.Account, error) {
	if groupID != 9 || platform != service.PlatformBaiduDocumentAI {
		return nil, nil
	}
	return append([]service.Account(nil), s.accounts...), nil
}

type documentAIHandlerHTTPUpstreamFunc func(*http.Request, string, int64, int) (*http.Response, error)

func (f documentAIHandlerHTTPUpstreamFunc) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return f(req, proxyURL, accountID, accountConcurrency)
}

func (f documentAIHandlerHTTPUpstreamFunc) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return f(req, proxyURL, accountID, accountConcurrency)
}

type documentAIHandlerAPIKeyRepoStub struct {
	service.APIKeyRepository
	holdRepo *documentAIHandlerHoldRepoStub
	keys     []string
}

func (s *documentAIHandlerAPIKeyRepoStub) BillingHoldRepository() service.BillingHoldRepository {
	return s.holdRepo
}

func (s *documentAIHandlerAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*service.APIKeyRateLimitData, error) {
	return &service.APIKeyRateLimitData{}, nil
}

func (s *documentAIHandlerAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	return nil
}

func (s *documentAIHandlerAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	return append([]string(nil), s.keys...), nil
}

func (s *documentAIHandlerAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	return nil, nil
}

func newDocumentAIBillingHandler(
	documentAIService *service.DocumentAIService,
	user *service.User,
	holdRepo *documentAIHandlerHoldRepoStub,
	cfg *config.Config,
) *DocumentAIHandler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	apiKeyRepo := &documentAIHandlerAPIKeyRepoStub{holdRepo: holdRepo, keys: []string{"sk-doc"}}
	billingCacheService := service.NewBillingCacheService(nil, &documentAIHandlerUserRepoStub{user: user}, nil, apiKeyRepo, cfg)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, cfg)
	apiKeyService.SetBillingCacheService(billingCacheService)
	return ProvideDocumentAIHandler(documentAIService, billingCacheService, apiKeyService, nil)
}

func newDocumentAITestAPIKey(balance float64) *service.APIKey {
	return &service.APIKey{
		ID:     1,
		Key:    "sk-doc",
		UserID: 2,
		User:   &service.User{ID: 2, Balance: balance, Status: service.StatusActive},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI, Status: service.StatusActive, Hydrated: true}},
		},
	}
}

func TestDocumentAIHandlerBuildSubmitInputJSONFileURL(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{ID: 1, User: &service.User{ID: 2}}
	ctx, _ := newDocumentAIHandlerContext(
		http.MethodPost,
		"/document-ai/v1/jobs",
		[]byte(`{"model":"pp-ocrv5-server","file_url":"https://example.com/doc.pdf","options":{"useChartRecognition":true}}`),
		"application/json",
		apiKey,
	)

	input, err := handler.buildSubmitInput(ctx, apiKey, 9)
	require.NoError(t, err)
	require.Equal(t, int64(9), input.GroupID)
	require.Equal(t, "pp-ocrv5-server", input.Model)
	require.Equal(t, service.DocumentAISourceTypeFileURL, input.SourceType)
	require.Equal(t, "https://example.com/doc.pdf", input.FileURL)
	require.Equal(t, true, input.Options["useChartRecognition"])
}

func TestDocumentAIHandlerBuildSubmitInputMultipart(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{ID: 1, User: &service.User{ID: 2}}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "pp-structurev3"))
	require.NoError(t, writer.WriteField("options", `{"useLayoutDetection":true}`))
	fileWriter, err := writer.CreateFormFile("file", "sample.pdf")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("pdf"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx, _ := newDocumentAIHandlerContext(http.MethodPost, "/document-ai/v1/jobs", body.Bytes(), writer.FormDataContentType(), apiKey)
	input, err := handler.buildSubmitInput(ctx, apiKey, 9)
	require.NoError(t, err)
	require.Equal(t, service.DocumentAISourceTypeFile, input.SourceType)
	require.Equal(t, "sample.pdf", input.FileName)
	require.Equal(t, int64(3), input.FileSize)
	require.Equal(t, true, input.Options["useLayoutDetection"])
}

func TestDocumentAIHandlerBuildSubmitInputMultipartRejectsOversizedFile(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.DocumentAIUploadMaxBytes = 2
	handler := NewDocumentAIHandler(service.NewDocumentAIService(nil, nil, nil, nil, nil, nil, cfg))
	apiKey := &service.APIKey{ID: 1, User: &service.User{ID: 2}}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "pp-structurev3"))
	fileWriter, err := writer.CreateFormFile("file", "sample.pdf")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("pdf"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx, _ := newDocumentAIHandlerContext(http.MethodPost, "/document-ai/v1/jobs", body.Bytes(), writer.FormDataContentType(), apiKey)
	_, err = handler.buildSubmitInput(ctx, apiKey, 9)

	require.Error(t, err)
	require.Equal(t, "document_ai_invalid_request", infraerrors.Reason(err))
}

func TestDocumentAIHandlerBuildDirectInputJSONBase64(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{ID: 1, User: &service.User{ID: 2}}
	encoded := base64.StdEncoding.EncodeToString([]byte("png"))
	ctx, _ := newDocumentAIHandlerContext(
		http.MethodPost,
		"/document-ai/v1/models/pp-ocrv5-server:parse",
		[]byte(`{"file_base64":"`+encoded+`","file_type":"image","options":{"useDocUnwarping":true}}`),
		"application/json",
		apiKey,
	)

	input, err := handler.buildDirectInput(ctx, apiKey, 9, service.DocumentAIModelPPOCRV5Server)
	require.NoError(t, err)
	require.Equal(t, service.DocumentAISourceTypeFileBase64, input.SourceType)
	require.Equal(t, service.DocumentAIFileTypeImage, input.FileType)
	require.Equal(t, int64(0), input.FileSize)
	require.Empty(t, input.FileBytes)
	require.Equal(t, encoded, input.FileBase64)
	require.Equal(t, true, input.Options["useDocUnwarping"])
}

func TestDocumentAIHandlerCreateJobDisabledReturns503(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{
		ID:   1,
		User: &service.User{ID: 2},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI}},
		},
	}
	ctx, rec := newDocumentAIHandlerContext(
		http.MethodPost,
		"/document-ai/v1/jobs",
		[]byte(`{"model":"pp-ocrv5-server","file_url":"https://example.com/doc.pdf"}`),
		"application/json",
		apiKey,
	)

	handler.CreateJob(ctx)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	resp := decodeResponseEnvelope(t, rec)
	require.Equal(t, "document_ai_disabled", resp.Reason)
}

func TestDocumentAIHandlerCreateJobRequiresBaiduGroupBinding(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{ID: 1, User: &service.User{ID: 2}}
	ctx, rec := newDocumentAIHandlerContext(http.MethodPost, "/document-ai/v1/jobs", []byte(`{}`), "application/json", apiKey)

	handler.CreateJob(ctx)

	require.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeResponseEnvelope(t, rec)
	require.Contains(t, resp.Message, "Baidu Document AI group")
}

func TestDocumentAIHandlerCreateJobRejectsInsufficientBalanceBeforeService(t *testing.T) {
	holdRepo := &documentAIHandlerHoldRepoStub{reserveErr: service.ErrInsufficientBalance}
	jobRepo := &documentAIHandlerRepoStub{}
	settingService := service.NewSettingService(&documentAIHandlerSettingRepoStub{enabled: true}, &config.Config{})
	documentAIService := service.NewDocumentAIService(jobRepo, nil, nil, nil, nil, settingService, nil)
	apiKey := newDocumentAITestAPIKey(0.01)
	handler := newDocumentAIBillingHandler(documentAIService, apiKey.User, holdRepo, &config.Config{})
	ctx, rec := newDocumentAIHandlerContext(
		http.MethodPost,
		"/document-ai/v1/jobs",
		[]byte(`{"model":"pp-ocrv5-server","file_url":"https://example.com/doc.pdf"}`),
		"application/json",
		apiKey,
	)

	handler.CreateJob(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponseEnvelope(t, rec)
	require.Equal(t, "INSUFFICIENT_BALANCE", resp.Reason)
	require.Equal(t, 1, holdRepo.reserveCalls)
	require.Equal(t, 0, holdRepo.releaseCalls)
	require.Equal(t, 0, jobRepo.createCalls)
}

func TestDocumentAIHandlerCreateJobReleasesHoldAfterServiceFailure(t *testing.T) {
	holdRepo := &documentAIHandlerHoldRepoStub{}
	handler := newDocumentAIBillingHandler(&service.DocumentAIService{}, &service.User{ID: 2, Balance: 1, Status: service.StatusActive}, holdRepo, &config.Config{})
	apiKey := newDocumentAITestAPIKey(1)
	ctx, rec := newDocumentAIHandlerContext(
		http.MethodPost,
		"/document-ai/v1/jobs",
		[]byte(`{"model":"pp-ocrv5-server","file_url":"https://example.com/doc.pdf"}`),
		"application/json",
		apiKey,
	)

	handler.CreateJob(ctx)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	resp := decodeResponseEnvelope(t, rec)
	require.Equal(t, "document_ai_disabled", resp.Reason)
	require.Equal(t, 1, holdRepo.reserveCalls)
	require.Equal(t, 1, holdRepo.releaseCalls)
	require.NotNil(t, apiKey.BillingHold)
	require.Equal(t, service.BillingHoldStatusReleased, apiKey.BillingHold.Status)
	require.NotEmpty(t, apiKey.BillingHold.RequestFingerprint)
}

func TestDocumentAIHandlerCreateJobReleasesHoldAfterSuccessfulFreeSubmit(t *testing.T) {
	holdRepo := &documentAIHandlerHoldRepoStub{}
	jobRepo := &documentAIHandlerRepoStub{}
	accountRepo := &documentAIHandlerAccountRepoStub{accounts: []service.Account{
		{
			ID:       7,
			Name:     "doc-ai",
			Platform: service.PlatformBaiduDocumentAI,
			Type:     service.AccountTypeAPIKey,
			Status:   service.StatusActive,
			Credentials: map[string]any{
				"async_bearer_token": "token",
				"async_base_url":     service.DefaultBaiduDocumentAIAsyncBaseURL(),
			},
		},
	}}
	upstream := documentAIHandlerHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, service.DefaultBaiduDocumentAIAsyncBaseURL()+"/jobs", req.URL.String())
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `"fileUrl":"https://example.com/doc.pdf"`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"jobId":"provider-job-1","status":"running"}`)),
		}, nil
	})
	settingService := service.NewSettingService(&documentAIHandlerSettingRepoStub{enabled: true}, &config.Config{})
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	documentAIService := service.NewDocumentAIService(jobRepo, accountRepo, upstream, nil, nil, settingService, cfg)
	apiKey := newDocumentAITestAPIKey(1)
	handler := newDocumentAIBillingHandler(documentAIService, apiKey.User, holdRepo, cfg)
	ctx, rec := newDocumentAIHandlerContext(
		http.MethodPost,
		"/document-ai/v1/jobs",
		[]byte(`{"model":"pp-ocrv5-server","file_url":"https://example.com/doc.pdf"}`),
		"application/json",
		apiKey,
	)

	handler.CreateJob(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, holdRepo.reserveCalls)
	require.Equal(t, 1, holdRepo.releaseCalls)
	require.Equal(t, 1, jobRepo.createCalls)
	require.Equal(t, 1, jobRepo.updateAfterSubmitCalls)
	require.NotNil(t, apiKey.BillingHold)
	require.Equal(t, service.BillingHoldStatusReleased, apiKey.BillingHold.Status)
	require.NotEmpty(t, apiKey.BillingHold.RequestFingerprint)
}

func TestDocumentAIHandlerParseReturns400OnInvalidAction(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{
		ID:   1,
		User: &service.User{ID: 2},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI}},
		},
	}
	ctx, rec := newDocumentAIHandlerContext(http.MethodPost, "/document-ai/v1/models/pp-ocrv5-server", nil, "application/json", apiKey)
	ctx.Params = gin.Params{{Key: "modelAction", Value: "/pp-ocrv5-server"}}

	handler.Parse(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponseEnvelope(t, rec)
	require.Contains(t, resp.Message, "invalid document ai action")
}

func TestDocumentAIHandlerParseReturns404OnUnknownAction(t *testing.T) {
	handler := NewDocumentAIHandler(&service.DocumentAIService{})
	apiKey := &service.APIKey{
		ID:   1,
		User: &service.User{ID: 2},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI}},
		},
	}
	ctx, rec := newDocumentAIHandlerContext(http.MethodPost, "/document-ai/v1/models/pp-ocrv5-server:preview", nil, "application/json", apiKey)
	ctx.Params = gin.Params{{Key: "modelAction", Value: "/pp-ocrv5-server:preview"}}

	handler.Parse(ctx)

	require.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponseEnvelope(t, rec)
	require.Contains(t, resp.Message, "action not found")
}

func TestDocumentAIHandlerListModelsReturnsAvailableModels(t *testing.T) {
	handler := newEnabledDocumentAIHandler(&documentAIHandlerRepoStub{})
	apiKey := &service.APIKey{
		ID:   1,
		User: &service.User{ID: 2},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI}},
		},
	}
	ctx, rec := newDocumentAIHandlerContext(http.MethodGet, "/document-ai/v1/models", nil, "", apiKey)

	handler.ListModels(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	payload := decodeDocumentAIResponseData[struct {
		Provider string                              `json:"provider"`
		GroupID  int64                               `json:"group_id"`
		Models   []service.DocumentAIModelDescriptor `json:"models"`
	}](t, rec)
	require.Equal(t, service.DocumentAIProviderBaidu, payload.Provider)
	require.Equal(t, int64(9), payload.GroupID)
	require.Len(t, payload.Models, 4)
}

func TestDocumentAIHandlerGetJobReturnsSummary(t *testing.T) {
	repo := &documentAIHandlerRepoStub{
		job: &service.DocumentAIJob{
			JobID:  "job-1",
			UserID: 2,
			Mode:   service.DocumentAIJobModeAsync,
			Model:  service.DocumentAIModelPPOCRV5Server,
			Status: service.DocumentAIJobStatusRunning,
		},
	}
	handler := newEnabledDocumentAIHandler(repo)
	apiKey := &service.APIKey{
		ID:   1,
		User: &service.User{ID: 2},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI}},
		},
	}
	ctx, rec := newDocumentAIHandlerContext(http.MethodGet, "/document-ai/v1/jobs/job-1", nil, "", apiKey)
	ctx.Params = gin.Params{{Key: "job_id", Value: "job-1"}}

	handler.GetJob(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	payload := decodeDocumentAIResponseData[map[string]any](t, rec)
	require.Equal(t, "job-1", payload["job_id"])
	require.Equal(t, service.DocumentAIJobStatusRunning, payload["status"])
}

func TestDocumentAIHandlerGetJobResultReturnsPendingResultEnvelope(t *testing.T) {
	repo := &documentAIHandlerRepoStub{
		job: &service.DocumentAIJob{
			JobID:  "job-2",
			UserID: 2,
			Mode:   service.DocumentAIJobModeAsync,
			Model:  service.DocumentAIModelPPStructureV3,
			Status: service.DocumentAIJobStatusPending,
		},
	}
	handler := newEnabledDocumentAIHandler(repo)
	apiKey := &service.APIKey{
		ID:   1,
		User: &service.User{ID: 2},
		GroupBindings: []service.APIKeyGroupBinding{
			{GroupID: 9, Group: &service.Group{ID: 9, Platform: service.PlatformBaiduDocumentAI}},
		},
	}
	ctx, rec := newDocumentAIHandlerContext(http.MethodGet, "/document-ai/v1/jobs/job-2/result", nil, "", apiKey)
	ctx.Params = gin.Params{{Key: "job_id", Value: "job-2"}}

	handler.GetJobResult(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	payload := decodeDocumentAIResponseData[map[string]any](t, rec)
	require.Equal(t, "job-2", payload["job_id"])
	require.Equal(t, service.DocumentAIJobStatusPending, payload["status"])
	require.Nil(t, payload["provider_result"])
	require.Nil(t, payload["normalized_result"])
}
