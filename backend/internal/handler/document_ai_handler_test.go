package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
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
	job *service.DocumentAIJob
}

func (r *documentAIHandlerRepoStub) Create(context.Context, *service.DocumentAIJob) error {
	panic("unexpected Create call")
}

func (r *documentAIHandlerRepoStub) GetByJobIDForUser(_ context.Context, jobID string, userID int64) (*service.DocumentAIJob, error) {
	if r.job == nil || r.job.JobID != jobID || r.job.UserID != userID {
		return nil, sql.ErrNoRows
	}
	cloned := *r.job
	return &cloned, nil
}

func (r *documentAIHandlerRepoStub) UpdateAfterSubmit(context.Context, string, *string, *string, string, *string) error {
	panic("unexpected UpdateAfterSubmit call")
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
	documentAIService := service.NewDocumentAIService(repo, nil, nil, nil, nil, settingService)
	return NewDocumentAIHandler(documentAIService)
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
	require.Equal(t, int64(3), input.FileSize)
	require.Equal(t, []byte("png"), input.FileBytes)
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
